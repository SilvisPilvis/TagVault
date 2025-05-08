package imageconv

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	chaiWebp "github.com/chai2010/webp"

	// svgo "github.com/ajstarks/svgo"
	"github.com/gen2brain/avif"
	"github.com/gen2brain/svg"

	// "github.com/jdeng/goheif"
	// strukHeif "github.com/strukturag/libheif/go/heif"
	"github.com/xfmoulet/qoi"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

var ImageTypes []string = []string{
	"PNG",
	"JPG",
	"WEBP",
	"GIF",
	"BMP",
	"TIFF",
	// "SVG",
	"AVIF",
	"HEIC",
	"QOI",
}

// var home, _ = os.UserHomeDir()

func ConvertImage(selectedFiles []string, selectedFormat string, selectedDir string) (bool, error) {
	// stores the converted image bytes
	// var resImages map[string]image.Image
	selectedDir = filepath.Clean(selectedDir) + "/"
	fmt.Println("Selected Dir Filepath Clean: ", selectedDir)

	// loops through selected files and decodes them
	for key := range selectedFiles {
		// gets the file extension
		ext := filepath.Ext(selectedFiles[key])
		// open the image file
		fmt.Println("Selected File: ", selectedFiles[key])
		file, err := os.Open(selectedFiles[key])
		if err != nil {
			// return error if failed to open file
			// should be changed to not crash the func
			return false, err
		}
		defer file.Close()

		// decode the image
		var img image.Image

		// switch on the image extension
		switch ext {
		case ".jpg", ".jpeg":
			img, _, err = image.Decode(file)
			if err != nil {
				return false, err
			}
		case ".png":
			img, _, err = image.Decode(file)
			if err != nil {
				return false, err
			}
		case ".gif":
			img, _, err = image.Decode(file)
			if err != nil {
				return false, err
			}
		case ".bmp":
			img, err = bmp.Decode(file)
			if err != nil {
				return false, err
			}
		case ".tiff", ".tif":
			img, err = tiff.Decode(file)
			if err != nil {
				return false, err
			}
		case ".webp":
			img, err = webp.Decode(file)
			if err != nil {
				return false, err
			}
		case ".svg":
			img, err = svg.Decode(file)
			if err != nil {
				return false, err
			}
		case ".avif":
			img, err = avif.Decode(file)
			if err != nil {
				return false, err
			}
		// case ".heif", ".heic":
		// 	img, err = goheif.Decode(file)
		// 	if err != nil {
		// 		return false, err
		// 	}
		case ".qoi":
			img, err = qoi.Decode(file)
			if err != nil {
				return false, err
			}
		default:
			return false, fmt.Errorf("selected file not an image")
		}

		// switch on the selected format
		// encode the image
		// save bytes to array
		imageName := filepath.Base(selectedFiles[key])
		imageName = imageName[:len(imageName)-len(filepath.Ext(imageName))]
		imageName += "." + strings.ToLower(selectedFormat)
		fmt.Println("Selected Dir Before Create: ", selectedDir)
		res, err := os.Create(selectedDir + imageName)
		if err != nil {
			fmt.Println("Failed to create converted file: ", err)
			return false, err
		}
		defer res.Close()

		resFile, err := os.Open(selectedDir + "/" + imageName)
		if err != nil {
			fmt.Println("Failed to open converted file: ", err)
			return false, err
		}
		defer resFile.Close()

		switch selectedFormat {
		case "PNG":
			err = png.Encode(res, img)
			if err != nil {
				return false, err
			}
			fmt.Println("File Converted: ", res.Name())
			// resImages[key] = res
		case "JPG", "JPEG":
			err = jpeg.Encode(res, img, &jpeg.Options{Quality: 85})
			if err != nil {
				return false, err
			}
			fmt.Println("File Converted: ", res.Name())
		case "WEBP":
			err = chaiWebp.Encode(res, img, &chaiWebp.Options{Quality: 85})
			if err != nil {
				return false, err
			}
			fmt.Println("File Converted: ", res.Name())
		case "GIF":
			err = gif.Encode(res, img, &gif.Options{})
			if err != nil {
				return false, err
			}
			fmt.Println("File Converted: ", res.Name())
		case "BMP":
			err = bmp.Encode(res, img)
			if err != nil {
				return false, err
			}
			fmt.Println("File Converted: ", res.Name())
		case "TIFF", "TIF":
			err = tiff.Encode(res, img, &tiff.Options{Compression: tiff.Deflate})
			if err != nil {
				return false, err
			}
			fmt.Println("File Converted: ", res.Name())
		case "AVIF":
			err = avif.Encode(res, img, avif.Options{Quality: 85, QualityAlpha: 85})
			if err != nil {
				return false, err
			}
			fmt.Println("File Converted: ", res.Name())
		// case "HEIC":
		// 	// heif_ctx, _ := strukHeif.NewContext()
		// 	_, err = strukHeif.EncodeFromImage(img, strukHeif.Compression(strukHeif.CompressionHEVC), 85, strukHeif.LosslessModeEnabled, strukHeif.LoggingLevelBasic)
		// 	if err != nil {
		// 		return false, err
		// 	}
		// 	fmt.Println("File Converted: ", res.Name())
		// strukHeif.NewImage()
		case "QOI":
			err = qoi.Encode(res, img)
			if err != nil {
				return false, err
			}
			fmt.Println("File Converted: ", res.Name())
		default:
			return false, fmt.Errorf("selected format not an image type")
		}

		// loop through resImages array and batch write images to disk
		// for imageName, image := range resImages {
		// 	// gets the filename with extension from path
		// 	imageName = filepath.Base(imageName)
		// 	// trims extension from filename
		// 	imageName = imageName[:len(imageName)-len(filepath.Ext(imageName))]
		// 	// adds selectedFormat to filename
		// 	imageName += "." + strings.ToLower(selectedFormat)
		// 	err := os.WriteFile(imageName, image, 0644)
		// 	if err != nil {
		// 		// return error if failed to create file
		// 		// should be changed to not crash the func
		// 		return false, err
		// 	}
		// }
		// return true, nil
	}

	return true, nil
}
