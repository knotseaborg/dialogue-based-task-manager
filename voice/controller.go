package voice

import (
	"fmt"
	"log"
	"os"

	"github.com/knotseaborg/dbtm/gpt"
)

func StartRecording(comm chan string, UIInput chan []byte) {
	fmt.Println("Press <enter> to start/stop recording.")

	//Make channel to detect start/stop inputs
	recordingTrigger := make(chan []byte, 1)

	var consoleInput []byte = make([]byte, 1)
	for {
		// Wait for user input to begin recording
		if UIInput == nil {
			os.Stdin.Read(consoleInput)
		} else {
			<-UIInput
		}
		go record(recordingTrigger, os.Getenv("DBTM_AUDIO_INPUT_PATH"))
		// This channel input stops recording go routine
		if UIInput == nil {
			os.Stdin.Read(consoleInput)
			recordingTrigger <- consoleInput
		} else {
			input, ok := <-UIInput
			if ok {
				log.Println("Error: Cannot process UIInput")
			}
			recordingTrigger <- input
		}
		text, err := gpt.Transcript()
		// Push transcript into channel
		if err != nil {
			log.Print(err)
			panic(err)
		}
		comm <- text
	}
}

func StartPlaying(comm chan string, UIOutput chan []byte) {
	for {
		_, err := ToAudio(<-comm, UIOutput)
		if err != nil {
			log.Panic(err)
		}
	}
}
