package widget

import (
	"image"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
)

// ClockTextRenderer renders clock in text mode
type ClockTextRenderer struct {
	config ClockTextConfig
}

// NewClockTextRenderer creates a new text mode clock renderer
func NewClockTextRenderer(cfg ClockTextConfig) *ClockTextRenderer {
	return &ClockTextRenderer{
		config: cfg,
	}
}

// Render draws the clock as formatted text
func (r *ClockTextRenderer) Render(img *image.Gray, t time.Time, _, _, _, _ int) error {
	timeStr := t.Format(r.config.Format)
	bitmap.SmartDrawAlignedText(img, timeStr, r.config.FontFace, r.config.FontName,
		r.config.HorizAlign, r.config.VertAlign, r.config.Padding)
	return nil
}

// NeedsUpdate returns false as text mode has no animations
func (r *ClockTextRenderer) NeedsUpdate() bool {
	return false
}
