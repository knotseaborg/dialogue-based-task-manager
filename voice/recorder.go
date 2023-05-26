package voice

import (
	"fmt"
	"log"
	"os/exec"
)

func record(trigger chan []byte, filename string) {

	cmd := exec.Command("rec", filename)
	if err := cmd.Start(); err != nil {
		log.Println("SOX is necessary to record spech")
		log.Panic(err)
	}
	go func() {
		<-trigger
		if err := exec.Command("kill", "-9", fmt.Sprintf("%d", cmd.Process.Pid)).Start(); err != nil {
			log.Panic(err)
		}
		log.Println("recording stopped...")
	}()
	log.Println("recording initiated...")
}
