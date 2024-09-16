package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"math"

	// "image/draw"

	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/draw"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/grafana/pyroscope-go"
	_ "github.com/mattn/go-sqlite3"
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
	// case theme.ColorNameButton:
	// 	return color.Alpha16{R: 0x95, G: 0xdd, B: 0xe9, A: 0xff}
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
		return 3
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

type Options struct {
	DatabasePath string
	ExcludedDirs map[string]int
	Profiling    bool
	Timezone     int // Timezone like UTC+3 or UTC-3
	SortDesc     bool
	UseRGB       bool
	ExifFields   []string // exif fields to display in the sidebar
}

func (opts Options) InitDefault() *Options {
	return &Options{
		DatabasePath: "./index.db",
		ExcludedDirs: map[string]int{},
		Profiling:    false,
		Timezone:     3,
		SortDesc:     true,
		UseRGB:       false,
		ExifFields:   []string{"DateTime"},
	}
}

// type CustomTheme struct {
// 	fyne.Theme
// }

// func (c *CustomTheme) NewCustomTheme() fyne.Theme {
// 	return &CustomTheme{}
// }

// func (c *CustomTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
// 	return theme.DefaultTheme().Color(n, v)
// }

var (
	imageTypes = map[string]struct{}{
		".jpg": {}, ".png": {}, ".jpeg": {}, ".gif": {}, ".bmp": {}, ".ico": {},
	}
	resourceCache sync.Map
	logger        *log.Logger
	options       = new(Options).InitDefault()
)

const thumbnailSize = 256

func main() {
	logger = log.New(os.Stdout, "", log.LstdFlags)

	// setupProfiling()
	db := setupDatabase()
	defer db.Close()

	logger.Println("Check Obsidian Todo list")
	logger.Println("Make displayImages work with getImagesFromDatabase")

	options.ExcludedDirs = map[string]int{"Games": 1, "games": 1, "go": 1}
	logger.Println("ExcludedDirs: ", options.ExcludedDirs)

	// logger.Println("Minimize widget updates:
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
	logger.Println("Before: ", getImageCount(db))

	// discoverSuccess, err := discoverImages(db)
	// if err != nil {
	// 	// dialog.ShowError(err, w)
	// 	// return
	// 	logger.Fatalln("Error discovering images:", err)
	// }
	// logger.Println("Discover success: ", discoverSuccess)

	// makes a channel that will be closed when the discovery is complete
	done := make(chan bool)

	// runs the discovery in the background
	// Discovery using goroutine
	go func() {
		_, err := discoverImages(db)
		if err != nil {
			logger.Println("Error discovering images:", err)
		}
		// sets the done channel to true
		done <- true
	}()

	// // Wait for the discovery to complete
	// <-done

	// Discovery using waitgroup
	// var wg sync.WaitGroup

	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	discoverImages(db)
	// }()

	// wg.Wait()

	logger.Println("After: ", getImageCount(db))

	content := container.NewVBox()
	scroll := container.NewVScroll(content)

	sidebar := container.NewVBox()
	sidebarScroll := container.NewVScroll(sidebar)
	sidebarScroll.Hide()

	split := container.NewHSplit(scroll, sidebarScroll)
	split.Offset = 1 // Start with sidebar hidden

	testPath := getImagePath()

	content.RemoveAll()

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter a Tag to Search by")

	form := widget.NewEntry()
	form.SetPlaceHolder("Enter a Tag to Search by")
	form.OnChanged = func(s string) {
		imagePaths, err := getImagePathsByTag(db, "%"+s+"%")
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
		logger.Fatal(err)
	}
	filterButton := widget.NewButton("", func() {
		// sidebarScroll.Show()
		return
	})
	filterButton.Icon = loadFilterButton

	optContainer := container.NewAdaptiveGrid(2, filterButton, settingsButton)

	// controls := container.NewBorder(nil, nil, nil, settingsButton, form)
	controls := container.NewBorder(nil, nil, nil, optContainer, form)
	// controls := container.NewBorder(nil, nil, nil, optContainer)
	mainContainer := container.NewBorder(controls, nil, nil, nil, container.NewPadded(split))
	// displayImages := createDisplayImagesFunction(db, w, sidebar, sidebarScroll, split, a, content)
	dbImages, err := getImagesFromDatabase(db)
	if err != nil {
		logger.Fatal(err)
	}
	displayImages := createDisplayImagesFunctionFromDb(db, w, sidebar, sidebarScroll, split, a, mainContainer, dbImages)

	displayImages(testPath)

	w.SetContent(mainContainer)
	w.ShowAndRun()
}

func setupProfiling() {
	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)
	pyroscope.Start(pyroscope.Config{
		ApplicationName: "tagvault.golang.app",
		ServerAddress:   "http://localhost:4040",
		Logger:          pyroscope.StandardLogger,
		Tags:            map[string]string{"hostname": os.Getenv("HOSTNAME")},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
}

func setupDatabase() *sql.DB {
	db, err := sql.Open("sqlite3", "file:./index.db")
	if err != nil {
		logger.Fatal("Failed to open database: ", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to connect to database: ", err)
	}
	logger.Println("DB connection success!")

	setupTables(db)
	return db
}

func setupTables(db *sql.DB) {
	tables := []string{
		"CREATE TABLE IF NOT EXISTS `Tag`(`id` INTEGER PRIMARY KEY NOT NULL, `name` VARCHAR(255) NOT NULL, `color` VARCHAR(7) NOT NULL);",
		"CREATE TABLE IF NOT EXISTS `Image`(`id` INTEGER PRIMARY KEY NOT NULL, `path` VARCHAR(1024) NOT NULL, `dateAdded` DATETIME NOT NULL);",
		"CREATE TABLE IF NOT EXISTS `ImageTag`(`imageId` INTEGER NOT NULL, `tagId` INTEGER NOT NULL);",
		"CREATE TABLE IF NOT EXISTS `Options`(`dbPath` VARCHAR(255) NOT NULL, `timezone` VARCHAR(1024) NOT NULL, `sortDesc` BOOLEAN);",
	}
	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			logger.Fatal("Failed to create table: ", err)
		}
	}
}

func setupMainWindow(a fyne.App) fyne.Window {
	w := a.NewWindow("Tag Vault")
	w.Resize(fyne.NewSize(1000, 600))

	icon, err := fyne.LoadResourceFromPath("icon.ico")
	if err != nil {
		logger.Fatal("Failed to load icon: ", err)
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

// func isExcludedDir(dir string, blackList map[string]int) bool {
// 	logger.Println(blackList)
// 	// checks if the directory is blacklisted
// 	if blackList[dir] == 1 {
// 		return true
// 	}
// 	// checks if the directory is a hidden directory
// 	return strings.HasPrefix(dir, ".")
// }

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
	return strings.Contains(dir, ".")
}

func discoverImages(db *sql.DB) (bool, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("error getting user home directory: %w", err)
	}

	var count int = 0

	logger.Println("Discovery started.")

	directories := []string{
		filepath.Join(userHome),
	}

	stmt, err := db.Prepare("INSERT INTO Image (path, dateAdded) SELECT ?, DATETIME('now') WHERE NOT EXISTS (SELECT 1 FROM Image WHERE path = ?)")
	if err != nil {
		return false, fmt.Errorf("error preparing SQL statement: %w", err)
	}
	defer stmt.Close()

	for _, directory := range directories {
		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("error walking path %s: %w", path, err)
			}
			if info.IsDir() && isExcludedDir(path, options.ExcludedDirs) {
				// logger.Println("Skipping hidden directory: ", info.Name())
				return filepath.SkipDir
			}
			if isImageFileMap(path) {
				_, err := stmt.Exec(path, path)
				if err != nil {
					return fmt.Errorf("error inserting image path into database: %w", err)
				}
				count++
				logger.Println("Added an image.")
			}
			return nil
		})
		if err != nil {
			return false, fmt.Errorf("error walking directory %s: %w", directory, err)
		}
	}

	logger.Println("Discovery Complete. Added: ", count, "images")

	return true, nil
}

func getImageCount(db *sql.DB) int {
	var imgCount int
	count, err := db.Query("SELECT DISTINCT count(id) FROM Image;")
	if err != nil {
		logger.Println("Error getting image count:", err)
	}
	count.Scan(&imgCount)
	return imgCount
}

// func discoverImages(db *sql.DB) (bool, error) {
// 	userHome, _ := os.UserHomeDir()

// 	directories := []string{
// 		userHome + "/Pictures/",
// 		userHome + "/Documents/",
// 		userHome + "/Desktop/",
// 		userHome + "/Downloads/",
// 		userHome + "/Music/",
// 		userHome + "/Videos/",
// 	}

// 	// for each directory
// 	for _, directory := range directories {
// 		// walk the directory
// 		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
// 			// if error, return error
// 			if err != nil {
// 				return errors.New("Error walking directory: " + err.Error())
// 			}
// 			// if directory, return nil
// 			if !info.IsDir() {
// 				// if image
// 				if isImageFileMap(path) {
// 					// add image path to database
// 					db.Exec("INSERT INTO Image (path, dateAdded) SELECT ?, DATETIME('now') WHERE NOT EXISTS (SELECT 1 FROM Image WHERE path = ?);", path, path)
// 				}
// 			}
// 			return errors.New("Error walking directory: file is directory")
// 		})
// 		// if walk error, return error
// 		if err != nil {
// 			return false, errors.New("Error walking directory: " + err.Error())
// 		}
// 	}

// 	// if everything is ok, return true
// 	return true, nil
// }

func getImagesFromDatabase(db *sql.DB) ([]string, error) {
	images, err := db.Query("SELECT path FROM Image")
	if err != nil {
		return nil, err
	}
	defer images.Close()

	var imagePaths []string
	for images.Next() {
		var path string
		if err := images.Scan(&path); err != nil {
			return nil, err
		}
		imagePaths = append(imagePaths, path)
	}

	return imagePaths, nil
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
		// content := container.NewGridWithRows(3, loadingIndicator, loadingMessage, imageContainer)
		// still loading so display loading message and bar
		mainContainer.Add(content)

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, runtime.NumCPU())

		// loop through images
		for _, file := range files {
			// check if it's an image
			if !file.IsDir() && isImageFile(file.Name()) {
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
		// content := container.NewGridWithRows(3, loadingIndicator, loadingMessage, imageContainer)
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

func isFile(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return !fileInfo.IsDir(), nil
}

func displayImage(db *sql.DB, w fyne.Window, path string, imageContainer *fyne.Container, sidebar *fyne.Container, sidebarScroll *container.Scroll, split *container.Split, a fyne.App) {
	// create a placeholder image
	placeholderResource := fyne.NewStaticResource("placeholder", []byte{})
	imgButton := newImageButton(placeholderResource, nil)
	resourceChan := make(chan fyne.Resource, 1)

	// claude ai
	go func() {
		// load the image as a fyne resource
		resource, err := loadImageResourceThumbnailEfficient(path)
		if err != nil {
			logger.Printf("No resource image empty %s: %v", path, err)
			resourceChan <- placeholderResource
			canvas.Refresh(imgButton)
			return
		}

		// set the image button image to the resource
		// logger.Println("Resource image not empty.", resource.Content()[:16])
		imgButton.image.Resource = resource
		canvas.Refresh(imgButton)
		resourceChan <- resource
	}()

	resource := <-resourceChan
	imgButton.onTapped = func() {
		// updates the sidebar
		updateSidebar(db, w, path, resource, sidebar, sidebarScroll, split, a, imageContainer)
	}

	// truncate the image name
	// truncatedName := truncateFilename(filepath.Base(path), 10)
	db.Exec("INSERT INTO Image (path, dateAdded) SELECT ?, DATETIME('now') WHERE NOT EXISTS (SELECT 1 FROM Image WHERE path = ?);", path, path)
	// label := widget.NewLabel(truncatedName)

	// make a parent container to hold the image button and label
	// imageTile := container.NewVBox(container.NewPadded(imgButton), label)
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

	// fullLabel := widget.NewLabel(filepath.Base(path))
	fullLabel := widget.NewLabel(truncateFilename(filepath.Base(path), 10))
	fullLabel.Wrapping = fyne.TextWrapWord

	dateAdded := widget.NewLabel("Date Added: " + getDate(db, path))
	dateAdded.Wrapping = fyne.TextWrapWord

	imageId := getImageId(db, path)
	tagDisplay := createTagDisplay(db, imageId)

	addTagButton := widget.NewButton("+", func() {
		showTagWindow(a, w, db, imageId, tagDisplay)
	})

	createTagButton := widget.NewButton("Create Tag", func() {
		showCreateTagWindow(a, w, db)
	})

	// sidebar.Add(fullImg)
	sidebar.Add(paddedImg)
	// sidebar.Add(fullLabel)
	// sidebar.Add(dateAdded)
	sidebar.Add(container.NewGridWithRows(2, dateAdded, fullLabel))
	sidebar.Add(tagDisplay)
	sidebar.Add(container.NewPadded(container.NewGridWithColumns(2, addTagButton, createTagButton)))

	sidebarScroll.Show()
	imageContainer.Refresh()
	tagDisplay.Refresh()
	// sidebar.Show()
	split.Offset = 0.65 // was 0.7 by default
	sidebar.Refresh()
}

func getImageId(db *sql.DB, path string) int {
	var imageId int
	err := db.QueryRow("SELECT id FROM Image WHERE path = ?", path).Scan(&imageId)
	if err != nil {
		logger.Println("Error getting image ID:", err)
		return 0
	}
	return imageId
}

func getDate(db *sql.DB, path string) string {
	var date string
	err := db.QueryRow("SELECT STRFTIME('%H:%M %d-%m-%Y', DATETIME(dateAdded, '+3 HOURS')) FROM Image WHERE path = ?", path).Scan(&date)
	if err != nil {
		logger.Println("Error getting date:", err)
		return ""
	}
	return date
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
			c, _ := HexToColor(color)
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

	if options.UseRGB {
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
			hex := HSVToHex(h.Value, s.Value/100, v.Value/100)
			if color, err := HexToColor(hex); err == nil {
				colorPreviewRect.FillColor = color
				colorPreviewRect.Refresh()
			}
		}
		getHexColor = func() string {
			return HSVToHex(h.Value, s.Value/100, v.Value/100)
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

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	_, ok := imageTypes[ext]
	return ok
}

func isImageFileMap(filename string) bool {
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

func truncateFilename(filename string, maxLength int) string {
	// get the file extension
	ext := filepath.Ext(filename)
	// get the filename without extension
	nameWithoutExt := filename[:len(filename)-len(ext)]
	// if filename without extension is bigger or equal to maxLength, return filename with extension
	if len(nameWithoutExt) <= maxLength {
		return filename
	} else {
		return nameWithoutExt[:maxLength] + ext
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
		thumbWidth = thumbnailSize
		thumbHeight = int(float64(thumbnailSize) / ratio)
	} else {
		thumbHeight = thumbnailSize
		thumbWidth = int(float64(thumbnailSize) * ratio)
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
	thumbImg := image.NewRGBA(image.Rect(0, 0, thumbnailSize, thumbnailSize))

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

func getImagePathsByTag(db *sql.DB, tagName string) ([]string, error) {
	query := `
        SELECT DISTINCT Image.path
        FROM Image
        JOIN ImageTag ON Image.id = ImageTag.imageId
        JOIN Tag ON ImageTag.tagId = Tag.id
        WHERE Tag.name LIKE ?
    `

	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(tagName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return paths, nil
}

// Function to handle tag-based search
func searchImagesByTag(db *sql.DB, tagName string) ([]string, error) {
	query := `
		SELECT DISTINCT Image.path
		FROM Image
		JOIN ImageTag ON Image.id = ImageTag.imageId
		JOIN Tag ON ImageTag.tagId = Tag.id
		WHERE Tag.name LIKE ?
	`
	rows, err := db.Query(query, tagName)
	// rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var imagePaths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		imagePaths = append(imagePaths, path)
	}

	return imagePaths, nil
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

// Add a function to remove a tag from an image
func removeTagFromImage(db *sql.DB, imageId int, tagId int) error {
	_, err := db.Exec("DELETE FROM ImageTag WHERE imageId = ? AND tagId = ?", imageId, tagId)
	return err
}

// Modify the createTagDisplay function to include tag removal functionality
func createTagDisplay(db *sql.DB, imageId int) *fyne.Container {
	tagDisplay := container.NewAdaptiveGrid(3)

	rows, err := db.Query("SELECT Tag.id, Tag.name, Tag.color FROM ImageTag INNER JOIN Tag ON ImageTag.tagId = Tag.id WHERE ImageTag.imageId = ?", imageId)
	if err != nil {
		logger.Println("Error querying image tags:", err)
		return tagDisplay
	}
	defer rows.Close()

	for rows.Next() {
		var tagId int
		var tagName, tagColor string
		if err := rows.Scan(&tagId, &tagName, &tagColor); err != nil {
			logger.Println("Error scanning tag data:", err)
			continue
		}

		tagButton := widget.NewButton(tagName, nil)
		tagButton.Importance = widget.LowImportance
		c, _ := HexToColor(tagColor)
		rect := canvas.NewRectangle(c)
		rect.CornerRadius = 5

		tagButton.OnTapped = func() {
			dialog.ShowConfirm("Remove Tag", "Are you sure you want to remove this tag?", func(remove bool) {
				if remove {
					if err := removeTagFromImage(db, imageId, tagId); err != nil {
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

// Helper function to convert a HSV color to Hex color string
func HSVToHex(h, s, v float64) string {
	h = math.Mod(h, 360)            // Ensure hue is between 0 and 359
	s = math.Max(0, math.Min(1, s)) // Clamp saturation between 0 and 1
	v = math.Max(0, math.Min(1, v)) // Clamp value between 0 and 1

	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c

	var r, g, b float64

	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	r = (r + m) * 255
	g = (g + m) * 255
	b = (b + m) * 255

	return fmt.Sprintf("#%02X%02X%02X", uint8(r), uint8(g), uint8(b))
}

// Helper function to convert hex color to color.Color
func HexToColor(hex string) (color.Color, error) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return nil, fmt.Errorf("invalid hex color")
	}
	rgb, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return nil, err
	}
	return color.RGBA{
		R: uint8(rgb >> 16),
		G: uint8(rgb >> 8 & 0xFF),
		B: uint8(rgb & 0xFF),
		A: 255,
	}, nil
}

// Add a settings window
func showSettingsWindow(a fyne.App, parent fyne.Window, db *sql.DB) {
	settingsWindow := a.NewWindow("Settings")

	// Create a form for database path
	dbPathEntry := widget.NewEntry()
	dbPathEntry.SetText(options.DatabasePath) // Set current path

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

	timeZone := widget.NewLabel("Timezone in UTC: UTC" + strconv.Itoa(options.Timezone))
	if options.Timezone > 0 {
		timeZone = widget.NewLabel("Timezone in UTC: UTC+" + strconv.Itoa(options.Timezone))
	} else {
		timeZone = widget.NewLabel("Timezone in UTC: UTC" + strconv.Itoa(options.Timezone))
	}

	// Create a button to open the theme editor
	themeEditorButton := widget.NewButton("Theme Editor", func() {
		showThemeEditorWindow(a, defaultTheme{}, parent)
	})

	// Create a container for the settings content
	content := container.NewVBox(
		dbPathForm,
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

func getColorComponentString(c color.Color, component int) string {
	r, g, b, a := c.RGBA()
	switch component {
	case 0:
		return fmt.Sprintf("%d", uint8(r>>8))
	case 1:
		return fmt.Sprintf("%d", uint8(g>>8))
	case 2:
		return fmt.Sprintf("%d", uint8(b>>8))
	case 3:
		return fmt.Sprintf("%d", uint8(a>>8))
	default:
		return "0"
	}
}

func parseColorComponent(s string) uint8 {
	v, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return 0
	}
	return uint8(v)
}

func showColorPickerWindow(propertyName string, colorPreview *canvas.Rectangle, currentTheme fyne.Theme, a fyne.App, w fyne.Window) {
	colorPickerWindow := a.NewWindow("Color Picker")
	colorPickerWindow.SetTitle("Color picker")

	colorPreviewRect := canvas.NewRectangle(color.NRGBA{0, 0, 130, 255})
	colorPreviewRect.SetMinSize(fyne.NewSize(64, 128))
	colorPreviewRect.CornerRadius = 5

	var content *fyne.Container
	var updateColor func()

	if options.UseRGB {
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
			hex := HSVToHex(h.Value, s.Value, v.Value)
			if newColor, err := HexToColor(hex); err == nil {
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
