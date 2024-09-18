package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"

	// "image/color"
	"image/jpeg"
	"image/png"
	"main/goexport/apptheme"
	"main/goexport/database"
	"main/goexport/fileutils"
	"main/goexport/logger"
	"main/goexport/options"
	"main/goexport/profiling"
	"main/goexport/tagwindow"
	"main/goexport/utilwindows"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	// "main/goexport/fynecomponents/imgbtn"
	"golang.org/x/image/draw"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	// "github.com/rwcarlsen/goexif/exif"
	// "github.com/rwcarlsen/goexif/mknote"
)

type imageButton struct {
	widget.BaseWidget
	onTapped func()
	image    *canvas.Image
}

func newImageButton(resource fyne.Resource, tapped func()) *imageButton {
	img := &imageButton{onTapped: tapped}
	img.ExtendBaseWidget(img)
	img.image = canvas.NewImageFromResource(resource)
	img.image.FillMode = canvas.ImageFillContain
	img.image.SetMinSize(fyne.NewSize(150, 150))
	return img
}

func (b *imageButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.image)
}

func (b *imageButton) Tapped(*fyne.PointEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

var (
	resourceCache sync.Map
	appOptions    = new(options.Options).InitDefault()
	optionsExist  = false
	appLogger     = logger.InitLogger()
)

func main() {
	if appOptions.Profiling {
		profiling.SetupProfiling()
	}

	db := database.Init()
	defer db.Close()

	appLogger.Println("Check Obsidian Todo list")
	appLogger.Println("Make displayImages work with getImagesFromDatabase")

	optionsExist, err := options.CheckOptionsExists(db)
	if err != nil {
		appLogger.Println(err)
	}
	appLogger.Println("Do options exist? ", optionsExist)

	if !optionsExist {
		appLogger.Println("Creating options")
		appOptions = new(options.Options).InitDefault()
		appOptions.ExcludedDirs = map[string]int{"Games": 1, "games": 1, "go": 1, "TagVault": 1} // try to add filepath.Base(os.Getwd()): 1
	} else {
		appLogger.Println("Loading options")
		appOptions, err = options.LoadOptionsFromDB(db)
		if err != nil {
			appLogger.Println("Error loading options: ", err)
		}
		appLogger.Println(appOptions.ExcludedDirs)
		// appOptions.ExcludedDirs = map[string]int{"Games": 1, "games": 1, "go": 1, "TagVault": 1}
	}

	appLogger.Println("ExcludedDirsLen: ", len(appOptions.ExcludedDirs))
	appLogger.Println("ExcludedDirs: ", appOptions.ExcludedDirs)
	appLogger.Println("If there are no images in the directory then add the highest directory that contains images to the ExcludedDirs list")

	// appLogger.Println("Downloading ffmpeg")
	// err := ffmpeg.DownloadFFmpegLinux()
	// if err != nil {
	// 	appLogger.Println(err)
	// }
	// appLogger.Println("Downloaded ffmpeg")
	// os.Exit(0)

	// appLogger.Println("Minimize widget updates:
	// Fyne's object tree walking is often triggered by widget updates. Try to reduce unnecessary updates by:

	// Only updating widgets when their data actually changes
	// Using Fyne's binding system for automatic updates
	// Batching updates where possible

	// Optimize layout:
	// Complex layouts can lead to more time-consuming tree walks. Consider:

	// Simplifying your UI structure
	// Using containers efficiently (e.g., VBox, HBox instead of nested containers)
	// Avoiding deep nesting of widgets

	// Use canvas objects:
	// For static or infrequently changing elements, consider using canvas objects instead of widgets. These are generally more lightweight.
	// Implement custom widgets:
	// If you have complex custom widgets, ensure they're implemented efficiently. Override the Refresh() method to minimize unnecessary redraws.
	// Lazy loading:
	// For large datasets or complex UIs, implement lazy loading techniques to render only visible elements.
	// Caching:
	// Implement caching mechanisms for expensive computations or frequently accessed data.
	// Background processing:
	// Move time-consuming operations off the main thread using goroutines, updating the UI only when necessary.
	// Profiling and benchmarking:
	// Continue using Go's profiling tools to identify specific bottlenecks. You might want to create benchmarks for critical parts of your app.")

	a := app.NewWithID("TagVault")
	w := setupMainWindow(a)

	a.Settings().SetTheme(&apptheme.DefaultTheme{})

	// walk trough all directories and if image add to db
	appLogger.Println("Before: ", database.GetImageCount(db))

	// Discovery using waitgroup
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		// discoverImages(db)
		database.DiscoverImages(db, appOptions.ExcludedDirs)
	}()

	wg.Wait()

	content := container.NewVBox()
	scroll := container.NewVScroll(content)

	sidebar := container.NewVBox()
	sidebarScroll := container.NewVScroll(sidebar)
	sidebarScroll.Hide()

	split := container.NewHSplit(scroll, sidebarScroll)
	split.Offset = 1 // Start with sidebar hidden

	content.RemoveAll()

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter a Tag to Search by")

	form := widget.NewEntry()
	form.SetPlaceHolder("Enter a Tag to Search by")
	form.OnChanged = func(s string) {
		imagePaths, err := database.GetImagePathsByTag(db, "%"+s+"%")
		if err != nil {
			fmt.Print("searchImagesByTag")
			dialog.ShowError(err, w)
			return
		}
		updateContentWithSearchResults(content, imagePaths, db, w, sidebar, sidebarScroll, split, a)
	}

	settingsButton := widget.NewButton("", func() {
		utilwindows.ShowSettingsWindow(a, w, db, appOptions)
	})
	settingsButton.Icon = theme.SettingsIcon()

	// loadFilterButton, err := fyne.LoadResourceFromPath("./icons/filter.svg")
	loadFilterButton, err := fyne.LoadResourceFromPath("./icons/filter.png")
	if err != nil {
		appLogger.Fatal("Failed to load filter button: ", err)
	}
	filterButton := widget.NewButton("", func() {
		// sidebarScroll.Show()
		return
	})
	filterButton.Icon = loadFilterButton

	optContainer := container.NewAdaptiveGrid(2, filterButton, settingsButton)

	controls := container.NewBorder(nil, nil, nil, optContainer, form)
	mainContainer := container.NewBorder(controls, nil, nil, nil, container.NewPadded(split))

	dbImages, err := database.GetImagesFromDatabase(db, appOptions.ImageNumber)
	if err != nil {
		appLogger.Fatal(err)
	}
	displayImages := createDisplayImagesFunctionFromDb(db, w, sidebar, sidebarScroll, split, a, content, dbImages)
	// displayImages := createDisplayImagesFunction(db, w, sidebar, sidebarScroll, split, a, mainContainer)
	displayImages("")

	w.SetContent(mainContainer)
	w.ShowAndRun()
}

func setupMainWindow(a fyne.App) fyne.Window {
	w := a.NewWindow("Tag Vault")
	w.Resize(fyne.NewSize(1000, 600))

	icon, err := fyne.LoadResourceFromPath("icon.ico")
	if err != nil {
		appLogger.Fatal("Failed to load icon: ", err)
	}
	a.SetIcon(icon)
	w.SetIcon(icon)

	return w
}

func createDisplayImagesFunction(db *sql.DB, w fyne.Window, sidebar *fyne.Container, sidebarScroll *container.Scroll, split *container.Split, a fyne.App, mainContainer *fyne.Container) func(string) {
	return func(dir string) {
		// get images from directory
		files, err := os.ReadDir(dir)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		// make a grid to display images
		imageContainer := container.NewAdaptiveGrid(5) // default value 4
		// create a loading bar & start it
		loadingIndicator := widget.NewProgressBarInfinite()
		loadingIndicator.Start()
		// create a loading message
		loadingMessage := widget.NewLabel("Loading images...")
		content := container.NewVBox(loadingIndicator, loadingMessage, imageContainer)
		// still loading so display loading message and bar
		mainContainer.Add(content)

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, runtime.NumCPU())

		// loop through images
		for _, file := range files {
			// check if it's an image
			if !file.IsDir() && fileutils.IsImageFile(file.Name()) {
				// get full image path
				imgPath := filepath.Join(dir, file.Name())
				wg.Add(1)
				go func(path string) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					// display image
					displayImage(db, w, path, imageContainer, sidebar, sidebarScroll, split, a)
				}(imgPath)
			}
		}

		go func() {
			wg.Wait()

			// images finished loading so stop & remove loading bar & loading message
			loadingIndicator.Stop()
			content.Remove(loadingMessage)
			content.Remove(loadingIndicator)
			// refresh the container that contains images
			canvas.Refresh(content)
		}()
	}
}

func createDisplayImagesFunctionFromDb(db *sql.DB, w fyne.Window, sidebar *fyne.Container, sidebarScroll *container.Scroll, split *container.Split, a fyne.App, mainContainer *fyne.Container, files []string) func(string) {
	return func(dir string) {
		// make a grid to display images
		imageContainer := container.NewAdaptiveGrid(5) // default value 4
		// create a loading bar & start it
		loadingIndicator := widget.NewProgressBarInfinite()
		loadingIndicator.Start()
		// create a loading message
		loadingMessage := widget.NewLabel("Loading images...")
		content := container.NewVBox(loadingIndicator, loadingMessage, imageContainer)
		// still loading so display loading message and bar
		mainContainer.Add(content)

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, runtime.NumCPU())

		// loop through images
		for _, file := range files {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// display image
				displayImage(db, w, path, imageContainer, sidebar, sidebarScroll, split, a)
			}(file)

		}

		go func() {
			wg.Wait()

			// images finished loading so stop & remove loading bar & loading message
			loadingIndicator.Stop()
			content.Remove(loadingMessage)
			content.Remove(loadingIndicator)
			// refresh the container that contains images
			canvas.Refresh(content)
		}()
	}
}

func displayImage(db *sql.DB, w fyne.Window, path string, imageContainer *fyne.Container, sidebar *fyne.Container, sidebarScroll *container.Scroll, split *container.Split, a fyne.App) {
	// create a placeholder image
	placeholderResource := fyne.NewStaticResource("placeholder", []byte{})
	imgButton := newImageButton(placeholderResource, nil)
	// imgButton := imgbtn.NewImageButton(placeholderResource, nil)

	resourceChan := make(chan fyne.Resource, 1)

	// claude ai solution to load images in bg
	go func() {
		// load the image as a fyne resource
		resource, err := loadImageResourceThumbnailEfficient(path)
		if err != nil {
			appLogger.Printf("No resource image empty %s: %v", path, err)
			resourceChan <- placeholderResource
			canvas.Refresh(imgButton)
			return
		}

		// set the image button image to the resource
		imgButton.image.Resource = resource
		canvas.Refresh(imgButton)
		resourceChan <- resource
	}()

	resource := <-resourceChan
	imgButton.onTapped = func() {
		// updates the sidebar
		updateSidebar(db, w, path, resource, sidebar, sidebarScroll, split, a, imageContainer)
	}

	// make a parent container to hold the image button and label
	imageTile := container.NewVBox(container.NewPadded(imgButton))
	imageContainer.Add(imageTile)
}

// UNDER NO CIRCUMSTANCES CHANGE THE ORDER IN displayImage func OR THERE WILL BE ERRORS WHEN FYNE IS LOADING IMAGES
func updateSidebar(db *sql.DB, w fyne.Window, path string, resource fyne.Resource, sidebar *fyne.Container, sidebarScroll *container.Scroll, split *container.Split, a fyne.App, imageContainer *fyne.Container) {
	// clear sidebar
	sidebar.RemoveAll()

	fullImg := canvas.NewImageFromResource(resource)
	// fullImg := canvas.NewImageFromFile(path)
	fullImg.FillMode = canvas.ImageFillContain
	fullImg.SetMinSize(fyne.NewSize(200, 200))
	paddedImg := container.NewPadded(fullImg)

	fullLabel := widget.NewLabel(truncateFilename(filepath.Base(path), 20))
	fullLabel.Wrapping = fyne.TextWrapWord

	dateAdded := widget.NewLabel("Date Added: " + database.GetDate(db, path))
	dateAdded.Wrapping = fyne.TextWrapWord

	ext := filepath.Ext(path)
	fileType := widget.NewLabel("Type: " + strings.ToUpper(ext[1:]))
	fileType.Wrapping = fyne.TextWrapWord

	imageId := database.GetImageId(db, path)
	tagDisplay := tagwindow.CreateTagDisplay(db, imageId, appLogger)

	addTagButton := widget.NewButton("+", func() {
		tagwindow.ShowTagWindow(a, w, db, imageId, tagDisplay)
	})

	createTagButton := widget.NewButton("Create Tag", func() {
		// showCreateTagWindow(a, w, db)
		tagwindow.ShowCreateTagWindow(a, w, db, appOptions)
	})

	sidebar.Add(paddedImg)
	sidebar.Add(container.NewGridWithRows(3, dateAdded, fullLabel, fileType))
	sidebar.Add(tagDisplay)
	sidebar.Add(container.NewPadded(container.NewGridWithColumns(2, addTagButton, createTagButton)))

	// Show sidebar if hidden else show
	// if sidebarScroll.Visible() {
	// sidebarScroll.Hide()
	// split.Offset = 0
	// } else {
	// sidebarScroll.Show()
	// split.Offset = 0.65 // was 0.7 by default
	// }
	sidebarScroll.Show()
	split.Offset = 0.65
	imageContainer.Refresh()
	tagDisplay.Refresh()
	sidebar.Refresh()
}

func truncateFilename(filename string, maxLength int) string {
	// get the file extension
	ext := filepath.Ext(filename)
	// get the filename without extension
	nameWithoutExt := filename[:len(filename)-len(ext)]
	// if filename without extension is bigger or equal to maxLength, return filename with extension
	if len(nameWithoutExt) <= maxLength {
		return nameWithoutExt
	} else {
		// return nameWithoutExt[:maxLength] + ext
		return nameWithoutExt[:maxLength] + "..."
	}
}

// Optimized function to load image resources
// Use this for thumbnails only or add a thumbnail bool
func loadImageResourceEfficient(path string) (fyne.Resource, error) {
	if cachedResource, ok := resourceCache.Load(path); ok {
		return cachedResource.(fyne.Resource), nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode the image
	var img image.Image
	// test:= image.Decode()
	switch filepath.Ext(path) {
	case ".jpg", ".jpeg":
		img, _, err = image.Decode(file)
	case ".png":
		img, _, err = image.Decode(file)
	// Add more cases for other image types if needed
	default:
		return nil, fmt.Errorf("unsupported image format")
	}
	if err != nil {
		return nil, err
	}

	// Calculate the thumbnail dimensions while maintaining aspect ratio
	bounds := img.Bounds()
	ratio := float64(bounds.Dx()) / float64(bounds.Dy())
	var thumbWidth, thumbHeight int
	if ratio > 1 {
		thumbWidth = appOptions.ThumbnailSize
		thumbHeight = int(float64(appOptions.ThumbnailSize) / ratio)
	} else {
		thumbHeight = appOptions.ThumbnailSize
		thumbWidth = int(float64(appOptions.ThumbnailSize) * ratio)
	}

	// Create a new image with the thumbnail dimensions
	thumbImg := image.NewRGBA(image.Rect(0, 0, thumbWidth, thumbHeight))

	// Resize the image
	draw.ApproxBiLinear.Scale(thumbImg, thumbImg.Bounds(), img, img.Bounds(), draw.Over, nil)

	// Encode the resized image
	var buf bytes.Buffer
	switch filepath.Ext(path) {
	case ".jpg", ".jpeg":
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".png":
		err = png.Encode(&buf, thumbImg)
	}
	if err != nil {
		return nil, err
	}

	// Create a new static resource with the thumbnail image
	resource := fyne.NewStaticResource(filepath.Base(path), buf.Bytes())

	// Store in cache
	resourceCache.Store(path, resource)

	return resource, nil
}

func loadImageResourceThumbnailEfficient(path string) (fyne.Resource, error) {
	if cachedResource, ok := resourceCache.Load(path); ok {
		return cachedResource.(fyne.Resource), nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// exif.RegisterParsers(mknote.All...)

	// exifData, err := exif.Decode(file)
	// if err != nil {
	// 	appLogger.Fatal("Exif decoding failed: ", err)
	// }

	// artist, _ := exifData.Get(exif.Artist)
	// appLogger.Println(artist.StringVal())
	// appLogger.Println(exifData)
	// appLogger.Println(exifData.Get(exif.Artist))
	// appLogger.Println(exifData.Get(exif.XResolution))
	// appLogger.Println(exifData.Get(exif.YResolution))

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	// Calculate the square crop region from the center of the image
	bounds := img.Bounds()
	size := bounds.Dx()
	if bounds.Dy() < size {
		size = bounds.Dy()
	}
	x := (bounds.Dx() - size) / 2
	y := (bounds.Dy() - size) / 2

	// Create a new square image for the thumbnail
	thumbImg := image.NewRGBA(image.Rect(0, 0, appOptions.ThumbnailSize, appOptions.ThumbnailSize))

	// Crop and resize the image
	draw.ApproxBiLinear.Scale(
		thumbImg,
		thumbImg.Bounds(),
		img,
		image.Rect(x, y, x+size, y+size),
		draw.Over,
		nil,
	)

	// Encode the resized image
	var buf bytes.Buffer
	switch filepath.Ext(path) {
	case ".jpg", ".jpeg":
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".png":
		err = png.Encode(&buf, thumbImg)
	default:
		return nil, fmt.Errorf("unsupported image format")
	}
	if err != nil {
		return nil, err
	}

	// Create a new static resource with the thumbnail image
	resource := fyne.NewStaticResource(filepath.Base(path), buf.Bytes())

	// Store in cache
	resourceCache.Store(path, resource)

	return resource, nil
}

// Function to update the main content based on search results
func updateContentWithSearchResults(content *fyne.Container, imagePaths []string, db *sql.DB, w fyne.Window, sidebar *fyne.Container, sidebarScroll *container.Scroll, split *container.Split, a fyne.App) {
	content.RemoveAll()
	imageContainer := container.NewAdaptiveGrid(4)
	content.Add(imageContainer)

	for _, path := range imagePaths {
		displayImage(db, w, path, imageContainer, sidebar, sidebarScroll, split, a)
	}

	content.Refresh()
}

// func setThemeColor(t fyne.Theme, prop string, c color.Color) {
// 	switch prop {
// 	case "BackgroundColor":
// 		t.Color(fyne.ThemeColorName(prop), theme.VariantDark) // this should return c
// 	case "ButtonColor":
// 		t.SetColor("button", theme.VariantDark, c)
// 	case "DisabledButtonColor":
// 		t.SetColor("disabledButton", theme.VariantDark, c)
// 	case "TextColor":
// 		t.SetColor("foreground", theme.VariantDark, c)
// 	case "DisabledTextColor":
// 		t.SetColor("disabledForeground", theme.VariantDark, c)
// 	case "IconColor":
// 		t.SetColor("icon", theme.VariantDark, c)
// 	case "DisabledIconColor":
// 		t.SetColor("disabledIcon", theme.VariantDark, c)
// 	case "PlaceHolderColor":
// 		t.SetColor("placeholder", theme.VariantDark, c)
// 	case "PrimaryColor":
// 		t.SetColor("primary", theme.VariantDark, c)
// 	case "HoverColor":
// 		t.SetColor("hover", theme.VariantDark, c)
// 	case "FocusColor":
// 		t.SetColor("focus", theme.VariantDark, c)
// 	case "ScrollBarColor":
// 		t.SetColor("scrollBar", theme.VariantDark, c)
// 	case "ShadowColor":
// 		t.SetColor("shadow", theme.VariantDark, c)
// 	case "ErrorColor":
// 		t.SetColor("error", theme.VariantDark, c)
// 	default:
// 		return
// 	}
// }
