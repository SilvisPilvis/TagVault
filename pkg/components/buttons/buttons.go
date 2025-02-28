package buttons

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	fyneGif "fyne.io/x/fyne/widget"
)

type imageButton struct {
	widget.BaseWidget
	Image        *canvas.Image
	onTapped     func()
	onLongTap    func()
	onRightClick func()
	pressedTime  time.Time
	longTapTimer *time.Timer
	Selected     bool
}

type FileButton struct {
	widget.BaseWidget
	Icon         fyne.Resource
	Text         string
	Selected     bool
	onTapped     func()
	onLongTap    func()
	pressedTime  time.Time
	longTapTimer *time.Timer
	onRightClick func()
}

func NewFileButton(icon *fyne.Resource, text string, tapped func(), rightClick func()) *FileButton {
	button := &FileButton{
		Icon:         *icon,
		Text:         text,
		Selected:     false,
		onTapped:     tapped,
		onRightClick: rightClick,
		// onLongTap:    longTap,
	}
	button.ExtendBaseWidget(button)
	return button
}

func (b *FileButton) Tapped(me *desktop.MouseEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

func (b *FileButton) SetOnTapped(f func()) {
	b.onTapped = f
}

func (b *FileButton) SetOnLongTap(f func()) {
	b.onLongTap = f
}

func (b *FileButton) LongTap(me *desktop.MouseEvent) {
	if me.Button == desktop.MouseButtonPrimary {
		if b.onLongTap != nil {
			b.onLongTap()
			b.Selected = !b.Selected
			// b.Refresh()
		}
	}
}

func (b *FileButton) MouseDown(me *desktop.MouseEvent) {
	if me.Button == desktop.MouseButtonPrimary {
		b.pressedTime = time.Now()
		b.longTapTimer = time.AfterFunc(time.Millisecond*200, func() {
			if b.onLongTap != nil {
				b.onLongTap()
			}
		})
	}
}

func (b *FileButton) MouseUp(_ *desktop.MouseEvent) {
	if b.longTapTimer != nil {
		b.longTapTimer.Stop()
	}
	if time.Since(b.pressedTime) < time.Millisecond*200 {
		b.Tapped(nil)
	}
}

func (b *FileButton) TappedSecondary(_ *fyne.PointEvent) {
	if b.onRightClick != nil {
		b.onRightClick()
	}
}

func (b *FileButton) CreateRenderer() fyne.WidgetRenderer {
	icon := canvas.NewImageFromResource(b.Icon)

	icon.SetMinSize(fyne.NewSize(theme.IconInlineSize(), theme.IconInlineSize()))

	text := canvas.NewText(b.Text, color.NRGBA{R: 0x9e, G: 0xbd, B: 0xff, A: 0xff})

	text.Alignment = fyne.TextAlignCenter

	content := container.NewHBox(icon, text)

	return widget.NewSimpleRenderer(content)
}

func NewImageButton(resource fyne.Resource) *imageButton {
	img := &imageButton{}
	img.ExtendBaseWidget(img)
	img.Image = canvas.NewImageFromResource(resource)
	img.Image.FillMode = canvas.ImageFillContain
	img.Image.SetMinSize(fyne.NewSize(150, 150))
	return img
}

func (b *imageButton) Tapped(me *desktop.MouseEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

func (b *imageButton) TappedSecondary(_ *fyne.PointEvent) {
	if b.onRightClick != nil {
		b.onRightClick()
	}
}

func (b *imageButton) LongTap(me *desktop.MouseEvent) {
	if me.Button == desktop.MouseButtonPrimary {
		if b.onLongTap != nil {
			b.onLongTap()
			b.Selected = !b.Selected
			b.Refresh()
		}
	}
}

func (b *imageButton) Refresh() {
	if b.Selected {
		b.Image.Translucency = 0.7
	} else {
		b.Image.Translucency = 0
	}
	canvas.Refresh(b.Image)
}

func (b *imageButton) MouseDown(me *desktop.MouseEvent) {
	if me.Button == desktop.MouseButtonPrimary {
		b.pressedTime = time.Now()
		b.longTapTimer = time.AfterFunc(time.Millisecond*200, func() {
			if b.onLongTap != nil {
				b.onLongTap()
			}
		})
	}
}

func (b *imageButton) MouseUp(_ *desktop.MouseEvent) {
	if b.longTapTimer != nil {
		b.longTapTimer.Stop()
	}
	if time.Since(b.pressedTime) < time.Millisecond*200 {
		b.Tapped(nil)
	}
}

func (b *imageButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.Image)
}

func (b *imageButton) SetOnTapped(f func()) {
	b.onTapped = f
}

// SetOnLongTap sets the function to be called when the button is long-pressed
func (b *imageButton) SetOnLongTap(f func()) {
	b.onLongTap = f
}

// SetOnRightClick sets the function to be called when the button is right-clicked
func (b *imageButton) SetOnRightClick(f func()) {
	b.onRightClick = f
}

type GifButton struct {
	widget.BaseWidget
	animation    *fyneGif.AnimatedGif
	onTapped     func()
	onLongTap    func()
	onRightClick func()
	pressedTime  time.Time
	longTapTimer *time.Timer
	Selected     bool
}

// NewGifButton creates a new animated GIF button from the specified resource
func NewGifButton(path fyne.URI) *GifButton {
	gif := &GifButton{}
	gif.ExtendBaseWidget(gif)
	// gif.animation, _ = fyneGif.NewAnimatedGifFromResource(resource)
	gif.animation, _ = fyneGif.NewAnimatedGif(path)
	gif.animation.SetMinSize(fyne.NewSize(150, 150))
	gif.animation.Start() // Start the animation by default
	return gif
}

// SetOnTapped sets the function to be called when the button is tapped
func (b *GifButton) SetOnTapped(f func()) {
	b.onTapped = f
}

// SetOnLongTap sets the function to be called when the button is long-pressed
func (b *GifButton) SetOnLongTap(f func()) {
	b.onLongTap = f
}

// SetOnRightClick sets the function to be called when the button is right-clicked
func (b *GifButton) SetOnRightClick(f func()) {
	b.onRightClick = f
}

// StartAnimation starts the GIF animation
func (b *GifButton) StartAnimation() {
	b.animation.Start()
}

// StopAnimation stops the GIF animation
func (b *GifButton) StopAnimation() {
	b.animation.Stop()
}

// Tapped handles the tap event
func (b *GifButton) Tapped(_ *desktop.MouseEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

// TappedSecondary handles the right-click event
func (b *GifButton) TappedSecondary(_ *fyne.PointEvent) {
	if b.onRightClick != nil {
		b.onRightClick()
	}
}

// MouseDown handles the mouse down event
func (b *GifButton) MouseDown(me *desktop.MouseEvent) {
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
func (b *GifButton) MouseUp(me *desktop.MouseEvent) {
	if b.longTapTimer != nil {
		b.longTapTimer.Stop()
	}
	if time.Since(b.pressedTime) < time.Millisecond*200 {
		b.Tapped(nil)
	}
}

// Refresh updates the widget's appearance
func (b *GifButton) Refresh() {
	b.BaseWidget.Refresh()
}

// CreateRenderer implements the fyne.Widget interface
func (b *GifButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.animation)
}
