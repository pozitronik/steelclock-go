package bitmap

import (
	"image"
	"image/color"

	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// Internal font names - use these in config "font" field
const (
	// FontNamePixel3x5 is the compact 3x5 pixel font name
	FontNamePixel3x5 = "pixel3x5"
	// FontNamePixel5x7 is the standard 5x7 pixel font name
	FontNamePixel5x7 = "pixel5x7"
)

// internalFonts maps font names to their GlyphSet
var internalFonts = map[string]*glyphs.GlyphSet{
	FontNamePixel3x5: glyphs.Font3x5,
	FontNamePixel5x7: glyphs.Font5x7,
	// Aliases for convenience
	"3x5": glyphs.Font3x5,
	"5x7": glyphs.Font5x7,
}

// IsInternalFont checks if the given font name refers to a built-in pixel font
func IsInternalFont(fontName string) bool {
	_, ok := internalFonts[fontName]
	return ok
}

// GetInternalFontByName returns the internal font by name, or nil if not found
// Valid names: "pixel3x5", "pixel5x7", "3x5", "5x7"
func GetInternalFontByName(name string) *glyphs.GlyphSet {
	return internalFonts[name]
}

// MeasureInternalText measures the width of text using the internal glyph-based font
func MeasureInternalText(text string, glyphSet *glyphs.GlyphSet) int {
	return glyphs.MeasureText(text, glyphSet)
}

// DrawAlignedInternalText draws text on an image with alignment and padding using internal fonts
func DrawAlignedInternalText(img *image.Gray, text string, glyphSet *glyphs.GlyphSet, horizAlign config.HAlign, vertAlign config.VAlign, padding int) {
	if glyphSet == nil {
		glyphSet = glyphs.Font5x7
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate content area
	contentW := width - padding*2
	contentH := height - padding*2

	// Measure text
	textWidth := glyphs.MeasureText(text, glyphSet)
	textHeight := glyphSet.GlyphHeight

	// Calculate X position
	var x int
	switch horizAlign {
	case config.AlignLeft:
		x = padding
	case config.AlignRight:
		x = padding + contentW - textWidth
	default: // center
		x = padding + (contentW-textWidth)/2
	}

	// Calculate Y position
	var y int
	switch vertAlign {
	case config.AlignTop:
		y = padding
	case config.AlignBottom:
		y = padding + contentH - textHeight
	default: // center
		y = padding + (contentH-textHeight)/2
	}

	// Draw text
	glyphs.DrawText(img, text, x, y, glyphSet, color.Gray{Y: 255})
}

// DrawInternalTextInRect draws text within a specific rectangle with alignment using internal fonts
func DrawInternalTextInRect(img *image.Gray, text string, glyphSet *glyphs.GlyphSet, rectX, rectY, rectW, rectH int, horizAlign config.HAlign, vertAlign config.VAlign, padding int) {
	if glyphSet == nil {
		glyphSet = glyphs.Font5x7
	}

	// Calculate content area
	contentX := rectX + padding
	contentY := rectY + padding
	contentW := rectW - padding*2
	contentH := rectH - padding*2

	// Measure text
	textWidth := glyphs.MeasureText(text, glyphSet)
	textHeight := glyphSet.GlyphHeight

	// Calculate X position
	var x int
	switch horizAlign {
	case config.AlignLeft:
		x = contentX
	case config.AlignRight:
		x = contentX + contentW - textWidth
	default: // center
		x = contentX + (contentW-textWidth)/2
	}

	// Calculate Y position
	var y int
	switch vertAlign {
	case config.AlignTop:
		y = contentY
	case config.AlignBottom:
		y = contentY + contentH - textHeight
	default: // center
		y = contentY + (contentH-textHeight)/2
	}

	// Draw text
	glyphs.DrawText(img, text, x, y, glyphSet, color.Gray{Y: 255})
}

// DrawInternalTextClipped draws text using internal font with clipping bounds
func DrawInternalTextClipped(img *image.Gray, text string, glyphSet *glyphs.GlyphSet, x, y, clipX, clipY, clipW, clipH int, c color.Gray) {
	if glyphSet == nil {
		glyphSet = glyphs.Font5x7
	}

	// Check if text is completely outside vertical clip bounds
	textHeight := glyphSet.GlyphHeight
	if y+textHeight < clipY || y >= clipY+clipH {
		return // Text is completely above or below clip area
	}

	// Draw each character with clipping
	currentX := x
	for _, r := range text {
		glyph := glyphs.GetGlyph(glyphSet, r)
		if glyph == nil {
			// Skip unknown characters, advance by glyph width + spacing
			currentX += glyphSet.GlyphWidth + 1
			continue
		}

		charWidth := glyph.Width

		// Skip if completely outside horizontal clip area
		if currentX+charWidth < clipX || currentX >= clipX+clipW {
			currentX += charWidth + 1
			continue
		}

		// Draw the glyph with clipping
		for dy := 0; dy < glyph.Height; dy++ {
			py := y + dy
			if py < clipY || py >= clipY+clipH {
				continue
			}
			for dx := 0; dx < glyph.Width; dx++ {
				px := currentX + dx
				if px < clipX || px >= clipX+clipW {
					continue
				}
				if glyph.Data[dy][dx] {
					img.Set(px, py, c)
				}
			}
		}

		currentX += charWidth + 1
	}
}
