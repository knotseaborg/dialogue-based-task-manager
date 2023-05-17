package main

import (
	"fmt"
	"os"

	"github.com/knotseaborg/dbtm/speaker"
)

func main() {
	//activity.Main()
	if err := os.Mkdir("./temp", os.ModePerm); err != nil {
		fmt.Println(err)
	}
	//speaker.TestTranscription()
	speaker.Main()
}
