package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/rivo/tview"
	"github.com/sedwards2009/smidgen"
	"github.com/sedwards2009/smidgen/micro/buffer"
)

func saveBuffer(b *buffer.Buffer, path string) error {
	return ioutil.WriteFile(path, b.Bytes(), 0600)
}

func main() {
	logFile := setupLogging()
	defer logFile.Close()

	// if len(os.Args) != 2 {
	// 	fmt.Fprintf(os.Stderr, "usage: smidgen [filename]\n")
	// 	os.Exit(1)
	// }
	// path := os.Args[1]
	path := "tview.go"

	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("could not read %v: %v", path, err)
	}

	colorscheme, _ := smidgen.LoadInternalColorscheme("monokai")

	app := tview.NewApplication()
	tview.DoubleClickInterval = 0 // Disable tview's double-click handling
	app.EnableMouse(true)

	buffer := smidgen.NewBufferFromString(string(content), path)
	root := smidgen.NewView(app, buffer)
	// root.SetRuntimeFiles(runtime.Files)
	root.SetColorscheme(colorscheme)
	// root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	// 	switch event.Key() {
	// 	case tcell.KeyCtrlS:
	// 		saveBuffer(buffer, path)
	// 		return nil
	// 	case tcell.KeyCtrlQ:
	// 		app.Stop()
	// 		return nil
	// 	}
	// 	return event
	// })
	app.SetRoot(root, true)
	app.SetFocus(root)

	if err := app.Run(); err != nil {
		log.Fatalf("%v", err)
	}
}
func setupLogging() *os.File {
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		panic("Failed to open log file: " + err.Error())
	}
	log.SetOutput(logFile)
	return logFile
}
