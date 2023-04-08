package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/zenwerk/go-wave"
)

const (
	//sampleRate      = 44100
	sampleRate      = 16000
	channels        = 1
	framesPerBuffer = 64
	seconds         = 5
)

func main() {
	err := portaudio.Initialize()
	if err != nil {
		fmt.Println("Error initializing PortAudio:", err)
		return
	}
	defer portaudio.Terminate()

	inputDevice, err := portaudio.DefaultInputDevice()
	if err != nil {
		fmt.Println("Error getting default input device:", err)
		return
	}

	streamParams := portaudio.LowLatencyParameters(inputDevice, nil)
	streamParams.Input.Channels = channels
	streamParams.SampleRate = sampleRate
	streamParams.FramesPerBuffer = framesPerBuffer

	stream, err := portaudio.OpenStream(streamParams, recordCallback)
	if err != nil {
		fmt.Println("Error opening PortAudio stream:", err)
		return
	}
	defer stream.Close()

	err = stream.Start()
	if err != nil {
		fmt.Println("Error starting stream:", err)
		return
	}

	fmt.Println("Recording for", seconds, "seconds...")
	time.Sleep(time.Duration(seconds) * time.Second)

	err = stream.Stop()
	if err != nil {
		fmt.Println("Error stopping stream:", err)
		return
	}

	saveAudioToFile("output.wav")

	msg, err := exec.Command("whisper.cpp", "./output.wav", "-tr", "-m", "../models/ggml-medium.bin", "-otxt").Output()
	if err != nil {
		fmt.Println("Error running the command:", err)
		fmt.Println(string(msg))
		return
	}
	fmt.Println(string(msg))

	f, err := os.Open("./output.wav.txt")
	if err != nil {
		fmt.Println("Error opening the file:", err)
		return
	}

	c, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println("Error reading the file:", err)
		return
	}

	cmdStr := "tts --text \"" + filter(string(c)) + "\""

	// print to file
	ff, err := os.Create("./output.sh")

	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}

	defer ff.Close()
	ff.WriteString(cmdStr)

	msg, err = exec.Command("sh", "./output.sh").Output()
	if err != nil {
		fmt.Println("Error running the command:", err)
		fmt.Println(string(msg))
		return
	}
	fmt.Println(string(msg))

	err = exec.Command("afplay", "./tts_output.wav").Run()
	if err != nil {
		fmt.Println("Error running the command:", err)
		fmt.Println(string(msg))
		return
	}
}

func filter(s string) string {
	var res []byte
	var filter_mode = false
	for i := 0; i < len(s); i++ {
		if s[i] == '[' || s[i] == '(' {
			filter_mode = true
			continue
		}
		if s[i] == ']' || s[i] == ')' {
			filter_mode = false
			continue
		}

		if filter_mode {
			continue
		}

		res = append(res, s[i])
	}
	return string(res)
}

var recordedData [][]float32

func recordCallback(in, out []float32) {
	data := make([]float32, len(in))
	copy(data, in)
	recordedData = append(recordedData, data)
}

func saveAudioToFile(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	//wavWrite := wave.NewWriter(file, &wave.WriterParam{
	wavWriter, err := wave.NewWriter(wave.WriterParam{
		Out:           file,
		SampleRate:    sampleRate,
		BitsPerSample: 16,
		Channel:       channels,
	})

	if err != nil {
		fmt.Println("Error creating WAV writer:", err)
		return
	}

	for _, data := range recordedData {
		_, err = wavWriter.WriteSample16(sampleToInt16(data))
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	}

	err = wavWriter.Close()
	if err != nil {
		fmt.Println("Error closing the WAV writer:", err)
		return
	}

	fmt.Println("Audio saved to", filename)
}

func sampleToInt16(sample []float32) []int16 {
	var res []int16
	for _, s := range sample {
		res = append(res, int16(s*32767))
	}
	return res
}
