package speaker

import (
	"fmt"
	"os"

	"github.com/knotseaborg/dbtm/gpt"
)

func Main() {
	fmt.Println("Press <enter> to start/stop recording.")

	//Make channel to detect start/stop inputs
	pipe := make(chan []byte, 1)

	var consoleInput []byte = make([]byte, 1)
	for {
		// Wait for user input to begin recording
		os.Stdin.Read(consoleInput)
		go record(pipe, os.Getenv("DBTM_AUDIO_INPUT_PATH"))
		// Wait for user input to begin recording
		os.Stdin.Read(consoleInput)
		// This channel input stops recording go routine
		pipe <- consoleInput
		text, err := gpt.Transcript()
		// Process transcript
		if err != nil {
			panic(err)
		}
		fmt.Println(text)
	}
}
