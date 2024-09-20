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
	imageMap = map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".bmp":  true,
		".gif":  true,
		".tiff": true,
		".webp": true,
		".svg":  true,
		".raw":  true,
		".heic": true,
		".avif": true,
		".avi":  true,
		".qoi":  true,
		".ico":  true,
	}
)

func IsFile(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return !fileInfo.IsDir(), nil
}

// func IsImageFile(filename string) bool {
// 	ext := strings.ToLower(filepath.Ext(filename))
// 	_, ok := imageTypes[ext]
// 	return ok
// }

func IsImageFileMap(filename string) bool {
	// get the file extension
	ext := strings.ToLower(filepath.Ext(filename))
	// if the file extension is in the image file map return true
	return imageMap[ext]
}
