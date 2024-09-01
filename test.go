package main

import (
	// "main/db" // for sqlite tag storage

	"database/sql" // for sql type
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"

	"runtime"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/grafana/pyroscope-go"
	_ "github.com/mattn/go-sqlite3"
)

type imageButton struct {
	widget.BaseWidget
	onTapped func()
	image    *canvas.Image
}

type Tag struct {
	Name  string
	Value string
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
	b.onTapped()
}

func main() {
	// start logger
	logger := log.New(os.Stdout, "", 0)

	logger.Print("Load only visible images\n")
	logger.Print("Fix sqlite insert ignore\n")
	logger.Print("When adding tags show tags that aren't on the image\n")

	profiling := false

	if profiling {
		logger.Print("Starting Pyroscope")
		runtime.SetMutexProfileFraction(5)
		runtime.SetBlockProfileRate(5)
		pyroscope.Start(pyroscope.Config{
			ApplicationName: "explorer.golang.app",

			// replace this with the address of pyroscope server
			ServerAddress: "http://192.168.101.30:4040",

			// you can disable logging by setting this to nil
			Logger: pyroscope.StandardLogger,

			// you can provide static tags via a map:
			Tags: map[string]string{"hostname": os.Getenv("HOSTNAME")},

			ProfileTypes: []pyroscope.ProfileType{
				// these profile types are enabled by default:
				pyroscope.ProfileCPU,
				pyroscope.ProfileAllocObjects,
				pyroscope.ProfileAllocSpace,
				pyroscope.ProfileInuseObjects,
				pyroscope.ProfileInuseSpace,

				// these profile types are optional:
				pyroscope.ProfileGoroutines,
				pyroscope.ProfileMutexCount,
				pyroscope.ProfileMutexDuration,
				pyroscope.ProfileBlockCount,
				pyroscope.ProfileBlockDuration,
			},
		})
	}

	Db, err := sql.Open("sqlite3", "file:./index.db")
	if err != nil {
		panic(err.Error())
	}
	Db.SetMaxOpenConns(1)
	defer Db.Close()

	// check the connection
	err = Db.Ping()
	if err != nil {
		log.Print("Not Connected to db!\n")
		log.Fatal(err.Error(), "\n")
	}
	log.Print("Connected to db!\n")
	// start sqlite db driver
	db := Db
	// db := db.Db
	// db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("CREATE TABLE IF NOT EXISTS `Tag`(`id` INTEGER PRIMARY KEY NOT NULL, `name` VARCHAR(255) NOT NULL, `color` VARCHAR(7) NOT NULL);")
	db.Exec("CREATE TABLE IF NOT EXISTS `Image`(`id` INTEGER PRIMARY KEY NOT NULL, `path` VARCHAR(1024) NOT NULL);")
	db.Exec("CREATE TABLE `ImageTag`(`imageId` INTEGER NOT NULL, `tagId` INTEGER NOT NULL);")
	// db.Close()
	// db.Exec("PRAGMA foreign_keys=ON")

	// create new app
	a := app.New()

	// load icon from image
	icon, err := fyne.LoadResourceFromPath("app.png")
	if err != nil {
		panic(err)
	}

	// create new window
	w := a.NewWindow("File Explorer")
	w.Resize(fyne.NewSize(1000, 600))

	// set window and app icon
	a.SetIcon(icon)
	w.SetIcon(icon)

	content := container.NewVBox()
	scroll := container.NewScroll(content)

	sidebar := container.NewVBox()
	sidebarScroll := container.NewScroll(sidebar)
	sidebarScroll.Hide()

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter directory path")

	split := container.NewHSplit(scroll, sidebarScroll)
	split.Offset = 1 // Start with sidebar hidden

	var resourceCache sync.Map

	displayImages := func(dir string) {
		files, err := os.ReadDir(dir)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		imageContainer := container.NewAdaptiveGrid(4)
		content.Add(imageContainer)

		for _, file := range files {
			if !file.IsDir() && isImageFile(file.Name()) {
				imgPath := filepath.Join(dir, file.Name())

				// Use a placeholder image initially
				placeholderResource := fyne.NewStaticResource("placeholder", []byte{})
				imgButton := newImageButton(placeholderResource, nil)

				truncatedName := truncateFilename(file.Name(), 10)
				// logger.Print(imgPath + "\n")
				db.Exec("INSERT IGNORE INTO Image (path) VALUES (?)", imgPath)
				label := widget.NewLabel(truncatedName)

				imgContainer := container.New(layout.NewVBoxLayout(), imgButton, label)
				imageContainer.Add(imgContainer)

				// Load the actual image asynchronously
				go func(path string, button *imageButton) {
					var resource fyne.Resource
					if cachedResource, ok := resourceCache.Load(path); ok {
						resource = cachedResource.(fyne.Resource)
					} else {
						var err error
						resource, err = fyne.LoadResourceFromPath(path)
						if err != nil {
							dialog.ShowError(err, w)
							return
						}
						resourceCache.Store(path, resource)
					}

					button.image.Resource = resource
					button.Refresh()

					// on click add image to sidebar
					button.onTapped = func() {
						sidebar.RemoveAll()

						fullImg := canvas.NewImageFromResource(resource)
						fullImg.FillMode = canvas.ImageFillContain
						fullImg.SetMinSize(fyne.NewSize(200, 200))

						fullLabel := widget.NewLabel(filepath.Base(path))
						fullLabel.Wrapping = fyne.TextWrapWord

						imageIdQuery := db.QueryRow("SELECT id FROM Image WHERE path = ?", path)
						imageId := 0
						err = imageIdQuery.Scan(&imageId)
						if err != nil {
							dialog.ShowError(err, w)
							return
						}

						imageTags, err := db.Query("SELECT Tag.name FROM ImageTag INNER JOIN Tag ON ImageTag.tagId = Tag.id WHERE ImageTag.imageId = ?", imageId)
						if err != nil {
							dialog.ShowError(err, w)
							return
						}

						tagDisplay := container.NewHBox()

						for imageTags.Next() {
							var tagName string
							err = imageTags.Scan(&tagName)
							if err != nil {
								dialog.ShowError(err, w)
								return
							}
							tagDisplay.Add(widget.NewButton(tagName, nil))
						}

						addTagButton := widget.NewButton("+", func() {
							showTagWindow(a, w, db, imageId)
						})

						createTagButton := widget.NewButton("Add Tag", func() {
							// createTagWindow(a, w)
							createTagWindow(a, w, db)
						})

						sidebar.Add(fullImg)
						sidebar.Add(fullLabel)
						sidebar.Add(tagDisplay)
						sidebar.Add(addTagButton)
						sidebar.Add(createTagButton)

						sidebarScroll.Show()
						split.Offset = 0.7 // Show sidebar
						sidebar.Refresh()
					}
				}(imgPath, imgButton)
			}
		}
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Directory", Widget: input},
		},
		OnSubmit: func() {
			content.RemoveAll()
			displayImages(input.Text)
		},
	}

	browseButton := widget.NewButton("Browse", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uri == nil {
				return
			}
			input.SetText(uri.Path())
			content.RemoveAll()
			displayImages(uri.Path())
		}, w)
	})

	controls := container.NewBorder(nil, nil, nil, browseButton, form)
	mainContainer := container.NewBorder(controls, nil, nil, nil, split)

	w.SetContent(mainContainer)
	w.ShowAndRun()
}

func showTagWindow(a fyne.App, parent fyne.Window, db *sql.DB, imgId int) {
	tagWindow := a.NewWindow("Add a Tag")

	// content := container.NewGridWrap(fyne.NewSize(300, 200))
	content := container.NewGridWithColumns(4)
	// content := container.NewVScroll()
	loadingLabel := widget.NewLabel("Loading tags...")
	content.Add(loadingLabel)
	// content.Add(widget.NewSeparator())

	tagWindow.SetContent(container.NewVScroll(content))
	tagWindow.Resize(fyne.NewSize(300, 200))
	tagWindow.Show()

	go func() {
		// should only load tags that aren't already on the image
		tags, err := db.Query("SELECT id, name FROM Tag;")
		if err != nil {
			parent.Canvas().Refresh(parent.Content())
			dialog.ShowError(err, parent)
			return
		}
		defer tags.Close()

		var buttons []*widget.Button
		for tags.Next() {
			var id int
			var name string
			err = tags.Scan(&id, &name)
			if err != nil {
				parent.Canvas().Refresh(parent.Content())
				dialog.ShowError(err, parent)
				return
			}

			button := widget.NewButton(name, nil)
			tagID := id // Create a new variable to avoid closure issues
			button.OnTapped = func() {
				go func() {
					_, err := db.Exec("INSERT INTO ImageTag (imageId, tagId) SELECT ?, ? WHERE NOT EXISTS (SELECT 1 FROM ImageTag WHERE imageId = ? AND tagId = ?);", imgId, tagID, imgId, tagID)
					// parent.Content().Refresh()
					if err != nil {
						dialog.ShowError(err, parent)
					} else {
						dialog.ShowInformation("Success", "Tag Added", parent)
						tagWindow.Close()
					}
				}()
			}
			// button.Resize(fyne.NewSize(50, 20))
			buttons = append(buttons, button)
		}

		// Update UI on the main thread
		tagWindow.Canvas().Refresh(content)
		content.Remove(loadingLabel)
		for _, button := range buttons {
			content.Add(button)
		}
		// content.Refresh()
	}()
	// parent.Content().Refresh()
}

func createTagWindow(a fyne.App, parent fyne.Window, db *sql.DB) {
	// func createTagWindow(a fyne.App, parent fyne.Window) {
	tagWindow := a.NewWindow("Create a Tag")

	var chosenColor color.Color = color.White
	colorButton := widget.NewButton("Choose Color", nil)
	stringInput := widget.NewEntry()
	stringInput.SetPlaceHolder("Enter tag name")

	updateColorButton := func(c color.Color) {
		chosenColor = c
		r, g, b, _ := c.RGBA()
		colorButton.SetText(fmt.Sprintf("Color: #%02X%02X%02X", uint8(r>>8), uint8(g>>8), uint8(b>>8)))
		// colorButton.
	}

	colorButton.OnTapped = func() {
		dialog.ShowColorPicker("Choose Tag Color", "Select a color for your tag", updateColorButton, tagWindow)
	}

	createButton := widget.NewButton("Create Tag", func() {
		tagName := stringInput.Text
		if tagName == "" {
			dialog.ShowInformation("Error", "Tag name cannot be empty", tagWindow)
			return
		}
		// // Here you would typically save the tag or do something with it
		r, g, b, _ := chosenColor.RGBA()
		hexColor := fmt.Sprintf("#%02X%02X%02X", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		db.Exec("INSERT INTO Tag (name, color) VALUES (?, ?)", tagName, hexColor)
		// db.Close()
		dialog.ShowInformation("Tag Created", "Tag Name: "+tagName+"\nColor: "+hexColor, tagWindow)
	})

	// fmt.Println(chosenColor)

	content := container.NewVBox(
		widget.NewLabel("Choose tag color:"),
		// canvas.NewRectangle(chosenColor), // should set the bg color of the button
		colorButton,
		widget.NewLabel("Enter tag name:"),
		stringInput,
		createButton,
	)

	tagWindow.SetContent(content)
	tagWindow.Resize(fyne.NewSize(300, 200))
	tagWindow.Show()
}

func colorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return string(r) + string(g) + string(b)
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif"
}

func truncateFilename(filename string, maxLength int) string {
	if len(filename) <= maxLength {
		return filename
	}
	ext := filepath.Ext(filename)
	nameWithoutExt := filename[:len(filename)-len(ext)]
	if len(ext) > maxLength-3 {
		return nameWithoutExt[:maxLength-3] + "..."
	}
	return nameWithoutExt[:maxLength-len(ext)-3] + "..." + ext
}
