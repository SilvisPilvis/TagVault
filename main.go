package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"time"

	// "image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"main/pkg/apptheme"
	"main/pkg/database"
	"main/pkg/fileutils"
	"main/pkg/icon"
	"main/pkg/logger"
	"main/pkg/options"
	"main/pkg/profiling"
	"main/pkg/tagwindow"
	"main/pkg/utilwindows"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	// "main/pkg/fynecomponents/imgbtn"

	"golang.org/x/image/draw"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"

	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	fyneGif "fyne.io/x/fyne/widget"
)

// type imageButton struct {
// 	widget.BaseWidget
// 	image        *canvas.Image
// 	onTapped     func()
// 	onLongTap    func()
// 	onRightClick func()
// }

// func newImageButton(resource fyne.Resource, tapped func(), longTap func(), rightClick func()) *imageButton {
// 	img := &imageButton{
// 		onTapped:     tapped,
// 		onLongTap:    longTap,
// 		onRightClick: rightClick,
// 	}
// 	img.ExtendBaseWidget(img)
// 	img.image = canvas.NewImageFromResource(resource)
// 	img.image.FillMode = canvas.ImageFillContain
// 	img.image.SetMinSize(fyne.NewSize(150, 150))
// 	return img
// }

// func (b *imageButton) Tapped(_ *fyne.PointEvent) {
// 	if b.onTapped != nil {
// 		b.onTapped()
// 	}
// }

// func (b *imageButton) TappedSecondary(_ *fyne.PointEvent) {
// 	if b.onRightClick != nil {
// 		b.onRightClick()
// 	}
// }

// func (b *imageButton) LongTap(_ *fyne.PointEvent) {
// 	if b.onLongTap != nil {
// 		b.onLongTap()
// 	}
// }

// func (b *imageButton) CreateRenderer() fyne.WidgetRenderer {
// 	return widget.NewSimpleRenderer(b.image)
// }

type imageButton struct {
	widget.BaseWidget
	image        *canvas.Image
	onTapped     func()
	onLongTap    func()
	onRightClick func()
	pressedTime  time.Time
	longTapTimer *time.Timer
	selected     bool
}

func newImageButton(resource fyne.Resource) *imageButton {
	img := &imageButton{}
	img.ExtendBaseWidget(img)
	img.image = canvas.NewImageFromResource(resource)
	img.image.FillMode = canvas.ImageFillContain
	img.image.SetMinSize(fyne.NewSize(150, 150))
	return img
}

func (b *imageButton) Tapped(me *desktop.MouseEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

func (b *imageButton) TappedSecondary(_ *fyne.PointEvent) {
	if b.onRightClick != nil {
		b.onRightClick()
	}
}

func (b *imageButton) LongTap(me *desktop.MouseEvent) {
	if me.Button == desktop.MouseButtonPrimary {
		if b.onLongTap != nil {
			b.onLongTap()
			b.selected = !b.selected
			b.Refresh()
		}
	}
}

func (b *imageButton) Refresh() {
	if b.selected {
		b.image.Translucency = 0.7
	} else {
		b.image.Translucency = 0
	}
	canvas.Refresh(b.image)
}

func (b *imageButton) MouseDown(me *desktop.MouseEvent) {
	if me.Button == desktop.MouseButtonPrimary {
		b.pressedTime = time.Now()
		b.longTapTimer = time.AfterFunc(time.Millisecond*200, func() {
			if b.onLongTap != nil {
				b.onLongTap()
			}
		})
	}
}

func (b *imageButton) MouseUp(_ *desktop.MouseEvent) {
	if b.longTapTimer != nil {
		b.longTapTimer.Stop()
	}
	if time.Since(b.pressedTime) < time.Millisecond*200 {
		b.Tapped(nil)
	}
}

func (b *imageButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.image)
}

var (
	resourceCache sync.Map
	appOptions    = new(options.Options).InitDefault()
	optionsExist  = false
	appLogger     = logger.InitLogger()
	page          = 0
	selectedFiles = []string{}
	home, _       = os.UserHomeDir()
	prevoiusImage = ""
)

func main() {
	db := database.Init()
	defer db.Close()

	appLogger.Println("Check Obsidian Todo list")
	appLogger.Println("Make displayImages work with getImagesFromDatabase")

	a := app.NewWithID("TagVault")
	w := setupMainWindow(a)

	optionsExist, err := options.CheckOptionsExists(db)
	if err != nil {
		appLogger.Println(err)
	}
	appLogger.Println("Do options exist? ", optionsExist)

	if !optionsExist {
		appLogger.Println("Creating options")
		appOptions = new(options.Options).InitDefault()
		// appOptions.ExcludedDirs = map[string]int{"Games": 1, "games": 1, "go": 1, "TagVault": 1, "Android": 1, "android": 1, "node_modules": 1} // try to add filepath.Base(os.Getwd()): 1
		utilwindows.ShowChooseDirWindow(a, appOptions, appLogger, db)
		err = options.SaveOptionsToDB(db, appOptions)
		if err != nil {
			appLogger.Fatalln("Failed to save Options: ", err)
		}
	} else {
		appLogger.Println("Loading options")
		appOptions, err = options.LoadOptionsFromDB(db)
		if err != nil {
			appLogger.Println("Error loading options: ", err)
		}
		appLogger.Println(appOptions.ExcludedDirs)
		// appOptions.ExcludedDirs = map[string]int{"Games": 1, "games": 1, "go": 1, "TagVault": 1}
	}

	if appOptions.Profiling {
		profiling.SetupProfiling()
	}

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

	loadFilterButton := fyne.NewStaticResource("filterIcon", icon.FilterIcon)

	filterButton := widget.NewButton("", func() {
		// sidebarScroll.Show()
		return
	})
	filterButton.Icon = loadFilterButton

	optContainer := container.NewAdaptiveGrid(2, filterButton, settingsButton)

	tabs := container.NewAppTabs(
		container.NewTabItem("Images", content),
		// widget.NewButtonWithIcon("Open new Tab", "Plus", func() {
		// 	tabs.Append(container.NewTabItem("Images", content))
		// }),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	controls := container.NewBorder(nil, nil, nil, optContainer, form)

	mainContainer := container.NewBorder(controls, nil, nil, nil, container.NewPadded(split))
	// mainContainer.Add(tabs)

	appLogger.Printf("Page: %d, ImageNumber: %d", page, appOptions.ImageNumber)

	// appLogger.Println("ExcludedDirs: ", appOptions.ExcludedDirs)
	// appLogger.Println("Current Binary Directory: ", filepath.Dir(os.Args[0]))
	// appLogger.Println("Current Binary Directory: ", cwd)

	if appOptions.FirstBoot {
		appLogger.Println("This is first boot")
		homeDir, _ := os.UserHomeDir()
		displayImages := createDisplayImagesFunction(db, w, sidebar, sidebarScroll, split, a, content)
		displayImages(homeDir + "/Pictures/wallpapers")
	} else {
		dbImages, err := database.GetImagesFromDatabase(db, page, appOptions.ImageNumber)
		if err != nil {
			appLogger.Fatal(err)
		}
		displayImages := createDisplayImagesFunctionFromDb(db, w, sidebar, sidebarScroll, split, a, content, dbImages)
		displayImages("")
	}

	// Add event listener to scroll
	scroll.OnScrolled = func(pos fyne.Position) {
		// stupid magic number
		// why does it increment by 648 when adding the same amound of images with the offset at 100?
		if scroll.Offset.Y == 100+(float32(page)*648) && !appOptions.FirstBoot {
			page += 1
			appLogger.Println("Scrolled to bottom. Current page: ", page)
			appLogger.Println("Skip images: ", page*int(appOptions.ImageNumber))
			nextImages, err := database.GetImagesFromDatabase(db, page*int(appOptions.ImageNumber), appOptions.ImageNumber)
			if err != nil {
				appLogger.Fatal("Failed to load more images on scroll: ", err)
			}

			displayImages := createDisplayImagesFunctionFromDb(db, w, sidebar, sidebarScroll, split, a, content, nextImages)
			displayImages("")
			content.Refresh()
		}
	}

	w.SetContent(mainContainer)
	w.ShowAndRun()
}

func setupMainWindow(a fyne.App) fyne.Window {
	w := a.NewWindow("Tag Vault")
	w.Resize(fyne.NewSize(1000, 600))

	icon := fyne.NewStaticResource("icon", icon.AppIcon)
	// icon, err := fyne.LoadResourceFromPath("./icon.ico")
	// if err != nil {
	// 	appLogger.Fatal("Failed to load icon: ", err)
	// }
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
			if !file.IsDir() && fileutils.IsImageFileMap(file.Name()) {
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

			// loadingMessage.Hide()
			// loadingIndicator.Hide()
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
	// imgButton := newImageButton(placeholderResource, nil, nil, nil)

	if filepath.Ext(path) == ".gif" {
		// appLogger.Println("Fix Gif Not Gifing...")

		// imgButton, err := fyneGif.NewAnimatedGifFromResource(placeholderResource)
		testPath, err := storage.ParseURI("file://" + path)
		if err != nil {
			appLogger.Fatal("Failed to parse uri: ", err)
		}
		gifButton, err := fyneGif.NewAnimatedGif(testPath)
		if err != nil {
			appLogger.Fatal("Failed to load gif: ", err)
		}
		gifButton.Show()
		// gifButton.Resize(fyne.NewSize(200, 200))
		gifButton.Start()

		imageContainer.Add(container.NewPadded(gifButton))
	} else {
		imgButton := newImageButton(placeholderResource)

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
			imgButton.image.Translucency = 0
			// imgButton.image.Refresh()
			canvas.Refresh(imgButton)
			resourceChan <- resource
		}()

		resource := <-resourceChan
		imgButton.onTapped = func() {
			// updates the sidebar
			updateSidebar(db, w, path, resource, sidebar, sidebarScroll, split, a, imageContainer)
		}

		imgButton.onLongTap = func() {
			selectedFiles = append(selectedFiles, path)
			appLogger.Println("Added new file: ", path)
			appLogger.Println("Selected files: ", selectedFiles)
			imgButton.image.Translucency = 0.7
			canvas.Refresh(imgButton)
		}

		imgButton.onRightClick = func() {
			appLogger.Println("Add functionality to open menu to add to archive and compress")
			utilwindows.ShowRightClickMenu(w, selectedFiles, a)
		}

		// make a parent container to hold the image button and label
		imageTile := container.NewVBox(container.NewPadded(imgButton))
		imageContainer.Add(imageTile)
	}

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
	if prevoiusImage == path && sidebarScroll.Visible() {
		sidebarScroll.Hide()
		split.Trailing.Hide()
		split.SetOffset(0.0)
		w.Content().Refresh()
		//	split.Offset = 0.65 // was 0.7 by default
	} else {
		sidebarScroll.Show()
		split.SetOffset(0.65)
		// sidebarScroll.Offset.X = 0.65
		prevoiusImage = path
	}

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
	case ".gif":
		err = gif.Encode(&buf, thumbImg, &gif.Options{NumColors: 256})
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
