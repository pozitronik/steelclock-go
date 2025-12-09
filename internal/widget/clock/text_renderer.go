package clock

import (
	"image"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
)

// TextRenderer renders clock in text mode
type TextRenderer struct {
	config TextConfig
}

// NewTextRenderer creates a new text mode clock renderer
func NewTextRenderer(cfg TextConfig) *TextRenderer {
	return &TextRenderer{
		config: cfg,
	}
}

// Render draws the clock as formatted text
func (r *TextRenderer) Render(img *image.Gray, t time.Time, _, _, _, _ int) error {
	timeStr := t.Format(r.config.Format)
	bitmap.SmartDrawAlignedText(img, timeStr, r.config.FontFace, r.config.FontName,
		r.config.HorizAlign, r.config.VertAlign, r.config.Padding)
	return nil
}

// NeedsUpdate returns false as text mode has no animations
func (r *TextRenderer) NeedsUpdate() bool {
	return false
}
