package colorutils

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"
)

// Helper function to convert a HSV color to Hex color string
func HSVToHex(h, s, v float64) string {
	h = math.Mod(h, 360)            // Ensure hue is between 0 and 359
	s = math.Max(0, math.Min(1, s)) // Clamp saturation between 0 and 1
	v = math.Max(0, math.Min(1, v)) // Clamp value between 0 and 1

	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c

	var r, g, b float64

	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	r = (r + m) * 255
	g = (g + m) * 255
	b = (b + m) * 255

	return fmt.Sprintf("#%02X%02X%02X", uint8(r), uint8(g), uint8(b))
}

func HexToHSV(hex string) (float64, float64, float64) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}

	// Convert hex to RGB
	r, _ := strconv.ParseUint(hex[:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)

	// Convert RGB to HSV
	h, s, v := rgbToHSV(float64(r)/255, float64(g)/255, float64(b)/255)

	return h, s, v
}

func rgbToHSV(r, g, b float64) (float64, float64, float64) {
	max := max(r, g, b)
	min := min(r, g, b)
	h := 0.0
	s := 0.0
	v := max

	d := max - min
	if max != 0 {
		s = d / max
	}

	if d != 0 {
		switch max {
		case r:
			h = (g - b) / d
		case g:
			h = 2 + (b-r)/d
		case b:
			h = 4 + (r-g)/d
		}
		h *= 60
		if h < 0 {
			h += 360
		}
	}

	return h / 360, s, v
}

func HexToRgb(hex string) (float64, float64, float64) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}

	rgb, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, 0, 0
	}
	return float64(rgb >> 16), float64(rgb >> 8 & 0xFF), float64(rgb & 0xFF)
}

// Helper function to convert hex color to color.Color
func HexToColor(hex string) (color.Color, error) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return nil, fmt.Errorf("invalid hex color")
	}
	rgb, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return nil, err
	}
	return color.RGBA{
		R: uint8(rgb >> 16),
		G: uint8(rgb >> 8 & 0xFF),
		B: uint8(rgb & 0xFF),
		A: 255,
	}, nil
}
