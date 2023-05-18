package main

import (
	"fmt"
	"log"
	"os"

	"github.com/knotseaborg/dbtm/activity"
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
	activity.Main()
	//speaker.TestTranscription()
	//speaker.Main()
}
