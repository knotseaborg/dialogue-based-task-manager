package voice

import (
	"fmt"
	"log"
	"os"

	"github.com/knotseaborg/dbtm/gpt"
)

func StartRecording(comm chan string, startSignal chan []byte) {
	fmt.Println("Press <enter> to start/stop recording.")

	//Make channel to detect start/stop inputs
	recordingTrigger := make(chan []byte, 1)

	var consoleInput []byte = make([]byte, 1)
	for {
		// Wait for user input to begin recording
		if startSignal == nil {
			os.Stdin.Read(consoleInput)
		} else {
			<-startSignal
		}
		go record(recordingTrigger, os.Getenv("DBTM_AUDIO_INPUT_PATH"))
		// This channel input stops recording go routine
		if startSignal == nil {
			os.Stdin.Read(consoleInput)
			recordingTrigger <- consoleInput
		} else {
			input, ok := <-startSignal
			if !ok {
				log.Println("Error: Cannot process startSignal")
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

func StartPlaying(comm chan string, stopSignal chan []byte, message chan string) {
	for {
		text := <-comm
		message <- text
		_, err := ToAudio(text)
		if err != nil {
			log.Panic(err)
		}

		// Send stop signal to UI. This tells the UI that the Audio has finished playing.
		stopSignal <- []byte("x")
	}
}
