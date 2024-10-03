package imgbtn

import (
	"fyne.io/fyne/v2"
	// "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	// "fyne.io/fyne/v2/container"
	// "fyne.io/fyne/v2/dialog"
	// "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ImageButton struct {
	widget.BaseWidget
	onTapped func()
	image    *canvas.Image
}

func NewImageButton(resource fyne.Resource, tapped func()) *ImageButton {
	img := &ImageButton{onTapped: tapped}
	img.ExtendBaseWidget(img)
	img.image = canvas.NewImageFromResource(resource)
	img.image.FillMode = canvas.ImageFillContain
	img.image.SetMinSize(fyne.NewSize(150, 150))
	return img
}

func (b *ImageButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.image)
}

func (b *ImageButton) Tapped(*fyne.PointEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}
