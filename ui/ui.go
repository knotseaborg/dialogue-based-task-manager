package ui

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/sciter-sdk/go-sciter"
	"github.com/sciter-sdk/go-sciter/window"
)

func CreateWindow(inputChannel chan []byte, outputChannel chan []byte) {

	// create window
	rect := sciter.NewRect(0, 0, 500, 500)
	w, err := window.New(sciter.SW_TITLEBAR|sciter.SW_CONTROLS|sciter.SW_MAIN|sciter.SW_ENABLE_DEBUG, rect)
	w.SetTitle("Emma")
	if err != nil {
		log.Println(err)
	}
	fullpath, err := filepath.Abs("ui/voice-ui.html")
	if err != nil {
		log.Println(err)
	}
	w.LoadFile(fullpath)
	setEventHandler(w, inputChannel)

	go func() {
		for {
			// Wait for Emma's output
			<-outputChannel
			err = switchStatus(w, "emma_2.png")
			if err != nil {
				log.Println(err)
			}
		}
	}()

	w.Show()
	w.Run()
}

func switchStatus(w *window.Window, imgFile string) error {
	root, err := w.GetRootElement()
	if err != nil {
		log.Panicln(err)
	}
	replaceImage, err := root.SelectById("trigger")
	if err != nil {
		return err
	}
	fmt.Println("Inside switch", replaceImage)
	err = replaceImage.SetAttr("src", imgFile)
	if err != nil {
		return err
	}
	w.UpdateWindow()
	fmt.Println("Exiting switch", imgFile)
	return nil
}

func setEventHandler(w *window.Window, inputChannel chan []byte) {
	w.DefineFunction("askEmma", func(args ...*sciter.Value) *sciter.Value {
		fmt.Println("Asking Emma")
		err := switchStatus(w, "emma_3.png")
		if err != nil {
			log.Println(err)
		}
		fmt.Println("Outside again")
		inputChannel <- []byte("x")
		ret := sciter.NewValue()
		ret.Set("ip", sciter.NewValue("127.0.0.1"))
		return ret
	})
}
