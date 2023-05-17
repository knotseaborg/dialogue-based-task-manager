package speaker

import (
	"log"
	"os"

	"github.com/MarkKremer/microphone"
	"github.com/faiface/beep/wav"
)

func record(pipe chan []byte, filename string) {
	log.Println("recording initiated...")

	err := microphone.Init()
	if err != nil {
		log.Fatal(err)
	}
	defer microphone.Terminate()

	stream, format, err := microphone.OpenDefaultStream(44101, 1)
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}

	// Stop the stream when the user tries to quit the program.
	go func() {
		<-pipe
		log.Println("recording stopped...")
		stream.Stop()
		stream.Close()
	}()

	stream.Start() //This is a stopping function

	err = wav.Encode(f, stream, format)
	if err != nil {
		log.Fatal(err)
	}
}
