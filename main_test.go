package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

// func TestMain(t *testing.T) {
// 	t.Run("test", func(t *testing.T) {
// 		t.Log("test")
// 	})
// }

// var testApp = test.NewApp()
// var window = setupMainWindowTest(testApp, t)

func TestMainWindowCreation(t *testing.T) {
	// Create a test app
	testApp := test.NewApp()

	// Call the function that creates your main window
	window := setupMainWindowTest(testApp, t)

	// Check if the window was created
	if window == nil {
		t.Error("Main window was not created")
	}

}

func TestWindowIcon(t *testing.T) {
	// Create a test app
	testApp := test.NewApp()

	// Call the function that creates your main window
	window := setupMainWindowTest(testApp, t)

	// Load the icon
	icon, err := fyne.LoadResourceFromPath("icon.ico")
	if err != nil {
		t.Error("Failed to load icon:", err, "\n")
	}

	// Set the icon
	window.SetIcon(icon)
	testApp.SetIcon(icon)

	// Check if the window icon is set
	if window.Icon() == nil {
		t.Error("Window icon is not set")
	}

	if testApp.Icon() == nil {
		t.Error("App icon is not set")
	}

}

func TestWindowSize(t *testing.T) {
	// Create a test app
	testApp := test.NewApp()

	// Call the function that creates your main window
	window := setupMainWindowTest(testApp, t)

	// Check the window size
	expectedSize := fyne.NewSize(1000, 600)
	if window.Canvas().Size() != expectedSize {
		t.Errorf("Expected window size %v, got %v instead", expectedSize, window.Canvas().Size())
	}
}

func TestWindowContent(t *testing.T) {
	// Create a test app
	testApp := test.NewApp()

	// Call the function that creates your main window
	window := setupMainWindowTest(testApp, t)

	// Check if the window content is set
	if window.Content() == nil {
		t.Error("Window content is not set")
	}

}

func TestWindowTitle(t *testing.T) {
	// Create a test app
	testApp := test.NewApp()

	// Call the function that creates your main window
	window := setupMainWindowTest(testApp, t)

	// Check the window title
	if window.Title() != "File Explorer" {
		t.Errorf("Expected window title 'File Explorer', got '%s' instead", window.Title())
	}
}

func setupMainWindowTest(a fyne.App, t *testing.T) fyne.Window {
	w := a.NewWindow("File Explorer")
	w.SetTitle("File Explorer")

	w.SetContent(container.NewPadded())
	w.Resize(fyne.NewSize(1000, 600))

	// ... rest of your setup code ...

	return w
}
