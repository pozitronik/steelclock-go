package bitmap

import (
	"image"
	"image/color"

	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// CalculateTextPosition calculates the X,Y position for aligned text within a content area.
// Returns the baseline position for drawing text with the given alignment.
// contentX, contentY define the top-left of the content area.
// contentW, contentH define the size of the content area.
func CalculateTextPosition(text string, face font.Face, contentX, contentY, contentW, contentH int, horizAlign config.HAlign, vertAlign config.VAlign) (x, y int) {
	fontMutex.Lock()
	defer fontMutex.Unlock()

	return calculateTextPositionUnsafe(text, face, contentX, contentY, contentW, contentH, horizAlign, vertAlign)
}

// calculateTextPositionUnsafe is the internal implementation without mutex (caller must hold lock)
func calculateTextPositionUnsafe(text string, face font.Face, contentX, contentY, contentW, contentH int, horizAlign config.HAlign, vertAlign config.VAlign) (x, y int) {
	textWidth, textHeight := measureTextUnsafe(text, face)

	// Calculate X position
	switch horizAlign {
	case config.AlignLeft:
		x = contentX
	case config.AlignRight:
		x = contentX + contentW - textWidth
	default: // center
		x = contentX + (contentW-textWidth)/2
	}

	// Calculate Y position (baseline)
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()

	switch vertAlign {
	case config.AlignTop:
		y = contentY + ascent
	case config.AlignBottom:
		y = contentY + contentH - textHeight + ascent
	default: // center
		y = contentY + (contentH-textHeight)/2 + ascent
	}

	return x, y
}

// DrawAlignedText draws text on an image with alignment and padding
func DrawAlignedText(img *image.Gray, text string, face font.Face, horizAlign config.HAlign, vertAlign config.VAlign, padding int) {
	// Protect font face access - font.Face is not thread-safe
	fontMutex.Lock()
	defer fontMutex.Unlock()

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate content area
	contentX := padding
	contentY := padding
	contentW := width - padding*2
	contentH := height - padding*2

	// Calculate position using shared logic
	x, y := calculateTextPositionUnsafe(text, face, contentX, contentY, contentW, contentH, horizAlign, vertAlign)

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
func DrawTextInRect(img *image.Gray, text string, face font.Face, x, y, width, height int, horizAlign config.HAlign, vertAlign config.VAlign, padding int) {
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
	case config.AlignLeft:
		textX = contentX
	case config.AlignRight:
		textX = contentX + contentW - textWidth
	default: // center
		textX = contentX + (contentW-textWidth)/2
	}

	// Calculate Y position (baseline)
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()

	var textY int
	switch vertAlign {
	case config.AlignTop:
		textY = contentY + ascent
	case config.AlignBottom:
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

// DrawTextAtPosition draws text at a specific position with clipping to a content area
// This is useful for scrolling text where the text may extend beyond visible bounds
func DrawTextAtPosition(img *image.Gray, text string, face font.Face, x, y, clipX, clipY, clipW, clipH int) {
	// Protect font face access - font.Face is not thread-safe
	fontMutex.Lock()
	defer fontMutex.Unlock()

	// Create a clipping mask by only drawing within the clip bounds
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

	// Get font metrics for vertical bounds checking
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()

	// Check if text is completely outside vertical clip bounds
	// y is the baseline position, so text extends from (y - ascent) to (y + descent)
	textTop := y - ascent
	textBottom := y + descent
	if textBottom < clipY || textTop >= clipY+clipH {
		return // Text is completely above or below clip area
	}

	// Draw each character, checking if it falls within horizontal clip bounds
	for _, r := range text {
		// Get glyph bounds
		advance, ok := face.GlyphAdvance(r)
		if !ok {
			continue
		}

		charX := drawer.Dot.X.Ceil()
		charWidth := advance.Ceil()

		// Skip if completely outside horizontal clip area
		if charX+charWidth < clipX || charX >= clipX+clipW {
			drawer.Dot.X += advance
			continue
		}

		// Draw the character
		drawer.DrawString(string(r))
	}
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

// SmartDrawAlignedText draws text using either TTF font or internal font based on the fontFace.
// If fontFace is nil and fontName is an internal font name, uses internal font rendering.
// Otherwise, uses TTF font rendering.
func SmartDrawAlignedText(img *image.Gray, text string, fontFace font.Face, fontName string, horizAlign config.HAlign, vertAlign config.VAlign, padding int) {
	if fontFace == nil && IsInternalFont(fontName) {
		glyphSet := GetInternalFontByName(fontName)
		DrawAlignedInternalText(img, text, glyphSet, horizAlign, vertAlign, padding)
		return
	}

	// Fall back to TTF rendering
	if fontFace != nil {
		DrawAlignedText(img, text, fontFace, horizAlign, vertAlign, padding)
	}
}

// SmartDrawTextInRect draws text within a rectangle using either TTF font or internal font.
// If fontFace is nil and fontName is an internal font name, uses internal font rendering.
// Otherwise, uses TTF font rendering.
func SmartDrawTextInRect(img *image.Gray, text string, fontFace font.Face, fontName string, x, y, width, height int, horizAlign config.HAlign, vertAlign config.VAlign, padding int) {
	if fontFace == nil && IsInternalFont(fontName) {
		glyphSet := GetInternalFontByName(fontName)
		DrawInternalTextInRect(img, text, glyphSet, x, y, width, height, horizAlign, vertAlign, padding)
		return
	}

	// Fall back to TTF rendering
	if fontFace != nil {
		DrawTextInRect(img, text, fontFace, x, y, width, height, horizAlign, vertAlign, padding)
	}
}

// SmartMeasureText measures text width using either TTF font or internal font.
// If fontFace is nil and fontName is an internal font name, uses internal font.
// Returns width and height.
func SmartMeasureText(text string, fontFace font.Face, fontName string) (int, int) {
	if fontFace == nil && IsInternalFont(fontName) {
		glyphSet := GetInternalFontByName(fontName)
		if glyphSet == nil {
			return 0, 0
		}
		width := MeasureInternalText(text, glyphSet)
		return width, glyphSet.GlyphHeight
	}

	// Fall back to TTF measurement
	if fontFace != nil {
		return MeasureText(text, fontFace)
	}

	return 0, 0
}

// SmartCalculateTextPosition calculates text position using either TTF font or internal font.
// If fontFace is nil and fontName is an internal font name, uses internal font.
// Returns x, y position for text drawing (baseline for TTF, top-left for internal).
func SmartCalculateTextPosition(text string, fontFace font.Face, fontName string, contentX, contentY, contentW, contentH int, horizAlign config.HAlign, vertAlign config.VAlign) (x, y int) {
	if fontFace == nil && IsInternalFont(fontName) {
		glyphSet := GetInternalFontByName(fontName)
		if glyphSet == nil {
			return contentX, contentY
		}
		// Calculate position for internal font (top-left based)
		textWidth := MeasureInternalText(text, glyphSet)
		textHeight := glyphSet.GlyphHeight

		// Calculate X position
		switch horizAlign {
		case config.AlignLeft:
			x = contentX
		case config.AlignRight:
			x = contentX + contentW - textWidth
		default: // center
			x = contentX + (contentW-textWidth)/2
		}

		// Calculate Y position (top-left, not baseline)
		switch vertAlign {
		case config.AlignTop:
			y = contentY
		case config.AlignBottom:
			y = contentY + contentH - textHeight
		default: // center
			y = contentY + (contentH-textHeight)/2
		}

		return x, y
	}

	// Fall back to TTF positioning
	if fontFace != nil {
		return CalculateTextPosition(text, fontFace, contentX, contentY, contentW, contentH, horizAlign, vertAlign)
	}

	return contentX, contentY
}

// SmartDrawTextAtPosition draws text at a specific position with clipping.
// If fontFace is nil and fontName is an internal font name, uses internal font.
func SmartDrawTextAtPosition(img *image.Gray, text string, fontFace font.Face, fontName string, x, y, clipX, clipY, clipW, clipH int) {
	if fontFace == nil && IsInternalFont(fontName) {
		glyphSet := GetInternalFontByName(fontName)
		if glyphSet == nil {
			return
		}
		// Draw internal font with clipping
		DrawInternalTextClipped(img, text, glyphSet, x, y, clipX, clipY, clipW, clipH, color.Gray{Y: 255})
		return
	}

	// Fall back to TTF rendering
	if fontFace != nil {
		DrawTextAtPosition(img, text, fontFace, x, y, clipX, clipY, clipW, clipH)
	}
}
