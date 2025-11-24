package widget

import (
	"image"
	"image/color"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// ErrorWidget displays error messages with warning symbols
type ErrorWidget struct {
	*BaseWidget
	message     string
	flashState  bool
	lastFlash   time.Time
	flashPeriod time.Duration
}

// NewErrorWidget creates a new error widget
func NewErrorWidget(displayWidth, displayHeight int, message string) *ErrorWidget {
	cfg := config.WidgetConfig{
		Type:    "error",
		ID:      "error_display",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: displayWidth,
			H: displayHeight,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          false,
			BorderColor:     255,
		},
	}

	return &ErrorWidget{
		BaseWidget:  NewBaseWidget(cfg),
		message:     message,
		flashState:  true,
		lastFlash:   time.Now(),
		flashPeriod: 500 * time.Millisecond, // Flash every 500ms
	}
}

// Update toggles flash state
func (w *ErrorWidget) Update() error {
	now := time.Now()
	if now.Sub(w.lastFlash) >= w.flashPeriod {
		w.flashState = !w.flashState
		w.lastFlash = now
	}
	return nil
}

// Render draws the error display with warning triangles
func (w *ErrorWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	style := w.GetStyle()

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Only draw if flash state is on
	if !w.flashState {
		return img, nil
	}

	c := color.Gray{Y: uint8(style.BorderColor)}

	// Draw warning triangles on left and right using glyph system
	// Select icon size based on display height
	var iconSet *glyphs.GlyphSet
	if pos.H < 20 {
		iconSet = glyphs.CommonIcons8x8
	} else {
		iconSet = glyphs.CommonIcons10x10
	}

	warningIcon := glyphs.GetIcon(iconSet, "warning")
	if warningIcon == nil {
		return img, nil // Fail gracefully if icon not found
	}

	// Left triangle
	leftX := 5
	centerY := pos.H / 2
	glyphs.DrawGlyph(img, warningIcon, leftX, centerY-warningIcon.Height/2, c)

	// Right triangle
	rightX := pos.W - 5 - warningIcon.Width
	glyphs.DrawGlyph(img, warningIcon, rightX, centerY-warningIcon.Height/2, c)

	// Draw message text centered between triangles
	availableX := leftX + warningIcon.Width + 5
	availableW := (rightX) - (leftX + warningIcon.Width + 5)

	// Calculate text width using glyph system
	textWidth := glyphs.MeasureText(w.message, glyphs.Font5x7)

	// Center text in available space
	textX := availableX + (availableW-textWidth)/2
	if textX < availableX {
		textX = availableX // Don't go past left boundary
	}

	// Draw text using 5×7 pixel font
	glyphs.DrawText(img, w.message, textX, centerY-3, glyphs.Font5x7, c)

	return img, nil
}
