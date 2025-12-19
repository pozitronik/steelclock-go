package clock

import (
	"image"
	"strings"
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
	format := r.config.Format

	// Convert to 12-hour format if enabled
	if r.config.Use12h {
		// Replace Go's 24-hour format "15" with 12-hour format "3"
		format = strings.ReplaceAll(format, "15", "3")
	}

	timeStr := t.Format(format)

	// Append AM/PM indicator if enabled
	if r.config.Use12h && r.config.ShowAmPm {
		if t.Hour() < 12 {
			timeStr += " AM"
		} else {
			timeStr += " PM"
		}
	}

	bitmap.SmartDrawAlignedText(img, timeStr, r.config.FontFace, r.config.FontName,
		r.config.HorizAlign, r.config.VertAlign, r.config.Padding)
	return nil
}

// NeedsUpdate returns false as text mode has no animations
func (r *TextRenderer) NeedsUpdate() bool {
	return false
}
