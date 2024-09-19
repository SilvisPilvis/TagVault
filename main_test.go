package main_test

import (
	"main/goexport/fileutils"
	"path/filepath"
	"testing"

	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
)

var (
	t         *testing.T
	testApp   = test.NewApp()
	window    = setupMainWindowTest(testApp, t)
	blackList = map[string]int{"go": 1, "Games": 1, "games": 1}
)

func setupMainWindowTest(a fyne.App, t *testing.T) fyne.Window {
	w := a.NewWindow("Tag Vault")
	w.SetTitle("Tag Vault")

	w.SetContent(container.NewPadded())
	w.Resize(fyne.NewSize(1000, 600))

	return w
}

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

	// Ja negrib errorus var vnk assert not nil icon
	assert.NotNil(t, window.Icon(), "Window icon is nil so not set")

	// Check if the window icon is set
	assert.NotNil(t, window.Icon(), "Window icon is nil so not set")
	assert.NotNil(t, testApp.Icon(), "Window icon is nil so not set")
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
	assert.Equal(t, "Tag Vault", window.Title(), "Incorrect window title")
}

func TestIsImageFile(t *testing.T) {
	// Check the window title
	assert.True(t, fileutils.IsImageFileMap("image.png"), "Check Image file test failed")
}

func TestNotImageFile(t *testing.T) {
	// Check the window title
	testFiles := []string{"amogus.sus", "big.chungus", "sigma.mail"}
	for _, v := range testFiles {
		assert.False(t, fileutils.IsImageFileMap(v), "Check Image file test failed")
	}
}

func TestDirectoryExclusionHidden(t *testing.T) {
	// Check the window title
	assert.True(t, isExcludedDir("/home/amaterasu/.cache", blackList), "Hidden Directory is not excluded")
}

func TestDirectoryExclusionBlacklistCreation(t *testing.T) {
	// Check the window title
	assert.NotNil(t, blackList, "Blacklist was not created")
}

func TestDirectoryExclusionBlacklist(t *testing.T) {
	// Check the window title
	assert.True(t, isExcludedDir("/home/amaterasu/go", blackList), "Blacklisted Directory is not excluded")
}

func isExcludedDir(dir string, blackList map[string]int) bool {
	// checks if the directory is blacklisted
	for key := range blackList {
		if strings.Contains(dir, key) {
			return true
		}
	}
	// checks if the directory (not the full path so useless) is a hidden directory
	// return strings.HasPrefix(dir, ".")
	// checks if the path is a hidden directory
	// Claude solution
	return strings.HasPrefix(filepath.Base(dir), ".")
}
