package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
)

var (
	t       *testing.T
	testApp = test.NewApp()
	window  = setupMainWindowTest(testApp, t)
)

func TestMainWindowCreation(t *testing.T) {
	// Check if the window was created
	assert.NotNil(t, window, "Main window was not created")
}

func TestWindowIcon(t *testing.T) {
	// Load the icon
	// Below code causes nil pointer dereference maybe
	icon, err := fyne.LoadResourceFromPath("./icon.png")
	assert.NotNil(t, icon, "Icon was not loaded")
	assert.Nil(t, err, "Error while loading icon: ")
	// assert.Nil(t, err, "Error while loading icon: ", err.Error())

	// Set the icon
	window.SetIcon(icon)
	testApp.SetIcon(icon)

	// Check if the window icon is set
	assert.NotNil(t, window.Icon(), "Window icon is not set")
	assert.NotNil(t, testApp.Icon(), "Window icon is not set")
	// assert.Nil(t, window.Icon(), "Window icon is not set")
	// assert.Nil(t, testApp.Icon(), "Window icon is not set")
}

func TestWindowSize(t *testing.T) {
	// Check the window size
	expectedSize := fyne.NewSize(1000, 600)
	assert.Equal(t, expectedSize, window.Canvas().Size(), "Incorrect window size")
}

func TestWindowContent(t *testing.T) {
	// Check if the window content is set
	assert.NotNil(t, window.Content(), "Window content is not set")
}

func TestWindowTitle(t *testing.T) {
	// Check the window title
	assert.Equal(t, "File Explorer", window.Title(), "Incorrect window title")
}

func setupMainWindowTest(a fyne.App, t *testing.T) fyne.Window {
	w := a.NewWindow("File Explorer")
	w.SetTitle("File Explorer")

	w.SetContent(container.NewPadded())
	w.Resize(fyne.NewSize(1000, 600))

	// ... rest of your setup code ...

	return w
}
