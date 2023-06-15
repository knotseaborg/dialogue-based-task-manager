package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/knotseaborg/dbtm/activity"
	"github.com/knotseaborg/dbtm/gpt"
	"github.com/knotseaborg/dbtm/ui"
	"github.com/knotseaborg/dbtm/voice"
)

func main() {

	if err := os.Mkdir("./temp", os.ModePerm); err != nil {
		fmt.Println(err)
	}
	f, err := os.OpenFile("./temp/log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	startSignal, stopSignal := make(chan []byte, 1), make(chan []byte, 1)
	message := make(chan string)
	toAudioComm, toTextcomm := make(chan string), make(chan string)

	// The stream which manages data flow between the processes and the UI
	uiStream := ui.UIStream{StartSignal: startSignal,
		StopSignal: stopSignal,
		Message:    message,
	}

	go ui.CreateWindow(&uiStream)

	go voice.StartPlaying(toAudioComm, uiStream.StopSignal, uiStream.Message)

	go activity.Main()
	time.Sleep(2 * 1000000000) //Wait for 2 seconds to allow the server to setup

	go func() {
		fmt.Print("Running spakehandler")
		for {
			text := <-toTextcomm
			fmt.Print(text)
			err := gpt.SpakeHandler(toAudioComm, &gpt.Spake{Source: "User", Content: text})
			if err != nil {
				log.Println(err)
			}
		}
	}()

	voice.StartRecording(toTextcomm, uiStream.StartSignal)
}
