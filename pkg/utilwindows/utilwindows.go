package utilwindows

import (
	"database/sql"
	"fmt"
	"image/color"
	"log"
	"main/pkg/apptheme"
	"main/pkg/archives"
	"main/pkg/colorutils"
	"main/pkg/database"
	"main/pkg/options"
	"main/pkg/tagwindow"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

func ShowAllTagWindow(a fyne.App, parent fyne.Window, db *sql.DB, opts *options.Options) {
	tagEditWindow := a.NewWindow("Show Tags")

	content := container.NewVBox()
	scroll := container.NewVScroll(content)
	tags, _ := database.GetTags(db)

	if len(tags) == 0 {
		content.Add(widget.NewLabel("No tags found."))
		tagEditWindow.SetContent(scroll)
		tagEditWindow.Resize(fyne.NewSize(300, 450))
		tagEditWindow.Show()
		return
	}

	for k, v := range tags {
		// fmt.Println("Tag Key value: ", k, " ", v)
		content.Add(
			container.NewHBox(
				widget.NewLabel(v),
				widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
					tagwindow.ShowCreateTagWindow(a, parent, db, opts, true, v, k)
					// ShowEditTagWindow(a, parent, db, i, scroll) // replace this with create tag window but update statement
				}),
			),
		)
	}

	tagEditWindow.SetContent(scroll)
	tagEditWindow.Resize(fyne.NewSize(300, 450))
	tagEditWindow.Show()
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
	// optimized
	// Create a slice of excluded directories
	excludedDirs := make([]string, 0, len(opts.ExcludedDirs))
	for dir := range opts.ExcludedDirs {
		excludedDirs = append(excludedDirs, dir)
	}

	// Create a list of all excluded directories
	blackList := widget.NewList(
		func() int {
			return len(excludedDirs)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			label := item.(*widget.Label)
			label.SetText(excludedDirs[id])
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

	// Create a label for the timezone
	var timeZone *widget.Label
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
		widget.NewButton("Show Tags", func() {
			ShowAllTagWindow(a, parent, db, opts)
		}),
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

func ShowChooseDirWindow(a fyne.App, opts *options.Options, logger *log.Logger, db *sql.DB) {
	chooseDirWindow := a.NewWindow("Choose directories you want to exclude from scanning")

	var selectedDirs []string

	content := container.NewVBox()

	updateContent := func() {
		content.Objects = nil
		for _, dir := range selectedDirs {
			label := widget.NewLabel(dir)
			content.Add(label)
		}
		content.Refresh()
	}

	scroll := container.NewScroll(content)

	chooseButton := widget.NewButton("Choose Directory", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil {
				path := uri.Path()
				if uri.Scheme() == "file" {
					opts.ExcludedDirs[path] = 1
					selectedDirs = append(selectedDirs, path)
					logger.Println("Added", path, "to excluded directories.")
				} else {
					opts.ExcludedDirs[uri.String()] = 1
					selectedDirs = append(selectedDirs, uri.String())
				}
				updateContent()
			}
		}, chooseDirWindow)
	})

	doneButton := widget.NewButton("Done", func() {
		err := options.SaveOptionsToDB(db, opts)
		if err != nil {
			logger.Println("Failed to save Options: ", err)
		}
		chooseDirWindow.Close()
	})

	chooseDirWindow.SetContent(container.NewBorder(chooseButton, doneButton, nil, nil, scroll))
	chooseDirWindow.Resize(fyne.NewSize(515, 380))
	chooseDirWindow.Show()
}

func mapToStringSlice(m map[string]bool) []string {
	slice := make([]string, 0, len(m))
	for k := range m {
		slice = append(slice, k)
	}
	return slice
}

func ShowRightClickMenu(w fyne.Window, fileList map[string]bool, a fyne.App) {
	home, _ := os.UserHomeDir()
	now := time.Now()
	formattedDate := now.Format("02-01-2006")

	listedFiles := mapToStringSlice(fileList)

	gzipButton := widget.NewButton("Create Gzip Archive", func() {
		archivePath := filepath.Join(home, "Desktop", formattedDate+".tar.gz")
		err := archives.CreateTarGzipArchive(archivePath, listedFiles, w)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Success", fmt.Sprintf("Archive created successfully at %s", archivePath), w)
		}
	})

	bzip2Button := widget.NewButton("Create Bzip2 Archive", func() {
		archivePath := filepath.Join(home, "Desktop", formattedDate+".tar.bz2")
		err := archives.CreateTarBzip2Archive(archivePath, listedFiles, w)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Success", fmt.Sprintf("Archive created successfully at %s", archivePath), w)
		}
	})

	zipButton := widget.NewButton("Create Zip Archive", func() {
		archivePath := filepath.Join(home, "Desktop", formattedDate+".zip")
		err := archives.CreateZipArchive(archivePath, listedFiles, w)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Success", fmt.Sprintf("Archive created successfully at %s", archivePath), w)
		}
	})

	encryptedButton := widget.NewButton("Create Encrypted Archive", func() {
		showPasswordWindow(a, formattedDate, listedFiles, w)
	})

	convertButton := widget.NewButton("Convert Files", func() {
		showChooseConvertDir(a, w)
	})

	content := container.NewVBox(
		convertButton,
		gzipButton,
		bzip2Button,
		zipButton,
		encryptedButton,
	)
	dialog.ShowCustom("File Actions", "Close", content, w)
}

func showChooseConvertDir(a fyne.App, w fyne.Window) {
	home, _ := os.UserHomeDir()
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err == nil {
			path := uri.Path()
			if uri.Scheme() == "file" {
				convertedPath := filepath.Join(home, "Desktop", path)
				convertedPath = filepath.Clean(convertedPath)
				fmt.Println("Converted Path: ", convertedPath)
			}
		}
	}, w)
}

func showPasswordWindow(a fyne.App, fmtDate string, fileList []string, tagVaultWindow fyne.Window) {
	passwordWindow := a.NewWindow("Enter Password")
	label := widget.NewLabel("Enter Password:")
	password := widget.NewEntry()
	password.OnSubmitted = func(password string) {
		archives.ArchivePassword = password
		showChooseArchiveType(tagVaultWindow, fmtDate, fileList)
		passwordWindow.Close()
	}
	container := container.NewVBox(label, password)
	passwordWindow.SetContent(container)
	passwordWindow.Resize(fyne.NewSize(300, 100))
	passwordWindow.Show()
}

func showChooseArchiveType(w fyne.Window, formattedDate string, fileList []string) {
	home, _ := os.UserHomeDir()
	gzipButton := widget.NewButton("Gzip Archive", func() {
		archivePath := filepath.Join(home, "Desktop", formattedDate+".tar.gz")
		archives.CreateEncryptedTarGzipArchive(archivePath, fileList, w)
	})
	bzip2Button := widget.NewButton("Bzip2 Archive", func() {
		archivePath := filepath.Join(home, "Desktop", formattedDate+".tar.bz2")
		archives.CreateEncryptedTarBzip2Archive(archivePath, fileList, w)
	})
	zipButton := widget.NewButton("Zip Archive", func() {
		archivePath := filepath.Join(home, "Desktop", formattedDate+".zip")
		archives.CreateEncryptedZipArchive(archivePath, fileList, w)
	})

	content := container.NewVBox(
		gzipButton,
		bzip2Button,
		zipButton,
	)
	dialog.ShowCustom("Choose Archive Type", "Close", content, w)
}
