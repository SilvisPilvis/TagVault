package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"main/goexport/colorutils"
	"main/goexport/database"
	"main/goexport/fileutils"
	"main/goexport/logger"
	"main/goexport/options"
	"main/goexport/profiling"
	"strings"

	// "main/goexport/fynecomponents/imgbtn"

	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	// "image/draw"

	"golang.org/x/image/draw"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

// make a new theme called defaultTheme
type defaultTheme struct{}

// the new defaultTheme colors
func (defaultTheme) Color(c fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch c {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x04, G: 0x10, B: 0x11, A: 0xff}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x9e, G: 0xbd, B: 0xff, A: 0xff}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 0x26, G: 0x26, B: 0x26, A: 0xff}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x42}
	case theme.ColorNameError:
		return color.NRGBA{R: 0xf4, G: 0x43, B: 0x36, A: 0xff}
	case theme.ColorNameFocus:
		return color.NRGBA{R: 0xa7, G: 0x2c, B: 0xd4, A: 0x7f}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xe6, G: 0xf7, B: 0xfa, A: 0xff}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xf}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x19}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0xe6, G: 0xf7, B: 0xfa, A: 0xff}
	case theme.ColorNamePressed:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x66}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0x95, G: 0xdd, B: 0xe9, A: 0xff}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0x3f, G: 0x1d, B: 0x8b, A: 0xff}
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x66}
	default:
		return theme.DefaultTheme().Color(c, v)
	}
}

// the new defaultTheme fonts
func (defaultTheme) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace {
		return theme.DefaultTheme().Font(s)
	}
	if s.Bold {
		if s.Italic {
			return theme.DefaultTheme().Font(s)
		}
		return theme.DefaultTheme().Font(s)
	}
	if s.Italic {
		return theme.DefaultTheme().Font(s)
	}
	return theme.DefaultTheme().Font(s)
}

// the new defaultTheme icons
func (defaultTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

// the new defaultTheme font size
func (defaultTheme) Size(s fyne.ThemeSizeName) float32 {
	switch s {
	case theme.SizeNameCaptionText:
		return 11
	case theme.SizeNameInlineIcon:
		return 20
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameScrollBar:
		return 16
	case theme.SizeNameScrollBarSmall:
		return 6
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameText:
		return 14
	case theme.SizeNameInputBorder:
		return 2
	default:
		return theme.DefaultTheme().Size(s)
	}
}

var (
	resourceCache sync.Map
	appOptions    = new(options.Options).InitDefault()
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

	appOptions.ExcludedDirs = map[string]int{"Games": 1, "games": 1, "go": 1, "TagVault": 1}
	appLogger.Println("ExcludedDirsLen: ", len(appOptions.ExcludedDirs))
	appLogger.Println("ExcludedDirs: ", appOptions.ExcludedDirs)
	appLogger.Println("If there are no images in the directory then add the highest directory that contains images to the ExcludedDirs list")

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

	a.Settings().SetTheme(&defaultTheme{})

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

	defaultPath := getImagePath()

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
		showSettingsWindow(a, w, db)
	})
	settingsButton.Icon = theme.SettingsIcon()

	loadFilterButton, err := fyne.LoadResourceFromPath("./icons/filter.png")
	if err != nil {
		appLogger.Fatal(err)
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
	displayImages(defaultPath)

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

func getImagePath() string {
	userHome, _ := os.UserHomeDir()
	if runtime.GOOS == "linux" {
		return userHome + "/Attēli/wallpapers/"
	}
	return `C:\Users\Silvestrs\Desktop\test`
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
		// mainContainer.Add(imageContainer)

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
	tagDisplay := createTagDisplay(db, imageId)

	addTagButton := widget.NewButton("+", func() {
		showTagWindow(a, w, db, imageId, tagDisplay)
	})

	createTagButton := widget.NewButton("Create Tag", func() {
		showCreateTagWindow(a, w, db)
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

func showTagWindow(a fyne.App, parent fyne.Window, db *sql.DB, imgId int, tagList *fyne.Container) {
	tagWindow := a.NewWindow("Tags")
	tagWindow.SetTitle("Add a Tag")

	content := container.NewGridWithColumns(4)
	loadingLabel := widget.NewLabel("Loading tags...")
	content.Add(loadingLabel)

	tagWindow.SetContent(container.NewVScroll(content))
	tagWindow.Resize(fyne.NewSize(300, 200))
	tagWindow.Show()

	go func() {
		tags, err := db.Query("SELECT id, name, color FROM Tag WHERE id NOT IN (SELECT tagId FROM ImageTag WHERE imageId = ?)", imgId)
		if err != nil {
			parent.Canvas().Refresh(parent.Content())
			fmt.Print("showTagWindow")
			dialog.ShowError(err, parent)
			return
		}
		defer tags.Close()

		var buttons []*fyne.Container
		for tags.Next() {
			var id int
			var name string
			var color string
			if err := tags.Scan(&id, &name, &color); err != nil {
				parent.Canvas().Refresh(parent.Content())
				fmt.Print("showTagWindow")
				dialog.ShowError(err, parent)
				return
			}

			button := widget.NewButton(name, nil)
			button.Importance = widget.LowImportance
			c, _ := colorutils.HexToColor(color)
			rect := canvas.NewRectangle(c)
			rect.CornerRadius = 5

			tagID := id
			button.OnTapped = func() {
				go func() {
					_, err := db.Exec("INSERT OR IGNORE INTO ImageTag (imageId, tagId) VALUES (?, ?)", imgId, tagID)
					parent.Content().Refresh()
					if err != nil {
						fmt.Print("showTagWindow")
						dialog.ShowError(err, parent)
					} else {
						tagList.Add(container.NewPadded(container.NewStack(rect, button)))
						tagList.Refresh()
						dialog.ShowInformation("Success", "Tag Added", parent)
						tagWindow.Close()
					}
				}()
			}
			buttons = append(buttons, container.NewPadded(container.NewStack(rect, button)))
		}

		tagWindow.Canvas().Refresh(content)
		content.Remove(loadingLabel)
		for _, button := range buttons {
			content.Add(button)
		}
	}()
}

func showCreateTagWindow(a fyne.App, parent fyne.Window, db *sql.DB) {
	tagWindow := a.NewWindow("Create Tag")
	tagWindow.SetTitle("Create a Tag")

	colorPreviewRect := canvas.NewRectangle(color.NRGBA{0, 0, 130, 255})
	colorPreviewRect.SetMinSize(fyne.NewSize(64, 128))
	colorPreviewRect.CornerRadius = 5

	stringInput := widget.NewEntry()
	stringInput.SetPlaceHolder("Enter Tag name")

	var content *fyne.Container
	var updateColor func()
	var getHexColor func() string

	if appOptions.UseRGB {
		r, g, b := widget.NewSlider(0, 255), widget.NewSlider(0, 255), widget.NewSlider(0, 255)
		updateColor = func() {
			colorPreviewRect.FillColor = color.NRGBA{uint8(r.Value), uint8(g.Value), uint8(b.Value), 255}
			colorPreviewRect.Refresh()
		}
		getHexColor = func() string {
			return fmt.Sprintf("#%02X%02X%02X", int(r.Value), int(g.Value), int(b.Value))
		}
		for _, slider := range []*widget.Slider{r, g, b} {
			slider.OnChanged = func(_ float64) { updateColor() }
		}
		content = container.NewVBox(
			widget.NewLabel("Color preview:"),
			colorPreviewRect,
			widget.NewLabel("Red:"), r,
			widget.NewLabel("Green:"), g,
			widget.NewLabel("Blue:"), b,
		)
	} else {
		h, s, v := widget.NewSlider(0, 359), widget.NewSlider(0, 100), widget.NewSlider(0, 100)
		h.Value, s.Value, v.Value = 200, 50, 100
		h.Step, s.Step, v.Step = 1, 1, 1
		updateColor = func() {
			hex := colorutils.HSVToHex(h.Value, s.Value/100, v.Value/100)
			if color, err := colorutils.HexToColor(hex); err == nil {
				colorPreviewRect.FillColor = color
				colorPreviewRect.Refresh()
			}
		}
		getHexColor = func() string {
			return colorutils.HSVToHex(h.Value, s.Value/100, v.Value/100)
		}
		for _, slider := range []*widget.Slider{h, s, v} {
			slider.OnChanged = func(_ float64) { updateColor() }
		}
		content = container.NewVBox(
			widget.NewLabel("Color preview:"),
			colorPreviewRect,
			widget.NewLabel("Hue:"), h,
			widget.NewLabel("Saturation:"), s,
			widget.NewLabel("Value:"), v,
		)
	}

	createButton := widget.NewButton("Create Tag", func() {
		tagName := stringInput.Text
		if tagName == "" {
			dialog.ShowInformation("Error", "Tag name cannot be empty", tagWindow)
			return
		}

		hexColor := getHexColor()

		_, err := db.Exec("INSERT INTO Tag (name, color) VALUES (?, ?)", tagName, hexColor)
		if err != nil {
			dialog.ShowError(fmt.Errorf("showCreateTagWindow: %w", err), tagWindow)
			return
		}

		dialog.ShowInformation("Tag Created", fmt.Sprintf("Tag Name: %s\nColor: %s", tagName, hexColor), tagWindow)
		tagWindow.Close()
	})

	content.Add(widget.NewLabel("Enter tag name:"))
	content.Add(stringInput)
	content.Add(createButton)

	tagWindow.SetContent(content)
	tagWindow.Resize(fyne.NewSize(300, 400))
	tagWindow.Show()

	updateColor() // Initial color update
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

// Modify the createTagDisplay function to include tag removal functionality
func createTagDisplay(db *sql.DB, imageId int) *fyne.Container {
	tagDisplay := container.NewAdaptiveGrid(3)

	rows, err := db.Query("SELECT Tag.id, Tag.name, Tag.color FROM ImageTag INNER JOIN Tag ON ImageTag.tagId = Tag.id WHERE ImageTag.imageId = ?", imageId)
	if err != nil {
		appLogger.Println("Error querying image tags:", err)
		return tagDisplay
	}
	defer rows.Close()

	for rows.Next() {
		var tagId int
		var tagName, tagColor string
		if err := rows.Scan(&tagId, &tagName, &tagColor); err != nil {
			appLogger.Println("Error scanning tag data:", err)
			continue
		}

		tagButton := widget.NewButton(tagName, nil)
		tagButton.Importance = widget.LowImportance
		c, _ := colorutils.HexToColor(tagColor)
		rect := canvas.NewRectangle(c)
		rect.CornerRadius = 5

		tagButton.OnTapped = func() {
			dialog.ShowConfirm("Remove Tag", "Are you sure you want to remove this tag?", func(remove bool) {
				if remove {
					if err := database.RemoveTagFromImage(db, imageId, tagId); err != nil {
						fmt.Print("createTagDisplay")
						dialog.ShowError(err, nil)
					} else {
						// remove the tag from the tag display
						// refreshSidebar()
					}
				}
			}, nil)
		}
		// New version with padding
		tagDisplay.Add(container.NewPadded(container.NewStack(rect, tagButton)))
	}

	return tagDisplay
}

// Add a settings window
func showSettingsWindow(a fyne.App, parent fyne.Window, db *sql.DB) {
	settingsWindow := a.NewWindow("Settings")

	// Create a form for database path
	dbPathEntry := widget.NewEntry()
	dbPathEntry.SetText(appOptions.DatabasePath) // Set current path

	// Create a form to change the index database location
	dbPathForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Database Path", Widget: dbPathEntry},
		},
		OnSubmit: func() {
			// Here you would implement the logic to change the database path
			// This might involve closing the current connection, copying the database, and opening a new connection
			dialog.ShowInformation("Database Path", "Path updated to: "+dbPathEntry.Text, settingsWindow)
		},
	}

	// Create a list of all excluded directories
	blackList := widget.NewList(
		func() int {
			return len(appOptions.ExcludedDirs)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Excluded directory")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			for excluded := range appOptions.ExcludedDirs {
				label := item.(*widget.Label)
				label.SetText(excluded)
				// widget.NewLabel(excluded)
			}
		},
	)

	// Create a list of all tags
	tagList := widget.NewList(
		func() int {
			// Return the number of tags
			var count int
			db.QueryRow("SELECT COUNT(*) FROM Tag").Scan(&count)
			return count
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Tag Name")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			label := item.(*widget.Label)
			var tagName string
			db.QueryRow("SELECT name FROM Tag WHERE id = ?", id+1).Scan(&tagName)
			label.SetText(tagName)
		},
	)

	timeZone := widget.NewLabel("Timezone in UTC: UTC" + strconv.Itoa(appOptions.Timezone))
	if appOptions.Timezone > 0 {
		timeZone = widget.NewLabel("Timezone in UTC: UTC+" + strconv.Itoa(appOptions.Timezone))
	} else {
		timeZone = widget.NewLabel("Timezone in UTC: UTC" + strconv.Itoa(appOptions.Timezone))
	}

	// Create a button to open the theme editor
	themeEditorButton := widget.NewButton("Theme Editor", func() {
		showThemeEditorWindow(a, defaultTheme{}, parent)
	})

	// Create a container for the settings content
	content := container.NewVBox(
		dbPathForm,
		widget.NewLabel("Excluded directories"),
		blackList,
		widget.NewLabel("Tags"),
		tagList,
		timeZone,
		themeEditorButton,
		widget.NewLabel("Default sorting: Date Added, Descending"),
	)

	settingsWindow.SetContent(content)
	settingsWindow.Resize(fyne.NewSize(400, 300))
	settingsWindow.Show()
}

func showThemeEditorWindow(app fyne.App, currentTheme fyne.Theme, w fyne.Window) {
	window := app.NewWindow("Theme Editor")
	window.SetTitle("Theme Editor")
	colorProperties := []string{
		"BackgroundColor",
		"ButtonColor",
		"DisabledButtonColor",
		"TextColor",
		"DisabledTextColor",
		"IconColor",
		"DisabledIconColor",
		"PlaceHolderColor",
		"PrimaryColor",
		"HoverColor",
		"FocusColor",
		"ScrollBarColor",
		"ShadowColor",
		"ErrorColor",
	}
	content := container.NewVBox()

	// Create a map to store color previews
	colorPreviews := make(map[string]*canvas.Rectangle)

	for _, prop := range colorProperties {
		colorValue := getThemeColor(currentTheme, prop)
		colorPreview := canvas.NewRectangle(colorValue)
		colorPreview.CornerRadius = 5
		colorPreview.SetMinSize(fyne.NewSize(35, 30))

		// Store the color preview in the map
		colorPreviews[prop] = colorPreview

		changeColorButton := widget.NewButton("Change Color", func() {
			showColorPickerWindow(prop, colorPreview, currentTheme, app, window)
		})

		row := container.NewHBox(
			widget.NewLabel(prop),
			colorPreview,
			changeColorButton,
		)
		content.Add(row)
	}

	applyButton := widget.NewButton("Apply Theme", func() {
		app.Settings().SetTheme(currentTheme)
		// w.Content().Refresh()
		window.Close()
	})
	content.Add(applyButton)

	window.SetContent(container.NewVScroll(content))
	window.Resize(fyne.NewSize(600, 400))
	window.Show()
}

func getThemeColor(t fyne.Theme, prop string) color.Color {
	switch prop {
	case "BackgroundColor":
		return t.Color("background", theme.VariantDark)
	case "ButtonColor":
		return t.Color("button", theme.VariantDark)
	case "DisabledButtonColor":
		return t.Color("disabledButton", theme.VariantDark)
	case "TextColor":
		return t.Color("foreground", theme.VariantDark)
	case "DisabledTextColor":
		return t.Color("disabledForeground", theme.VariantDark)
	case "IconColor":
		return t.Color("icon", theme.VariantDark)
	case "DisabledIconColor":
		return t.Color("disabledIcon", theme.VariantDark)
	case "PlaceHolderColor":
		return t.Color("placeholder", theme.VariantDark)
	case "PrimaryColor":
		return t.Color("primary", theme.VariantDark)
	case "HoverColor":
		return t.Color("hover", theme.VariantDark)
	case "FocusColor":
		return t.Color("focus", theme.VariantDark)
	case "ScrollBarColor":
		return t.Color("scrollBar", theme.VariantDark)
	case "ShadowColor":
		return t.Color("shadow", theme.VariantDark)
	case "ErrorColor":
		return t.Color("error", theme.VariantDark)
	default:
		return color.White
	}
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

// func getColorComponentString(c color.Color, component int) string {
// 	r, g, b, a := c.RGBA()
// 	switch component {
// 	case 0:
// 		return fmt.Sprintf("%d", uint8(r>>8))
// 	case 1:
// 		return fmt.Sprintf("%d", uint8(g>>8))
// 	case 2:
// 		return fmt.Sprintf("%d", uint8(b>>8))
// 	case 3:
// 		return fmt.Sprintf("%d", uint8(a>>8))
// 	default:
// 		return "0"
// 	}
// }

// func parseColorComponent(s string) uint8 {
// 	v, err := strconv.ParseUint(s, 10, 8)
// 	if err != nil {
// 		return 0
// 	}
// 	return uint8(v)
// }

func showColorPickerWindow(propertyName string, colorPreview *canvas.Rectangle, currentTheme fyne.Theme, a fyne.App, w fyne.Window) {
	colorPickerWindow := a.NewWindow("Color Picker")
	colorPickerWindow.SetTitle("Color picker")

	colorPreviewRect := canvas.NewRectangle(color.NRGBA{0, 0, 130, 255})
	colorPreviewRect.SetMinSize(fyne.NewSize(64, 128))
	colorPreviewRect.CornerRadius = 5

	var content *fyne.Container
	var updateColor func()

	if appOptions.UseRGB {
		r, g, b := widget.NewSlider(0, 255), widget.NewSlider(0, 255), widget.NewSlider(0, 255)
		updateColor = func() {
			newColor := color.NRGBA{uint8(r.Value), uint8(g.Value), uint8(b.Value), 255}
			colorPreviewRect.FillColor = newColor
			colorPreview.FillColor = newColor
			// doesn't work
			// setThemeColor(currentTheme, propertyName, newColor)
			w.Content().Refresh()
			colorPreviewRect.Refresh()
			colorPreview.Refresh()
		}
		for _, slider := range []*widget.Slider{r, g, b} {
			slider.OnChanged = func(_ float64) { updateColor() }
		}
		content = container.NewVBox(
			widget.NewLabel("Color preview:"),
			colorPreviewRect,
			widget.NewLabel("Red:"), r,
			widget.NewLabel("Green:"), g,
			widget.NewLabel("Blue:"), b,
		)
	} else {
		h, s, v := widget.NewSlider(0, 359), widget.NewSlider(0, 1), widget.NewSlider(0, 1)
		h.Value, s.Value, v.Value = 200, 0.5, 1
		h.Step, s.Step, v.Step = 1, 0.01, 0.01
		updateColor = func() {
			hex := colorutils.HSVToHex(h.Value, s.Value, v.Value)
			if newColor, err := colorutils.HexToColor(hex); err == nil {
				colorPreviewRect.FillColor = newColor
				colorPreview.FillColor = newColor
				// doesn't work
				// setThemeColor(currentTheme, propertyName, newColor)
				w.Content().Refresh()
				colorPreviewRect.Refresh()
				colorPreview.Refresh()
			}
		}
		for _, slider := range []*widget.Slider{h, s, v} {
			slider.OnChanged = func(_ float64) { updateColor() }
		}
		content = container.NewVBox(
			widget.NewLabel("Color preview:"),
			colorPreviewRect,
			widget.NewLabel("Hue:"), h,
			widget.NewLabel("Saturation:"), s,
			widget.NewLabel("Value:"), v,
		)
	}

	pickColorButton := widget.NewButton("Pick Color", func() {
		colorPickerWindow.Close()
	})
	content.Add(pickColorButton)

	colorPickerWindow.SetContent(content)
	colorPickerWindow.Resize(fyne.NewSize(300, 400))
	colorPickerWindow.Show()
	updateColor() // Initial color update
}
