package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"main/pkg/apptheme"
	"main/pkg/components/buttons"
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

	"github.com/gen2brain/avif"
	"github.com/gen2brain/svg"
	"github.com/jdeng/goheif"
	"github.com/xfmoulet/qoi"
	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	// fyneGif "fyne.io/x/fyne/widget"
	// "fyne.io/fyne/v2/storage"
)

var (
	resourceCache sync.Map
	appOptions    = new(options.Options).InitDefault()
	optionsExist  = false
	appLogger     = logger.InitLogger()
	page          = 0
	selectedFiles = map[string]bool{}
	home, _       = os.UserHomeDir()
	prevoiusImage = ""
	orderBy       = ""
)

func main() {
	db := database.Init()
	defer db.Close()

	appLogger.Println("Check Obsidian Todo list")
	appLogger.Println("Make displayImages work with getImagesFromDatabase")

	a := app.NewWithID("TagVault")
	w := setupMainWindow(a)

	// If no options exist, this means that this is first boot
	optionsExist, err := options.CheckOptionsExists(db)
	if err != nil {
		appLogger.Println(err)
	}
	appLogger.Println("Do options exist? ", optionsExist)

	if !optionsExist {
		appLogger.Println("Creating options")
		appOptions = new(options.Options).InitDefault()

		utilwindows.ShowChooseDirWindow(a, appOptions, appLogger, db)
		err = options.SaveOptionsToDB(db, appOptions)
		if err != nil {
			appLogger.Fatalln("Failed to save Options: ", err)
		}

		database.AddImageTypeTags(db)
	} else {
		appLogger.Println("Loading options")
		appOptions, err = options.LoadOptionsFromDB(db)
		if err != nil {
			appLogger.Println("Error loading options: ", err)
		}

		appLogger.Println(appOptions.ExcludedDirs)

		err = database.VacuumDb(db)
		if err != nil {
			appLogger.Println("Failed to vacuum database: ", err)
		}

		appLogger.Println("VACUUM Executed Successfully")
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
		database.DiscoverImages(db, appOptions.ExcludedDirs)
	}()

	wg.Wait()

	// ---------- CLAUDE LAYOUT START

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

	imageContent := content

	form.OnSubmitted = func(s string) {
		imagePaths, err := database.GetImagePathsByTag(db, "%"+s+"%")
		if err != nil {
			fmt.Print("searchImagesByTag")
			dialog.ShowError(err, w)
			return
		}
		updateContentWithSearchResults(imageContent, imagePaths, db, w, sidebar, sidebarScroll, split, a)
	}

	// form.OnChanged = func(s string) {
	// 	imagePaths, err := database.GetImagePathsByTag(db, "%"+s+"%")
	// 	if err != nil {
	// 		fmt.Print("searchImagesByTag")
	// 		dialog.ShowError(err, w)
	// 		return
	// 	}
	// 	updateContentWithSearchResults(imageContent, imagePaths, db, w, sidebar, sidebarScroll, split, a)
	// 	// updateContentWithSearchResults(imageContent, imagePaths, db, w, sidebar, sidebarScroll, split, a)
	// }

	settingsButton := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		utilwindows.ShowSettingsWindow(a, w, db, appOptions)
	})

	loadFilterButton := fyne.NewStaticResource("filterIcon", icon.FilterIconLight)
	filterButton := widget.NewButton("", func() {
		return
	})
	filterButton.Icon = loadFilterButton
	// test := orderBy

	optContainer := container.NewGridWithColumns(2, filterButton, settingsButton)
	controls := container.NewBorder(nil, nil, nil, optContainer, form)

	// Create main container with tabs above controls
	mainContainer := container.NewBorder(
		controls,
		nil,
		nil,
		nil,
		container.NewPadded(split),
	)

	mainContainer = container.NewPadded(mainContainer)

	mainTab := container.NewTabItem("Images", mainContainer)
	// testTab := container.NewTabItem("Test", container.NewVBox()) // let user pick dir and diaplay all files in dir
	defaultDir, _ := fileutils.GetDirFiles("/home/amaterasu/")
	homeTab := container.NewTabItem("Home", CreateDisplayDirContentsContainer(defaultDir))

	tabs := container.NewDocTabs(
		mainTab,
		// testTab,
		homeTab,
		// Add more tabs as needed
	)
	tabs.SetTabLocation(container.TabLocationTop)

	appLogger.Printf("Page: %d, ImageNumber: %d", page, appOptions.ImageNumber)
	if appOptions.FirstBoot {
		appLogger.Println("This is first boot")
		displayImages := createDisplayImagesFunction(db, w, sidebar, sidebarScroll, split, a, imageContent)
		displayImages(home + "/Pictures")
	} else {
		dbImages, err := database.GetImagesFromDatabase(db, page, appOptions.ImageNumber)
		if err != nil {
			appLogger.Fatal(err)
		}
		displayImages := createDisplayImagesFunctionFromDb(db, w, sidebar, sidebarScroll, split, a, imageContent, dbImages)
		displayImages("")
	}

	scroll.OnScrolled = func(pos fyne.Position) {
		// appLogger.Println("Scrolled: ", pos.Y)
		if scroll.Offset.Y == 148+(float32(page)*648) && !appOptions.FirstBoot {
			page += 1
			appLogger.Println("Scrolled to bottom. Current page: ", page)
			appLogger.Println("Skip images: ", page*int(appOptions.ImageNumber))
			nextImages, err := database.GetImagesFromDatabase(db, page*int(appOptions.ImageNumber), appOptions.ImageNumber)
			if err != nil {
				appLogger.Fatal("Failed to load more images on scroll: ", err)
			}
			displayImages := createDisplayImagesFunctionFromDb(db, w, sidebar, sidebarScroll, split, a, imageContent, nextImages)
			displayImages("")
			imageContent.Refresh()
		}
	}

	// ---------- CLAUDE LAYOUT END

	appLogger.Println("Remember to delete fyne folder from `.config/fyne` folder")
	w.SetContent(tabs)
	w.ShowAndRun()
}

func setupMainWindow(a fyne.App) fyne.Window {
	w := a.NewWindow("Tag Vault")
	w.Resize(fyne.NewSize(1000, 600))

	icon := fyne.NewStaticResource("icon", icon.AppIcon)

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

	if filepath.Ext(path) == ".gif" {
		gifPath, _ := storage.ParseURI("file://" + path)
		gifButton := buttons.NewGifButton(gifPath)
		gifButton.StartAnimation()

		resourceChan := make(chan fyne.Resource, 1)

		// claude ai solution to load images in bg
		go func() {
			// load the image as a fyne resource
			resource, err := loadImageResourceThumbnailEfficient(path)
			if err != nil {
				appLogger.Printf("No resource image empty %s: %v", path, err)
				resourceChan <- placeholderResource
				return
			}

			// set the image button image to the resource
			resourceChan <- resource
		}()

		resource := <-resourceChan

		gifButton.SetOnTapped(func() {
			appLogger.Println("TAPPED GIF")
			updateSidebar(db, w, path, resource, sidebar, sidebarScroll, split, a, imageContainer)
		})

		gifButton.SetOnLongTap(func() {
			// If image is not already selected and selectedFiles is 0 or bigger than 0
			if len(selectedFiles) >= 0 && !selectedFiles[path] {
				selectedFiles[path] = true
				appLogger.Println("Added new file: ", path)
				// gifButton.Image.Translucency = 0.7
				gifButton.Selected = true
				canvas.Refresh(gifButton)
				// If image is already selected and selectedFiles is 0 or bigger than 0
			} else if len(selectedFiles) >= 0 && selectedFiles[path] {
				appLogger.Println("Removed file: ", path)
				delete(selectedFiles, path)
				// gifButton.Image.Translucency = 0
				gifButton.Selected = false
				canvas.Refresh(gifButton)
			}
			appLogger.Println("Selected files: ", selectedFiles)
		})

		gifButton.SetOnRightClick(func() {
			utilwindows.ShowRightClickMenu(w, selectedFiles, a)
		})

		imageContainer.Add(container.NewPadded(gifButton))
		// appLogger.Println("Skipping GIF")
	} else {
		imgButton := buttons.NewImageButton(placeholderResource)

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
			imgButton.Image.Resource = resource
			imgButton.Image.Translucency = 0
			// imgButton.image.Refresh()
			canvas.Refresh(imgButton)
			resourceChan <- resource
		}()

		resource := <-resourceChan
		imgButton.SetOnTapped(func() {
			updateSidebar(db, w, path, resource, sidebar, sidebarScroll, split, a, imageContainer)
		})
		// imgButton.OnTapped = func() {
		// 	// updates the sidebar
		// 	updateSidebar(db, w, path, resource, sidebar, sidebarScroll, split, a, imageContainer)
		// }

		imgButton.SetOnLongTap(func() {
			// If image is not already selected and selectedFiles is 0 or bigger than 0
			if len(selectedFiles) >= 0 && !selectedFiles[path] {
				selectedFiles[path] = true
				appLogger.Println("Added new file: ", path)
				imgButton.Image.Translucency = 0.7
				imgButton.Selected = true
				canvas.Refresh(imgButton)
				// If image is already selected and selectedFiles is 0 or bigger than 0
			} else if len(selectedFiles) >= 0 && selectedFiles[path] {
				appLogger.Println("Removed file: ", path)
				delete(selectedFiles, path)
				imgButton.Image.Translucency = 0
				imgButton.Selected = false
				canvas.Refresh(imgButton)
			}
			appLogger.Println("Selected files: ", selectedFiles)
		})
		// imgButton.OnLongTap = func() {
		// 	// If image is not already selected and selectedFiles is 0 or bigger than 0
		// 	if len(selectedFiles) >= 0 && !selectedFiles[path] {
		// 		selectedFiles[path] = true
		// 		appLogger.Println("Added new file: ", path)
		// 		imgButton.Image.Translucency = 0.7
		// 		imgButton.Selected = true
		// 		canvas.Refresh(imgButton)
		// 		// If image is already selected and selectedFiles is 0 or bigger than 0
		// 	} else if len(selectedFiles) >= 0 && selectedFiles[path] {
		// 		appLogger.Println("Removed file: ", path)
		// 		delete(selectedFiles, path)
		// 		imgButton.Image.Translucency = 0
		// 		imgButton.Selected = false
		// 		canvas.Refresh(imgButton)
		// 	}
		// 	appLogger.Println("Selected files: ", selectedFiles)
		// }

		imgButton.SetOnRightClick(func() {
			appLogger.Println("Add functionality to open menu to add to archive and compress")
			utilwindows.ShowRightClickMenu(w, selectedFiles, a)
		})
		// imgButton.OnRightClick = func() {
		// 	appLogger.Println("Add functionality to open menu to add to archive and compress")
		// 	utilwindows.ShowRightClickMenu(w, selectedFiles, a)
		// }

		// make a parent container to hold the image button and label
		imageTile := container.NewVBox(container.NewPadded(imgButton))
		imageContainer.Add(imageTile)
	}
	// appLogger.Println("Showing ", len(imageContainer.Objects), " images")
}

func CreateDisplayDirContentsContainer(dirFiles []string) *fyne.Container {
	fileContainer := container.NewAdaptiveGrid(4) // default value 4
	fileContainer.RemoveAll()

	scrollContainer := container.NewVScroll(fileContainer)

	for _, v := range dirFiles {
		// check if current item is a directory
		if string(v[len(v)-1]) == "/" {
			test, _ := fileutils.GetDirFiles(home + "/" + v)
			icon := widget.NewButtonWithIcon(truncateDirname(v, 10), theme.FolderIcon(), func() {
				SetDisplayDirNewContent(fileContainer, test, home+"/"+v)
			})
			icon.Alignment = widget.ButtonAlignCenter
			icon.IconPlacement = widget.ButtonIconLeadingText
			fileContainer.Add(icon)
		} else {
			// this will run if current item is not a dir
			icon := widget.NewButtonWithIcon(truncateFilename(v, 10, true), theme.FileIcon(), nil)
			icon.Alignment = widget.ButtonAlignCenter
			icon.IconPlacement = widget.ButtonIconLeadingText
			fileContainer.Add(icon)
		}
	}

	return container.NewPadded(container.NewStack(scrollContainer))
}

func SetDisplayDirNewContent(content *fyne.Container, files []string, currentDir string) *fyne.Container {
	content.RemoveAll()
	// appLogger.Println("Back Button: ", currentDir)
	if currentDir != home || currentDir != "/home" || currentDir != "/" {
		content.Add(widget.NewButtonWithIcon("", theme.NavigateBackIcon(),
			func() {
				parentDir := filepath.Dir(currentDir)
				if parentDir == "/" || parentDir == "/home" || parentDir == home {
					SetDisplayDirNewContent(content, nil, home)
				} else {
					SetDisplayDirNewContent(content, nil, parentDir)
				}
			}),
		)
	}

	var dirContent []string
	var err error

	if files == nil {
		dirContent, err = fileutils.GetDirFiles(currentDir)
		if err != nil {
			appLogger.Println("Error getting directory contents:", err)
			return content
		}
	} else {
		dirContent = files
	}

	for _, v := range dirContent {
		isDir := string(v[len(v)-1]) == "/"
		var icon *widget.Button

		if isDir {
			icon = widget.NewButtonWithIcon(truncateDirname(v, 10), theme.FolderIcon(), func() {
				newDir := filepath.Join(currentDir, v)
				SetDisplayDirNewContent(content, nil, newDir)
			})
		} else {
			icon = widget.NewButtonWithIcon(truncateFilename(v, 10, true), theme.FileIcon(), nil)
		}

		icon.Alignment = widget.ButtonAlignCenter
		icon.IconPlacement = widget.ButtonIconLeadingText
		content.Add(icon)
	}

	return content
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
	fullLabel := widget.NewLabel(truncateFilename(filepath.Base(path), 20, false))
	fullLabel.Wrapping = fyne.TextWrapWord
	dateAdded := widget.NewLabel("Date Added: " + database.GetDate(db, path))
	dateAdded.Wrapping = fyne.TextWrapWord
	ext := filepath.Ext(path)
	fileType := widget.NewLabel("Type: " + strings.ToUpper(ext[1:]))
	fileType.Wrapping = fyne.TextWrapWord
	imageId := database.GetImageId(db, path)
	tagDisplay := tagwindow.CreateTagDisplay(db, imageId, appLogger, sidebar, w)
	addTagButton := widget.NewButton("+", func() {
		tagwindow.ShowTagWindow(a, w, db, imageId, tagDisplay)
	})
	createTagButton := widget.NewButton("Create Tag", func() {
		tagwindow.ShowCreateTagWindow(a, w, db, appOptions, false, "", 0)
	})

	// Create fullscreen button
	fullscreenButton := widget.NewButtonWithIcon("", theme.ViewFullScreenIcon(), func() {
		// Create new image for fullscreen view
		// fullscreenImg := canvas.NewImageFromResource(resource)
		fullscreenImg := canvas.NewImageFromFile(path)
		fullscreenImg.FillMode = canvas.ImageFillContain
		fullscreenImg.SetMinSize(fyne.NewSize(600, 400)) // Set a reasonable default size

		// Create container for the image
		content := container.NewStack(fullscreenImg)

		// Show the dialog
		dialog.ShowCustom("View Image", "Close", content, w)
	})
	fullscreenButton.Importance = widget.LowImportance

	// Create button container with right alignment
	// buttonContainer := container.NewHBox(layout.NewSpacer(), fullscreenButton)

	sidebar.Add(container.NewStack(paddedImg, fullscreenButton))
	sidebar.Add(container.NewGridWithRows(3, dateAdded, fullLabel, fileType))
	sidebar.Add(tagDisplay)
	sidebar.Add(container.NewPadded(container.NewGridWithColumns(2, addTagButton, createTagButton)))
	// sidebar.Add(buttonContainer) // Add the fullscreen button container

	// Show sidebar if hidden else show
	if prevoiusImage == path && sidebarScroll.Visible() {
		sidebar.RemoveAll()
		sidebarScroll.Hide()
		split.SetOffset(1)
	} else {
		split.SetOffset(0.65) // 0.65 was 0.7 by default
		sidebarScroll.Show()
		prevoiusImage = path
	}
	imageContainer.Refresh()
	// tagDisplay.Refresh()
	// sidebar.Refresh()
}

func truncateFilename(filename string, maxLength int, showExt bool) string {
	// if hidden file don't truncate it
	if string(filename[0]) == "." {
		return filename
	}
	// get the file extension
	ext := filepath.Ext(filename)
	// get the filename without extension
	nameWithoutExt := filename[:len(filename)-len(ext)]
	// if filename without extension is bigger or equal to maxLength, return filename with extension
	if len(nameWithoutExt) > maxLength {

		if showExt {
			return nameWithoutExt[:maxLength] + ".." + ext
		}
		return nameWithoutExt[:maxLength] + "..."

	} else {
		return filename
	}
}

func truncateDirname(dirname string, maxLength int) string {
	// if is hidden directory don't truncate it
	if string(dirname[0]) == "." {
		// if len(dirname) >= maxLength {
		// 	return dirname[:maxLength]
		// }
		return dirname
	}

	// if normal directory
	if len(dirname) > maxLength {
		return dirname[:maxLength] + "..."
	}
	return dirname
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

	switch filepath.Ext(path) {
	case ".jpg", ".jpeg":
		img, _, err = image.Decode(file)
	case ".png":
		img, _, err = image.Decode(file)
	case ".bmp":
		img, err = png.Decode(file)
	case ".webp":
		img, err = webp.Decode(file)
	case ".heic":
		img, err = goheif.Decode(file)
	case ".avif":
		img, err = avif.Decode(file)
	case ".qoi":
		img, err = qoi.Decode(file)
	case ".tiff", ".tif":
		img, err = tiff.Decode(file)
	case ".svg":
		img, err = svg.Decode(file)
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
	case ".bmp":
		// err = bmp.Encode(&buf, thumbImg)
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".webp":
		// err = chaiWebp.Encode(&buf, thumbImg, &chaiWebp.Options{Lossless: true})
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".heic":
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".avif":
		// avif.Encode(&buf, thumbImg, avif.Options{Quality: 85})
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
		// os.Exit(1)
	case ".qoi":
		// err = qoi.Encode(&buf, thumbImg)
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".tiff", ".tif":
		// err = tiff.Encode(&buf, thumbImg, &tiff.Options{Compression: tiff.Deflate})
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".svg":
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
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
	var img image.Image
	// test:= image.Decode()
	switch filepath.Ext(path) {
	case ".jpg", ".jpeg":
		img, _, err = image.Decode(file)
	case ".png":
		img, _, err = image.Decode(file)
	case ".gif":
		img, err = gif.Decode(file)
	case ".bmp":
		img, err = bmp.Decode(file)
	case ".webp":
		img, err = webp.Decode(file)
	case ".heic":
		img, err = goheif.Decode(file)
	case ".avif":
		img, err = avif.Decode(file)
	case ".qoi":
		img, err = qoi.Decode(file)
	case ".tiff", ".tif":
		img, err = tiff.Decode(file)
	case ".svg":
		img, err = svg.Decode(file)
	default:
		return nil, fmt.Errorf("unsupported image format")
	}
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
		// err = gif.Encode(&buf, thumbImg, &gif.Options{NumColors: 256})
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".bmp":
		// img, err = png.Decode(file)
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".webp":
		// img, err = webp.Decode(file)
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".heic":
		// img, err = goheif.Decode(file)
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".avif":
		// img, err = avif.Decode(file)
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".qoi":
		// img, err = qoi.Decode(file)
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".tiff", ".tif":
		// img, err = tiff.Decode(file)
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
	case ".svg":
		// img, err = svg.Decode(file)
		err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85})
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
	imageContainer := container.NewAdaptiveGrid(5)
	content.Add(imageContainer)

	for _, path := range imagePaths {
		displayImage(db, w, path, imageContainer, sidebar, sidebarScroll, split, a)
	}

	content.Refresh()
}
