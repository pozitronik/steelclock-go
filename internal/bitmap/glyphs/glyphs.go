package glyphs

import (
	"image"
	"image/color"
)

// Glyph represents a single bitmap character or icon
type Glyph struct {
	Width  int      // Width in pixels
	Height int      // Height in pixels
	Data   [][]bool // [row][col] pixel data, true = draw pixel
}

// GlyphSet represents a collection of glyphs (font or icon set)
type GlyphSet struct {
	Name        string            // Name of the glyph set (e.g., "5x7", "keyboard_8x8")
	GlyphWidth  int               // Standard glyph width (for monospace fonts)
	GlyphHeight int               // Standard glyph height
	Glyphs      map[rune]*Glyph   // Character glyphs (for fonts)
	Icons       map[string]*Glyph // Named icons (for icon sets)
}

// DrawGlyph draws a single glyph at the specified position
func DrawGlyph(img *image.Gray, glyph *Glyph, x, y int, c color.Gray) {
	if glyph == nil {
		return
	}

	bounds := img.Bounds()

	for row := 0; row < glyph.Height && row < len(glyph.Data); row++ {
		pixelY := y + row
		if pixelY < 0 || pixelY >= bounds.Max.Y {
			continue
		}

		for col := 0; col < glyph.Width && col < len(glyph.Data[row]); col++ {
			pixelX := x + col
			if pixelX < 0 || pixelX >= bounds.Max.X {
				continue
			}

			if glyph.Data[row][col] {
				img.Set(pixelX, pixelY, c)
			}
		}
	}
}

// DrawText draws a text string using the specified glyph set
// Returns the total width of the rendered text in pixels
func DrawText(img *image.Gray, text string, x, y int, glyphSet *GlyphSet, c color.Gray) int {
	if glyphSet == nil || glyphSet.Glyphs == nil {
		return 0
	}

	currentX := x
	spacing := 1 // 1 pixel spacing between characters

	for _, ch := range text {
		glyph := GetGlyph(glyphSet, ch)
		if glyph == nil {
			// Skip unknown characters
			continue
		}

		DrawGlyph(img, glyph, currentX, y, c)
		currentX += glyph.Width + spacing
	}

	return currentX - x
}

// GetGlyph retrieves a glyph for the specified character from a glyph set
// Returns nil if the character is not found
func GetGlyph(glyphSet *GlyphSet, ch rune) *Glyph {
	if glyphSet == nil || glyphSet.Glyphs == nil {
		return nil
	}
	return glyphSet.Glyphs[ch]
}

// GetIcon retrieves a named icon from a glyph set
// Returns nil if the icon is not found
func GetIcon(glyphSet *GlyphSet, name string) *Glyph {
	if glyphSet == nil || glyphSet.Icons == nil {
		return nil
	}
	return glyphSet.Icons[name]
}

// MeasureText calculates the width in pixels of a text string when rendered
func MeasureText(text string, glyphSet *GlyphSet) int {
	if glyphSet == nil || glyphSet.Glyphs == nil {
		return 0
	}

	width := 0
	spacing := 1

	for _, ch := range text {
		glyph := GetGlyph(glyphSet, ch)
		if glyph != nil {
			width += glyph.Width + spacing
		}
	}

	// Remove trailing spacing
	if width > 0 {
		width -= spacing
	}

	return width
}
