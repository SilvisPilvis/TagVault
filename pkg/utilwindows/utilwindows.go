package utilwindows

import (
	"archive/tar"
	// "archive/zip"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"image/color"
	"io"
	"log"
	"main/pkg/apptheme"
	"main/pkg/colorutils"
	"main/pkg/options"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/dsnet/compress/bzip2"

	"github.com/alexmullins/zip"
)

const LAYOUT = "02-01-2006"

var archivePassword string

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

func ShowRightClickMenu(w fyne.Window, fileList []string, a fyne.App) {
	home, _ := os.UserHomeDir()
	now := time.Now()
	formattedDate := now.Format("02-01-2006")

	gzipButton := widget.NewButton("Create Gzip Archive", func() {
		archivePath := filepath.Join(home, "Desktop", formattedDate+".tar.gz")
		err := createTarGzipArchive(archivePath, fileList, w)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Success", fmt.Sprintf("Archive created successfully at %s", archivePath), w)
		}
	})

	bzip2Button := widget.NewButton("Create Bzip2 Archive", func() {
		archivePath := filepath.Join(home, "Desktop", formattedDate+".tar.bz2")
		err := createTarBzip2Archive(archivePath, fileList, w)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Success", fmt.Sprintf("Archive created successfully at %s", archivePath), w)
		}
	})

	zipButton := widget.NewButton("Create Zip Archive", func() {
		archivePath := filepath.Join(home, "Desktop", formattedDate+".zip")
		err := createZipArchive(archivePath, fileList, w)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Success", fmt.Sprintf("Archive created successfully at %s", archivePath), w)
		}
	})

	encryptedButton := widget.NewButton("Create Encrypted Archive", func() {
		showPasswordWindow(a, formattedDate, fileList, w)
	})

	content := container.NewVBox(
		gzipButton,
		bzip2Button,
		zipButton,
		encryptedButton,
	)
	dialog.ShowCustom("File Actions", "Close", content, w)
}

func showPasswordWindow(a fyne.App, fmtDate string, fileList []string, tagVaultWindow fyne.Window) {
	passwordWindow := a.NewWindow("Enter Password")
	label := widget.NewLabel("Enter Password:")
	password := widget.NewEntry()
	password.OnSubmitted = func(password string) {
		archivePassword = password
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
		createEncryptedTarGzipArchive(archivePath, fileList, w)
	})
	bzip2Button := widget.NewButton("Bzip2 Archive", func() {
		archivePath := filepath.Join(home, "Desktop", formattedDate+".tar.bz2")
		createEncryptedTarBzip2Archive(archivePath, fileList, w)

	})
	zipButton := widget.NewButton("Zip Archive", func() {
		archivePath := filepath.Join(home, "Desktop", formattedDate+".zip")
		createEncryptedZipArchive(archivePath, fileList, w)

	})

	content := container.NewVBox(
		gzipButton,
		bzip2Button,
		zipButton,
	)
	dialog.ShowCustom("Choose Archive Type", "Close", content, w)
}

func createTarBzip2Archive(archivePath string, fileList []string, w fyne.Window) error {
	if len(fileList) <= 1 {
		dialog.ShowError(errors.New("no files to archive"), w)
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	bzipWriter, err := bzip2.NewWriter(archive, &bzip2.WriterConfig{
		Level: bzip2.BestCompression,
	})
	if err != nil {
		dialog.ShowError(err, nil)
	}
	defer bzipWriter.Close() // Ensure bzipWriter is closed

	tarWriter := tar.NewWriter(bzipWriter)
	defer tarWriter.Close() // Ensure tarWriter is closed

	for _, filePath := range fileList {
		err := addFileToTarArchive(filePath, tarWriter)
		if err != nil {
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	// Ensure tarWriter is closed properly
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Ensure gzipWriter/bzip2Writer is closed properly
	if err := bzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Ensure archive is closed properly
	if err := archive.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %w", err)
	}

	// Verify the archive is not empty
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive file: %w", err)
	}

	fmt.Printf("Archive created successfully at %s with size %d bytes\n", archivePath, info.Size())
	return nil
}

func createTarGzipArchive(archivePath string, fileList []string, w fyne.Window) error {
	if len(fileList) <= 1 {
		dialog.ShowError(errors.New("no files to archive"), w)
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	gzipWriter := gzip.NewWriter(archive)
	defer gzipWriter.Close() // Ensure gzipWriter is closed to finalize the archive

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close() // Ensure tarWriter is closed

	for _, filePath := range fileList {
		err := addFileToTarArchive(filePath, tarWriter)
		if err != nil {
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	// Ensure tarWriter is closed properly
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Ensure gzipWriter/bzip2Writer is closed properly
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Ensure archive is closed properly
	if err := archive.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %w", err)
	}

	// Verify the archive is not empty
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive file: %w", err)
	}

	fmt.Printf("Archive created successfully at %s with size %d bytes\n", archivePath, info.Size())
	return nil
}

func createZipArchive(archivePath string, fileList []string, w fyne.Window) error {
	if len(fileList) <= 1 {
		dialog.ShowError(errors.New("no files to archive"), w)
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close() // Ensure gzipWriter is closed to finalize the archive

	for _, filePath := range fileList {
		err := addFileZipToArchive(filePath, zipWriter)
		if err != nil {
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	// Ensure zipWriter is closed properly
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Ensure archive is closed properly
	if err := archive.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %w", err)
	}

	// Verify the archive is not empty
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive file: %w", err)
	}

	fmt.Printf("Archive created successfully at %s with size %d bytes\n", archivePath, info.Size())
	return nil
}

func createEncryptedZipArchive(archivePath string, fileList []string, w fyne.Window) error {
	if len(fileList) <= 1 {
		dialog.ShowError(errors.New("no files to archive"), w)
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close() // Ensure gzipWriter is closed to finalize the archive

	for _, filePath := range fileList {
		err := addEncryptedFileZipToArchive(filePath, zipWriter)
		if err != nil {
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	// Ensure zipWriter is closed properly
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Ensure archive is closed properly
	if err := archive.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %w", err)
	}

	// Verify the archive is not empty
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive file: %w", err)
	}

	fmt.Printf("Archive created successfully at %s with size %d bytes\n", archivePath, info.Size())
	return nil
}

func createEncryptedTarBzip2Archive(archivePath string, fileList []string, w fyne.Window) error {
	if len(fileList) <= 1 {
		dialog.ShowError(errors.New("no files to archive"), w)
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	bzipWriter, err := bzip2.NewWriter(archive, &bzip2.WriterConfig{
		Level: bzip2.BestCompression,
	})
	if err != nil {
		dialog.ShowError(err, nil)
	}
	defer bzipWriter.Close() // Ensure bzipWriter is closed

	tarWriter := tar.NewWriter(bzipWriter)
	defer tarWriter.Close() // Ensure tarWriter is closed

	for _, filePath := range fileList {
		err := addEncryptedFileToTarArchive(filePath, tarWriter)
		if err != nil {
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	// Ensure tarWriter is closed properly
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Ensure gzipWriter/bzip2Writer is closed properly
	if err := bzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Ensure archive is closed properly
	if err := archive.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %w", err)
	}

	// Verify the archive is not empty
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive file: %w", err)
	}

	fmt.Printf("Archive created successfully at %s with size %d bytes\n", archivePath, info.Size())
	return nil
}

func createEncryptedTarGzipArchive(archivePath string, fileList []string, w fyne.Window) error {
	if len(fileList) <= 1 {
		dialog.ShowError(errors.New("no files to archive"), w)
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	gzipWriter := gzip.NewWriter(archive)
	defer gzipWriter.Close() // Ensure gzipWriter is closed to finalize the archive

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close() // Ensure tarWriter is closed

	for _, filePath := range fileList {
		err := addEncryptedFileToTarArchive(filePath, tarWriter)
		if err != nil {
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	// Ensure tarWriter is closed properly
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Ensure gzipWriter/bzip2Writer is closed properly
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Ensure archive is closed properly
	if err := archive.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %w", err)
	}

	// Verify the archive is not empty
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive file: %w", err)
	}

	fmt.Printf("Archive created successfully at %s with size %d bytes\n", archivePath, info.Size())
	return nil
}

func addFileToTarArchive(filePath string, tarWriter *tar.Writer) error {

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return fmt.Errorf("failed to create tar header for %s: %w", filePath, err)
	}

	header.Name = filepath.Base(filePath)

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header for %s: %w", filePath, err)
	}

	_, err = io.Copy(tarWriter, file) // you can replace _ with bytes and uncoment the print below to see info
	if err != nil {
		return fmt.Errorf("failed to write file content for %s: %w", filePath, err)
	}

	// fmt.Printf("Added %s to archive (size: %d bytes)\n", filePath, bytesWritten)
	return nil
}

func addFileZipToArchive(filePath string, zipWriter *zip.Writer) error {

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("failed to create tar header for %s: %w", filePath, err)
	}

	header.Name = filepath.Base(filePath)

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to write zip header for %s: %w", filePath, err)
	}

	_, err = io.Copy(writer, file) // you can replace _ with bytes and uncoment the print below to see info
	if err != nil {
		return fmt.Errorf("failed to write file content for %s: %w", filePath, err)
	}

	// fmt.Printf("Added %s to archive (size: %d bytes)\n", filePath, bytesWritten)
	return nil
}

func addEncryptedFileZipToArchive(filePath string, zipWriter *zip.Writer) error {

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("failed to create tar header for %s: %w", filePath, err)
	}
	header.SetPassword(archivePassword)

	header.Name = filepath.Base(filePath)

	// --- Normal Zip Section Start
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to write zip header for %s: %w", filePath, err)
	}

	_, err = io.Copy(writer, file) // you can replace _ with bytes and uncoment the print below to see info
	if err != nil {
		return fmt.Errorf("failed to write file content for %s: %w", filePath, err)
	}

	// fmt.Printf("Added %s to archive (size: %d bytes)\n", filePath, bytesWritten)
	return nil
	// --- Normal Zip Section End

	// hasher := sha256.New()
	// hasher.Write([]byte(archivePassword))
	// hashedPass := sha256.Sum256([]byte(archivePassword))

	// block, err := aes.NewCipher(hashedPass[:])
	// if err != nil {
	// 	return fmt.Errorf("failed to create cipher: %w", err)
	// }

	// gcm, err := cipher.NewGCM(block)
	// if err != nil {
	// 	return fmt.Errorf("failed to create GCM: %w", err)
	// }

	// nonce := make([]byte, gcm.NonceSize())
	// if _, err = rand.Read(nonce); err != nil {
	// 	return fmt.Errorf("failed to generate nonce: %w", err)
	// }

	// fileBytes, err := io.ReadAll(file)
	// if err != nil {
	// 	return fmt.Errorf("failed to read file content for %s: %w", filePath, err)
	// }

	// cipherText := gcm.Seal(nil, nonce, fileBytes, nil)

	// fullEncryptedContent := append(nonce, cipherText...)

	// encryptedWriter, err := zipWriter.CreateHeader(header)
	// if err != nil {
	// 	return fmt.Errorf("failed to write zip header for %s: %w", filePath, err)
	// }

	// _, err = encryptedWriter.Write(fullEncryptedContent)
	// if err != nil {
	// 	return fmt.Errorf("failed to write file content for %s: %w", filePath, err)
	// }

	// return nil
}

func addEncryptedFileToTarArchive(filePath string, tarWriter *tar.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return fmt.Errorf("failed to create tar header for %s: %w", filePath, err)
	}

	header.Name = filepath.Base(filePath)

	hasher := sha256.New()
	hasher.Write([]byte(archivePassword))
	hashedPass := sha256.Sum256([]byte(archivePassword))

	block, err := aes.NewCipher(hashedPass[:])
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file content for %s: %w", filePath, err)
	}

	cipherText := gcm.Seal(nil, nonce, fileBytes, nil)

	fullEncryptedContent := append(nonce, cipherText...)

	// Update the size in the header to match the encrypted content
	header.Size = int64(len(fullEncryptedContent))

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header for %s: %w", filePath, err)
	}

	_, err = tarWriter.Write(fullEncryptedContent)
	if err != nil {
		return fmt.Errorf("failed to write encrypted file content for %s: %w", filePath, err)
	}

	// fmt.Printf("Added %s to archive (size: %d bytes)\n", filePath, bytesWritten)

	return nil
}