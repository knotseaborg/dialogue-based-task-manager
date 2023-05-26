package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/knotseaborg/dbtm/activity"
	"github.com/knotseaborg/dbtm/gpt"
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

	toAudioComm := make(chan string)
	toTextcomm := make(chan string)
	go voice.StartPlaying(toAudioComm)

	go activity.Main()
	time.Sleep(2 * 1000000000) //Wait for 2 seconds to allow the server to setup

	go func() {
		fmt.Print("Running spakehandler")
		for {
			//reader := bufio.NewReader(os.Stdin)
			//fmt.Print("Enter text: ")
			//text, _ := reader.ReadString('\n')
			text := <-toTextcomm
			fmt.Print(text)
			err := gpt.SpakeHandler(toAudioComm, &gpt.Spake{Source: "User", Content: text})
			if err != nil {
				log.Println(err)
			}
		}
	}()

	voice.StartRecording(toTextcomm)
	// x := `::GET_ACTIVITIES
	// {"start_time_bounds": {"lower_bound":"2023-05-25T00:00:00.000+0900" ,"upper_bound":"2023-05-25T23:59:59.000+0900"},
	// "end_time_bounds":  {"lower_bound":"2023-05-25T00:00:00.000+0900" ,"upper_bound":"2023-05-25T23:59:59.000+0900"},
	// "participants": ["Reuben"]
	// }`
	// response, err := gpt.APIHandler(x)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Print(response)
	//speaker.TestTranscription()
	//speaker.Main()
}
