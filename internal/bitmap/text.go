package bitmap

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// DrawAlignedText draws text on an image with alignment and padding
func DrawAlignedText(img *image.Gray, text string, face font.Face, horizAlign, vertAlign string, padding int) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Measure text
	textWidth, textHeight := MeasureText(text, face)

	// Calculate available space
	contentX := padding
	contentY := padding
	contentW := width - padding*2
	contentH := height - padding*2

	// Calculate X position
	var x int
	switch horizAlign {
	case "left":
		x = contentX
	case "right":
		x = contentX + contentW - textWidth
	default: // center
		x = contentX + (contentW-textWidth)/2
	}

	// Calculate Y position (baseline)
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()

	var y int
	switch vertAlign {
	case "top":
		y = contentY + ascent
	case "bottom":
		y = contentY + contentH - textHeight + ascent
	default: // center
		y = contentY + (contentH-textHeight)/2 + ascent
	}

	// Draw text
	point := fixed.Point26_6{
		X: fixed.Int26_6(x << 6),
		Y: fixed.Int26_6(y << 6),
	}

	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: face,
		Dot:  point,
	}

	drawer.DrawString(text)
}

// DrawBorder draws a border around the image
func DrawBorder(img *image.Gray, borderColor uint8) {
	if img == nil {
		return
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	c := color.Gray{Y: borderColor}

	// Top and bottom borders
	for x := 0; x < width; x++ {
		img.Set(x, 0, c)
		img.Set(x, height-1, c)
	}

	// Left and right borders
	for y := 0; y < height; y++ {
		img.Set(0, y, c)
		img.Set(width-1, y, c)
	}
}
