package imagebutton

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// ImageButton represents a clickable image widget
type ImageButton struct {
	widget.BaseWidget
	Image        *canvas.Image
	onTapped     func()
	onLongTap    func()
	onRightClick func()
	pressedTime  time.Time
	longTapTimer *time.Timer
	Selected     bool
}

// NewImageButton creates a new image button from the specified resource
func NewImageButton(resource fyne.Resource) *ImageButton {
	img := &ImageButton{}
	img.ExtendBaseWidget(img)
	img.Image = canvas.NewImageFromResource(resource)
	img.Image.FillMode = canvas.ImageFillContain
	img.Image.SetMinSize(fyne.NewSize(150, 150))
	return img
}

// SetOnTapped sets the function to be called when the button is tapped
func (b *ImageButton) SetOnTapped(f func()) {
	b.onTapped = f
}

// SetOnLongTap sets the function to be called when the button is long-pressed
func (b *ImageButton) SetOnLongTap(f func()) {
	b.onLongTap = f
}

// SetOnRightClick sets the function to be called when the button is right-clicked
func (b *ImageButton) SetOnRightClick(f func()) {
	b.onRightClick = f
}

// Tapped handles the tap event
func (b *ImageButton) Tapped(_ *fyne.PointEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

// func (b *ImageButton) Tapped(me *desktop.MouseEvent) {
// 	if me.Button == desktop.MouseButtonPrimary {
// 		if b.onTapped != nil {
// 			b.onTapped()
// 		}
// 	}
// }

// TappedSecondary handles the right-click event
func (b *ImageButton) TappedSecondary(_ *fyne.PointEvent) {
	if b.onRightClick != nil {
		b.onRightClick()
	}
}

// MouseDown handles the mouse down event
func (b *ImageButton) MouseDown(me *desktop.MouseEvent) {
	if me.Button == desktop.MouseButtonPrimary {
		b.pressedTime = time.Now()
		b.longTapTimer = time.AfterFunc(time.Millisecond*200, func() {
			if b.onLongTap != nil {
				b.onLongTap()
			}
		})
	}
}

// MouseUp handles the mouse up event
func (b *ImageButton) MouseUp(me *desktop.MouseEvent) {
	if b.longTapTimer != nil {
		b.longTapTimer.Stop()
	}
	if time.Since(b.pressedTime) < time.Millisecond*200 {
		b.Tapped(nil)
	}
}

// Refresh updates the widget's appearance
func (b *ImageButton) Refresh() {
	b.BaseWidget.Refresh()
	if b.Selected {
		b.Image.Translucency = 0.7
	} else {
		b.Image.Translucency = 0
	}
	canvas.Refresh(b.Image)
}

// CreateRenderer implements the fyne.Widget interface
func (b *ImageButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.Image)
}
