package tagwindow

import (
	"database/sql"
	"fmt"
	"image/color"
	"log"
	"main/pkg/colorutils"
	"main/pkg/database"
	"main/pkg/options"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ShowCreateTagWindow(a fyne.App, parent fyne.Window, db *sql.DB, opts *options.Options, edit bool, tag string, tagId int) {

	tagWindow := a.NewWindow("Create Tag")

	tagHex, err := database.GetTagColorById(db, tagId)
	if err != nil {
		log.Fatal(err)
	}
	tagColor, _ := colorutils.HexToColor(tagHex)
	if edit {
		tagWindow.SetTitle("Edit a Tag")
	}

	var colorPreviewRect *canvas.Rectangle

	if edit {
		colorPreviewRect = canvas.NewRectangle(tagColor)
	} else {
		colorPreviewRect = canvas.NewRectangle(color.NRGBA{0, 0, 130, 255})
	}
	colorPreviewRect.SetMinSize(fyne.NewSize(64, 128))
	colorPreviewRect.CornerRadius = 5

	stringInput := widget.NewEntry()
	if edit {
		stringInput.SetText(tag)
	}
	stringInput.SetPlaceHolder("Enter Tag name")

	var content *fyne.Container
	var updateColor func()
	var getHexColor func() string

	if opts.UseRGB {
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
		if edit {
			rc, gc, bc := colorutils.HexToRgb(tagHex)
			r.SetValue(rc)
			g.SetValue(gc)
			b.SetValue(bc)
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
		if edit {
			hc, sc, vc := colorutils.HexToHSV(tagHex)
			h.SetValue(hc * 360)
			s.SetValue(sc * 100)
			v.SetValue(vc * 100)
			updateColor()
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

	updateButton := widget.NewButton("Edit Tag", func() {
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
	if edit {
		content.Add(updateButton)
	} else {
		content.Add(createButton)
	}

	tagWindow.SetContent(content)
	tagWindow.Resize(fyne.NewSize(300, 400))
	tagWindow.Show()

	updateColor() // Initial color update
}

func ShowTagWindow(a fyne.App, parent fyne.Window, db *sql.DB, imgId int, tagList *fyne.Container) {
	tagWindow := a.NewWindow("Tags")
	tagWindow.SetTitle("Add a Tag")

	content := container.NewGridWithColumns(4)
	loadingLabel := widget.NewLabel("Loading tags...")
	content.Add(loadingLabel)

	tagWindow.SetContent(container.NewVScroll(content))
	tagWindow.Resize(fyne.NewSize(300, 200))
	tagWindow.Show()

	go func() {
		tags, err := db.Query("SELECT id, name, color FROM Tag WHERE id NOT IN (SELECT tagId FROM FileTag WHERE fileId = ?)", imgId)
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
					_, err := db.Exec("INSERT OR IGNORE INTO FileTag (fileId, tagId) VALUES (?, ?)", imgId, tagID)
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

// Modify the createTagDisplay function to include tag removal functionality
func CreateTagDisplay(db *sql.DB, imageId int, appLogger *log.Logger) *fyne.Container {
	tagDisplay := container.NewAdaptiveGrid(3)

	rows, err := db.Query("SELECT Tag.id, Tag.name, Tag.color FROM FileTag INNER JOIN Tag ON FileTag.tagId = Tag.id WHERE FileTag.fileId = ?", imageId)
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
