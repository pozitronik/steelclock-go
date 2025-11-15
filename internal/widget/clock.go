package widget

import (
	"fmt"
	"image"
	"time"

	"github.com/pozitronik/steelclock/internal/bitmap"
	"github.com/pozitronik/steelclock/internal/config"
	"golang.org/x/image/font"
)

// ClockWidget displays current time
type ClockWidget struct {
	*BaseWidget
	format      string
	fontSize    int
	fontName    string
	horizAlign  string
	vertAlign   string
	padding     int
	currentTime string
	fontFace    font.Face
}

// NewClockWidget creates a new clock widget
func NewClockWidget(cfg config.WidgetConfig) (*ClockWidget, error) {
	base := NewBaseWidget(cfg)

	format := cfg.Properties.Format
	if format == "" {
		format = "15:04:05" // Default Go time format (HH:MM:SS)
	} else {
		// Convert Python strftime to Go format
		format = convertStrftimeToGo(format)
	}

	fontSize := cfg.Properties.FontSize
	if fontSize == 0 {
		fontSize = 12
	}

	horizAlign := cfg.Properties.HorizontalAlign
	if horizAlign == "" {
		horizAlign = "center"
	}

	vertAlign := cfg.Properties.VerticalAlign
	if vertAlign == "" {
		vertAlign = "center"
	}

	// Load font
	fontFace, err := bitmap.LoadFont(cfg.Properties.Font, fontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	return &ClockWidget{
		BaseWidget: base,
		format:     format,
		fontSize:   fontSize,
		fontName:   cfg.Properties.Font,
		horizAlign: horizAlign,
		vertAlign:  vertAlign,
		padding:    cfg.Properties.Padding,
		fontFace:   fontFace,
	}, nil
}

// Update updates the current time
func (w *ClockWidget) Update() error {
	w.currentTime = time.Now().Format(w.format)
	return nil
}

// Render creates an image of the clock
func (w *ClockWidget) Render() (image.Image, error) {
	// Update time if not set
	if w.currentTime == "" {
		if err := w.Update(); err != nil {
			return nil, fmt.Errorf("failed to update clock: %w", err)
		}
	}

	pos := w.GetPosition()
	style := w.GetStyle()

	// Create image with background
	var img *image.Gray
	if style.BackgroundOpacity < 255 {
		// TODO: Support alpha channel for transparency
		img = bitmap.NewGrayscaleImage(pos.W, pos.H, uint8(style.BackgroundColor))
	} else {
		img = bitmap.NewGrayscaleImage(pos.W, pos.H, uint8(style.BackgroundColor))
	}

	// Draw border if enabled
	if style.Border {
		bitmap.DrawBorder(img, uint8(style.BorderColor))
	}

	// Draw text
	bitmap.DrawAlignedText(img, w.currentTime, w.fontFace, w.horizAlign, w.vertAlign, w.padding)

	return img, nil
}

// convertStrftimeToGo converts Python strftime format to Go time format
// This is a simplified converter for common formats
func convertStrftimeToGo(strftime string) string {
	// Map common strftime patterns to Go format
	replacements := map[string]string{
		"%H:%M:%S": "15:04:05",
		"%H:%M":    "15:04",
		"%Y-%m-%d": "2006-01-02",
		"%d.%m.%Y": "02.01.2006",
		"%Y":       "2006",
		"%m":       "01",
		"%d":       "02",
		"%H":       "15",
		"%M":       "04",
		"%S":       "05",
	}

	result := strftime
	for old, goFmt := range replacements {
		if result == old {
			return goFmt
		}
	}

	// If no exact match found, try common patterns
	// For more complex formats, users should use Go format directly
	return result
}
