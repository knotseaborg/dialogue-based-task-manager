package ui

import (
	"log"
	"path/filepath"
	"time"

	"github.com/sciter-sdk/go-sciter"
	"github.com/sciter-sdk/go-sciter/window"
)

type UIStream struct {
	StartSignal chan []byte
	StopSignal  chan []byte
	Message     chan string
}

var currentStatus string

func CreateWindow(stream *UIStream) {
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
	setEventHandler(w, stream.StartSignal)

	// Trigger UI when a stop signal is received
	go func() {
		for {
			// Wait for Emma's output
			<-stream.StopSignal
			err = switchEmma(w, "emma_2.png")
			if err != nil {
				log.Println(err)
			}
		}
	}()

	// Trigger UI when Emma's reponse is received
	go func() {
		for {
			text := <-stream.Message
			if err = updateMessage(w, text); err != nil {
				log.Println(err)
			}
		}
	}()

	w.Show()
	w.Run()
}

func switchEmma(w *window.Window, imgFile string) error {
	/*
		Emma has two states.
		1. Active - Eyes Open
			Emma listens and speaks
		2. Standby - Eyes closed
			Does nothing
	*/
	root, err := w.GetRootElement()
	if err != nil {
		log.Panicln(err)
	}
	// Replace Emma's image
	replaceImage, err := root.SelectById("trigger")
	if err != nil {
		return err
	}
	err = replaceImage.SetAttr("src", imgFile)
	if err != nil {
		return err
	}
	w.UpdateWindow()

	return nil
}

func showSymbolicGif(w *window.Window) error {
	/*
		Uses package variable "currentStatus" to determine the gif which will be displayed
		There are two types of images.
		1. Hearing - Voice recoridng
		2. Loading - Processing Emma's response
	*/

	// Add sound wave gif to message
	root, err := w.GetRootElement()
	if err != nil {
		return err
	}
	area, err := root.SelectById("message")
	if err != nil {
		return err
	}
	area.Clear()
	// Add hearing image
	img, err := sciter.CreateElement("img", "")
	if err != nil {
		return err
	}
	img.SetAttr("src", currentStatus+".gif")
	area.Insert(img, 0)
	w.UpdateWindow()

	return nil
}

func updateMessage(w *window.Window, text string) error {
	root, err := w.GetRootElement()
	if err != nil {
		return err
	}
	area, err := root.SelectById("message")
	if err != nil {
		return err
	}
	area.Clear()
	for i := range text {
		if i > 0 && (text[i-1] == '.' || text[i-1] == '!' || text[i-1] == ',') { // Pause for the period
			time.Sleep(5 * 100000000)
		}
		time.Sleep(4 * 10000000)
		area.SetText(text[:i+1])
		w.UpdateWindow()
	}
	return nil
}

func setEventHandler(w *window.Window, startSignal chan []byte) {
	w.DefineFunction("askEmma", func(args ...*sciter.Value) *sciter.Value {
		err := switchEmma(w, "emma_3.png")
		if err != nil {
			log.Println(err)
		}
		// Set the status of Emma
		if currentStatus == "hearing" {
			currentStatus = "loading"
		} else {
			currentStatus = "hearing"
		}
		// show the loading image
		err = showSymbolicGif(w)
		if err != nil {
			log.Println(err)
		}
		startSignal <- []byte("x")
		// This is just a dummy value
		ret := sciter.NewValue()
		ret.Set("ip", sciter.NewValue("127.0.0.1"))
		return ret
	})
}
