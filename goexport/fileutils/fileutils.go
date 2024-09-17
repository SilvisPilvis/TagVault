package fileutils

import (
	"os"
	"path/filepath"
	"strings"
)

var (
	imageTypes = map[string]struct{}{
		".jpg": {}, ".png": {}, ".jpeg": {}, ".gif": {}, ".bmp": {}, ".ico": {},
	}
)

func IsFile(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return !fileInfo.IsDir(), nil
}

func IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	_, ok := imageTypes[ext]
	return ok
}

func IsImageFileMap(filename string) bool {
	// create a map of image extensions
	imageTypes := map[string]int{
		".jpg":  1,
		".png":  1,
		".jpeg": 1,
		".gif":  1,
		".bmp":  1,
		".ico":  1,
	}
	// get the file extension
	ext := strings.ToLower(filepath.Ext(filename))
	// if the file extension is in the map return true
	return imageTypes[ext] != 0
}
