package utilwindows

import (
	"archive/tar"
	"database/sql"
	"image/color"
	"io"
	"main/goexport/apptheme"
	"main/goexport/colorutils"
	"main/goexport/options"
	"os"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/dsnet/compress/bzip2"
)

const LAYOUT = "02-01-2006"

func ShowThemeEditorWindow(app fyne.App, currentTheme fyne.Theme, w fyne.Window, opts *options.Options) {
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
		colorValue := apptheme.GetThemeColor(currentTheme, prop)
		colorPreview := canvas.NewRectangle(colorValue)
		colorPreview.CornerRadius = 5
		colorPreview.SetMinSize(fyne.NewSize(35, 30))

		// Store the color preview in the map
		colorPreviews[prop] = colorPreview

		changeColorButton := widget.NewButton("Change Color", func() {
			ShowColorPickerWindow(prop, colorPreview, currentTheme, app, window, *opts)
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

func ShowColorPickerWindow(propertyName string, colorPreview *canvas.Rectangle, currentTheme fyne.Theme, a fyne.App, w fyne.Window, opts options.Options) {
	colorPickerWindow := a.NewWindow("Color Picker")
	colorPickerWindow.SetTitle("Color picker")

	colorPreviewRect := canvas.NewRectangle(color.NRGBA{0, 0, 130, 255})
	colorPreviewRect.SetMinSize(fyne.NewSize(64, 128))
	colorPreviewRect.CornerRadius = 5

	var content *fyne.Container
	var updateColor func()

	if opts.UseRGB {
		r, g, b := widget.NewSlider(0, 255), widget.NewSlider(0, 255), widget.NewSlider(0, 255)
		updateColor = func() {
			newColor := color.NRGBA{uint8(r.Value), uint8(g.Value), uint8(b.Value), 255}
			colorPreviewRect.FillColor = newColor
			colorPreview.FillColor = newColor
			// doesn't work
			// setThemeColor(currentTheme, propertyName, newColor)
			// apptheme.SetThemeColor(currentTheme, propertyName, newColor)
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

// Add a settings window
func ShowSettingsWindow(a fyne.App, parent fyne.Window, db *sql.DB, opts *options.Options) {
	settingsWindow := a.NewWindow("Settings")

	// Create a form for database path
	dbPathEntry := widget.NewEntry()
	dbPathEntry.SetText(opts.DatabasePath) // Set current path

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
			return len(opts.ExcludedDirs)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Excluded directory")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			for excluded := range opts.ExcludedDirs {
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

	timeZone := widget.NewLabel("Timezone in UTC: UTC" + strconv.Itoa(opts.Timezone))
	if opts.Timezone > 0 {
		timeZone = widget.NewLabel("Timezone in UTC: UTC+" + strconv.Itoa(opts.Timezone))
	} else {
		timeZone = widget.NewLabel("Timezone in UTC: UTC" + strconv.Itoa(opts.Timezone))
	}

	saveOptionsButton := widget.NewButton("Save Options", func() {
		err := options.SaveOptionsToDB(db, opts)
		if err == nil {
			dialog.ShowInformation("Success", "Options saved successfully", settingsWindow)
		} else {
			dialog.ShowError(err, settingsWindow)
		}
	})

	// Create a button to open the theme editor
	// themeEditorButton := widget.NewButton("Theme Editor", func() {
	// 	ShowThemeEditorWindow(a, apptheme.DefaultTheme{}, parent, opts)
	// })

	// Create a container for the settings content
	content := container.NewVBox(
		dbPathForm,
		widget.NewLabel("Excluded directories"),
		blackList,
		widget.NewLabel("Tags"),
		tagList,
		timeZone,
		// themeEditorButton,
		widget.NewLabel("Default sorting: Date Added, Descending"),
		saveOptionsButton,
	)

	settingsWindow.SetContent(content)
	settingsWindow.Resize(fyne.NewSize(400, 300))
	settingsWindow.Show()
}

func ShowChooseDirWindow(a fyne.App, finalUrl string) {
	chooseDirWindow := a.NewWindow("Choose a directory where your pictures are stored")
	content := container.NewVBox(
		widget.NewLabel("Choose a directory where your pictures are stored"),
	)

	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err == nil {
			if uri.Scheme() == "file" {
				finalUrl = uri.Path()
				chooseDirWindow.Close()
			} else {
				finalUrl = uri.String()
				chooseDirWindow.Close()
			}
		}
	}, chooseDirWindow)

	chooseDirWindow.SetContent(content)
	chooseDirWindow.Resize(fyne.NewSize(515, 380))
	chooseDirWindow.Show()
}

func ShowRightClickMenu(w fyne.Window, fileList []string) {
	// Create the menu
	home, _ := os.UserHomeDir()
	now := time.Now()
	formattedDate := now.Format("02-01-2006")
	// fmt.Println(formattedDate)

	archiveButton := widget.NewButton("Add to Archive", func() {
		archive, err := os.Create(home + "/Desktop/" + formattedDate + ".tar.bz2")
		if err != nil {
			dialog.ShowError(err, w)
		}
		defer archive.Close()

		bz2Writer, err := bzip2.NewWriter(archive, &bzip2.WriterConfig{
			Level: bzip2.BestCompression,
		})
		if err != nil {
			dialog.ShowError(err, w)
		}
		defer bz2Writer.Close()

		tarWriter := tar.NewWriter(bz2Writer)
		defer tarWriter.Close()

		for _, filePath := range fileList {
			err := AddFileToArchive(filePath, tarWriter)
			if err != nil {
				dialog.ShowError(err, w)
			}
		}
	})

	content := container.NewVBox(archiveButton)

	// Show the menu
	dialog.ShowCustom("Right Click", "Close", content, w)
}

func AddFileToArchive(filePath string, tarWriter *tar.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileStat, err := file.Stat()
	if err != nil {
		return err
	}
	defer file.Close()

	header := &tar.Header{
		Name:    filePath,
		Mode:    int64(fileStat.Mode()),
		ModTime: fileStat.ModTime(),
		Size:    fileStat.Size(),
	}

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return err
	}

	return nil
}
