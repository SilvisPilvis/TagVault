package main

import (
	"os"
	"path/filepath"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// func getFiles() []string {
// 	fs.ReadDir()
// 	return []string{"hello.txt"}
// }

func main() {

	images := map[string]int{
		".jpg":  1,
		".png":  1,
		".jpeg": 1,
	}

	test := `C:\Users\Silvestrs\Desktop\test\`

	fileList, err := os.ReadDir(test)
	if err != nil {
		return
	}

	a := app.New()
	w := a.NewWindow("Hello World")
	w.Resize(fyne.NewSize(800, 600))

	// containers := container.NewVBox()
	containers := container.NewGridWrap(fyne.NewSize(256, 512))

	for _, file := range fileList {
		// containers.Add(widget.NewLabel(file.Name()))
		// check if extension is image

		if images[filepath.Ext(file.Name())] != 0 {
			// should retain aspect ratio
			// if len(file.Name()) > 10 {

			// }
			containers.Add(widget.NewCard(file.Name(), "", canvas.NewImageFromFile(test+file.Name())))
		} else {
			// the icon should be square
			containers.Add(widget.NewCard(file.Name(), "", canvas.NewImageFromFile("./icon.png")))
		}

	}

	w.SetContent(containers)

	w.ShowAndRun()
}
