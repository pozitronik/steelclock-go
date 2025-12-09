package shared

import (
	"image"
	"image/color"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
)

// StatusRenderer renders status text using internal bitmap fonts
type StatusRenderer struct {
	glyphSet *glyphs.GlyphSet
	color    color.Gray
}

// NewStatusRenderer creates a renderer with the specified internal font
// fontName: "3x5", "5x7", or "" for default (5x7)
func NewStatusRenderer(fontName string) *StatusRenderer {
	gs := bitmap.GetInternalFontByName(fontName)
	if gs == nil {
		gs = glyphs.Font5x7
	}
	return &StatusRenderer{
		glyphSet: gs,
		color:    color.Gray{Y: 255},
	}
}

// DrawCentered draws text centered in the given bounds
func (r *StatusRenderer) DrawCentered(img *image.Gray, text string, x, y, width, height int) {
	textWidth, textHeight := r.MeasureText(text)

	// Center horizontally and vertically
	drawX := x + (width-textWidth)/2
	drawY := y + (height-textHeight)/2

	r.DrawAt(img, text, drawX, drawY)
}

// DrawAt draws text at the specified position
func (r *StatusRenderer) DrawAt(img *image.Gray, text string, x, y int) {
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	currentX := x
	for _, ch := range text {
		glyph := glyphs.GetGlyph(r.glyphSet, ch)
		if glyph == nil {
			continue
		}

		for row := 0; row < glyph.Height && row < len(glyph.Data); row++ {
			for col := 0; col < glyph.Width && col < len(glyph.Data[row]); col++ {
				if glyph.Data[row][col] {
					px, py := currentX+col, y+row
					if px >= 0 && px < imgWidth && py >= 0 && py < imgHeight {
						img.Set(px, py, r.color)
					}
				}
			}
		}
		currentX += glyph.Width + 1 // 1 pixel spacing between characters
	}
}

// DrawLeftAligned draws text left-aligned in the given bounds
func (r *StatusRenderer) DrawLeftAligned(img *image.Gray, text string, x, y, _, height int) {
	_, textHeight := r.MeasureText(text)

	// Left-aligned horizontally, centered vertically
	drawY := y + (height-textHeight)/2

	r.DrawAt(img, text, x, drawY)
}

// DrawRightAligned draws text right-aligned in the given bounds
func (r *StatusRenderer) DrawRightAligned(img *image.Gray, text string, x, y, width, height int) {
	textWidth, textHeight := r.MeasureText(text)

	// Right-aligned horizontally, centered vertically
	drawX := x + width - textWidth
	drawY := y + (height-textHeight)/2

	r.DrawAt(img, text, drawX, drawY)
}

// MeasureText returns the width and height of the text
func (r *StatusRenderer) MeasureText(text string) (width, height int) {
	if len(text) == 0 {
		return 0, 0
	}

	totalWidth := 0
	maxHeight := 0

	for _, ch := range text {
		glyph := glyphs.GetGlyph(r.glyphSet, ch)
		if glyph != nil {
			totalWidth += glyph.Width + 1 // 1 pixel spacing
			if glyph.Height > maxHeight {
				maxHeight = glyph.Height
			}
		}
	}

	if totalWidth > 0 {
		totalWidth-- // Remove trailing space
	}

	return totalWidth, maxHeight
}

// SetColor sets the rendering color
func (r *StatusRenderer) SetColor(c color.Gray) {
	r.color = c
}

// GetColor returns the current rendering color
func (r *StatusRenderer) GetColor() color.Gray {
	return r.color
}

// GlyphSet returns the underlying glyph set
func (r *StatusRenderer) GlyphSet() *glyphs.GlyphSet {
	return r.glyphSet
}
