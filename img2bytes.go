package main

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"log"
	"main/pkg/fileutils"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Println("Usage: ", os.Args[0], " <image_path>")
		os.Exit(1)
	}
	imagePath := os.Args[1]

	if !fileutils.IsImageFileMap(imagePath) {
		log.Fatalln("Error: ", imagePath, " is not an image")
	}

	imgBytes, err := ImgToBytes(imagePath)
	if err != nil {
		log.Fatalln("Error:", err)
	}
	fmt.Println(imgBytes)
}

func ImgToBytes(imagePath string) ([]byte, error) {
	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("error opening image file: %v", err)
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	// Create a buffer to store the encoded image
	buf := new(bytes.Buffer)

	// Encode the image based on its type
	switch img := img.(type) {
	case *image.Paletted:
		err = gif.Encode(buf, img, nil)
	case *image.RGBA, *image.NRGBA, *image.Gray:
		err = png.Encode(buf, img)
	case *image.YCbCr:
		err = jpeg.Encode(buf, img, nil)
	default:
		return nil, fmt.Errorf("unsupported image type: %T", img)
	}

	if err != nil {
		return nil, fmt.Errorf("error encoding image: %v", err)
	}

	return buf.Bytes(), nil
}
