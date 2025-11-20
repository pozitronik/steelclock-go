package bitmap

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// DrawAlignedText draws text on an image with alignment and padding
func DrawAlignedText(img *image.Gray, text string, face font.Face, horizAlign, vertAlign string, padding int) {
	// Protect font face access - font.Face is not thread-safe
	fontMutex.Lock()
	defer fontMutex.Unlock()

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Measure text
	textWidth, textHeight := measureTextUnsafe(text, face)

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

// DrawTextInRect draws text within a specific rectangle with alignment and padding
func DrawTextInRect(img *image.Gray, text string, face font.Face, x, y, width, height int, horizAlign, vertAlign string, padding int) {
	// Protect font face access - font.Face is not thread-safe
	fontMutex.Lock()
	defer fontMutex.Unlock()

	// Measure text
	textWidth, textHeight := measureTextUnsafe(text, face)

	// Calculate available space
	contentX := x + padding
	contentY := y + padding
	contentW := width - padding*2
	contentH := height - padding*2

	// Calculate X position
	var textX int
	switch horizAlign {
	case "left":
		textX = contentX
	case "right":
		textX = contentX + contentW - textWidth
	default: // center
		textX = contentX + (contentW-textWidth)/2
	}

	// Calculate Y position (baseline)
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()

	var textY int
	switch vertAlign {
	case "top":
		textY = contentY + ascent
	case "bottom":
		textY = contentY + contentH - textHeight + ascent
	default: // center
		textY = contentY + (contentH-textHeight)/2 + ascent
	}

	// Draw text
	point := fixed.Point26_6{
		X: fixed.Int26_6(textX << 6),
		Y: fixed.Int26_6(textY << 6),
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
