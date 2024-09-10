package main

import (
	// 	"bytes"
	// 	"database/sql"
	// 	"fmt"
	"image"
	"image/color"

	// 	// "image/draw"
	// 	"image/jpeg"
	// 	"image/png"
	// 	"log"
	// 	"os"
	// 	"path/filepath"
	// 	"runtime"
	// 	"strconv"
	// 	"strings"
	// 	"sync"

	// 	"golang.org/x/image/draw"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	// "github.com/grafana/pyroscope-go"
	// _ "github.com/mattn/go-sqlite3"
)

// CreateColoredButton creates a button with a custom background color
func CreateColoredButton(text string, color color.Color, tapped func()) *widget.Button {
	button := widget.NewButton(text, tapped)

	// Create a new image with the desired color
	img := image.NewRGBA(image.Rect(0, 0, 150, 50))
	for x := 0; x < img.Bounds().Max.X; x++ {
		for y := 0; y < img.Bounds().Max.Y; y++ {
			img.Set(x, y, color)
		}
	}

	// Create a new static resource from the image
	resource := fyne.NewStaticResource("button-bg", img.Pix)

	// Set the resource as the button's icon
	button.SetIcon(resource)

	return button
}

func setupMainWindow(a fyne.App) fyne.Window {
	w := a.NewWindow("Tag Vault")
	w.Resize(fyne.NewSize(1000, 600))

	icon, err := fyne.LoadResourceFromPath("icon.ico")
	if err != nil {
		panic(err)
	}
	a.SetIcon(icon)
	w.SetIcon(icon)

	return w
}

func main() {
	a := app.New()
	w := setupMainWindow(a)

	w.SetContent(CreateColoredButton("Test", color.RGBA{R: 255, G: 0, B: 0, A: 255}, func() {
		dialog.ShowInformation("Test", "Test", w)
	}))
	w.ShowAndRun()
}
