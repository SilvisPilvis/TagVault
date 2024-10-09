package imageconv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"image"
	"image/gif"
	"image/jpeg"
	"image/png"

	chaiWebp "github.com/chai2010/webp"

	// svgo "github.com/ajstarks/svgo"
	"github.com/gen2brain/avif"
	"github.com/gen2brain/svg"
	"github.com/jdeng/goheif"
	strukHeif "github.com/strukturag/libheif/go/heif"
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

var home, _ = os.UserHomeDir()

func ConvertImage(selectedFiles map[string]bool, selectedFormat string, selectedDir string) (bool, error) {
	// stores the converted image bytes
	// var resImages map[string]image.Image

	// loops through selected files and decodes them
	for key, _ := range selectedFiles {
		// gets the file extension
		ext := filepath.Ext(key)
		// open the image file
		file, err := os.Open(key)
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
		case ".png":
			img, _, err = image.Decode(file)
		case ".gif":
			img, _, err = image.Decode(file)
		case ".bmp":
			img, err = bmp.Decode(file)
		case ".tiff", ".tif":
			img, err = tiff.Decode(file)
		case ".webp":
			img, err = webp.Decode(file)
		case ".svg":
			img, err = svg.Decode(file)
		case ".avif":
			img, err = avif.Decode(file)
		case ".heif", ".heic":
			img, err = goheif.Decode(file)
		case ".qoi":
			img, err = qoi.Decode(file)
		default:
			return false, fmt.Errorf("Selected file not an image")
		}

		// switch on the selected format
		// encode the image
		// save bytes to array
		imageName := filepath.Base(key)
		imageName = imageName[:len(imageName)-len(filepath.Ext(imageName))]
		imageName += "." + strings.ToLower(selectedFormat)
		res, err := os.Create(selectedDir + "/" + imageName)

		switch selectedFormat {
		case "PNG":
			err = png.Encode(res, img)
			// resImages[key] = res
		case "JPG", "JPEG":
			err = jpeg.Encode(res, img, &jpeg.Options{Quality: 85})
		case "WEBP":
			err = chaiWebp.Encode(res, img, &chaiWebp.Options{Quality: 85})
		case "GIF":
			err = gif.Encode(res, img, &gif.Options{})
		case "BMP":
			err = bmp.Encode(res, img)
		case "TIFF", "TIF":
			err = tiff.Encode(res, img, &tiff.Options{Compression: tiff.Deflate})
		// case "SVG":
		// 	// err = svg.Encode(res, img)
		// 	w := img.Bounds().Size().X
		// 	h := img.Bounds().Size().Y

		// 	canvas := svgo.New(res)
		// 	canvas.Start(w, h)
		// 	canvas.Rect(0, 0, w, h)

		// 	for y := 0; y < h; y++ {
		// 		for x := 0; x < w; x++ {
		// 			r, g, b, a := img.At(x, y).RGBA()
		// 			canvas.RGBA((x*100)+50, (y*100)+50, r, g, b, a)
		// 			canvas.Writer
		// 		}
		// 	}

		// 	canvas.End()
		case "AVIF":
			err = avif.Encode(res, img, avif.Options{Quality: 85})
		case "HEIC":
			// heif_ctx, _ := strukHeif.NewContext()
			_, err = strukHeif.EncodeFromImage(img, strukHeif.Compression(strukHeif.CompressionHEVC), 85, strukHeif.LosslessModeEnabled, strukHeif.LoggingLevelBasic)
			if err != nil {
				return false, err
			}
			// strukHeif.NewImage()
		case "QOI":
			err = qoi.Encode(res, img)
		default:
			return false, fmt.Errorf("Selected format not an image type")
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
		return true, nil
	}

	return true, nil
}
