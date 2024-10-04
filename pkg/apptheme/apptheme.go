package apptheme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// make a new theme called defaultTheme
type DefaultTheme struct{}

// the new defaultTheme colors
func (DefaultTheme) Color(c fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch c {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x04, G: 0x10, B: 0x11, A: 0xff}
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
		// return color.NRGBA{R: 0xe6, G: 0xf7, B: 0xfa, A: 0xff} // button icon color
		return color.NRGBA{R: 0x04, G: 0x10, B: 0x11, A: 0xff}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xf}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x19}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0xe6, G: 0xf7, B: 0xfa, A: 0xff} // button icon color
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
func (DefaultTheme) Font(s fyne.TextStyle) fyne.Resource {
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
func (DefaultTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

// the new defaultTheme font size
func (DefaultTheme) Size(s fyne.ThemeSizeName) float32 {
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
		return 6
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

func GetThemeColor(t fyne.Theme, prop string) color.Color {
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

// func SetThemeColor(t fyne.Theme, prop string, newColor color.Color) color.Color {
// 	switch prop {
// 	case "BackgroundColor":
// 		return DefaultTheme.Color(t, "BackgroundColor", theme.VariantDark)
// 	case "ButtonColor":
// 		return DefaultTheme.Color(t, "ButtonColor", theme.VariantDark)
// 	case "DisabledButtonColor":
// 		return DefaultTheme.Color(t, "DisabledButtonColor", theme.VariantDark)
// 	case "TextColor":
// 		return DefaultTheme.Color(t, "TextColor", theme.VariantDark)
// 	case "DisabledTextColor":
// 		return DefaultTheme.Color(t, "DisabledTextColor", theme.VariantDark)
// 	case "IconColor":
// 		return DefaultTheme.Color(t, "IconColor", theme.VariantDark)
// 	case "DisabledIconColor":
// 		return DefaultTheme.Color(t, "DisabledIconColor", theme.VariantDark)
// 	case "PlaceHolderColor":
// 		return DefaultTheme.Color(t, "PlaceHolderColor", theme.VariantDark)
// 	case "PrimaryColor":
// 		return DefaultTheme.Color(t, "PrimaryColor", theme.VariantDark)
// 	case "HoverColor":
// 		return DefaultTheme.Color(t, "HoverColor", theme.VariantDark)
// 	case "FocusColor":
// 		return DefaultTheme.Color(t, "FocusColor", theme.VariantDark)
// 	case "ScrollBarColor":
// 		return DefaultTheme.Color(t, "ScrollBarColor", theme.VariantDark)
// 	case "ShadowColor":
// 		return DefaultTheme.Color(t, "ShadowColor", theme.VariantDark)
// 	case "ErrorColor":
// 		return DefaultTheme.Color(t, "ErrorColor", theme.VariantDark)
// 	default:
// 		return color.White
// 	}
// }
