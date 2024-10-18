package fileutils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	// imageTypes = map[string]struct{}{
	// 	".jpg": {}, ".png": {}, ".jpeg": {}, ".gif": {}, ".bmp": {}, ".ico": {},
	// }
	imageMap = map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".bmp":  true,
		".gif":  true,
		".tiff": true,
		".webp": true,
		".svg":  false,
		".ico":  false,
		".raw":  true,
		".heic": true,
		".avif": true,
		".avi":  true,
		".qoi":  true,
		".dng":  true,
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

func GetFileMD5HashBuffered(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	hash := md5.New()
	buf := make([]byte, 1024*1024) // 1MB buffer

	for {
		n, err := file.Read(buf)
		if n > 0 {
			_, err := hash.Write(buf[:n])
			if err != nil {
				return "", fmt.Errorf("error writing to hash: %v", err)
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return "", fmt.Errorf("error reading file: %v", err)
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func GetDirFiles(path string) ([]string, error) {
	dirFiles, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %v", err)
	}

	var files []string
	for _, v := range dirFiles {
		if v.IsDir() {
			// if file is a directory append /
			files = append(files, v.Name()+"/")
			// continue
		} else {
			files = append(files, v.Name())
		}
		// if filepath.Ext(v.Name()) != ".md" {
		// 	continue
		// }
	}

	return files, nil
}
