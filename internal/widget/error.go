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
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1, // disabled
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

// NewErrorWidgetWithConfig creates an error widget using an existing widget config.
// This is useful for creating error proxies that inherit the original widget's position.
func NewErrorWidgetWithConfig(cfg config.WidgetConfig, message string) *ErrorWidget {
	// Override some config values for error display
	cfg.Type = "error"
	cfg.ID = cfg.ID + "_error"
	if cfg.Style == nil {
		cfg.Style = &config.StyleConfig{}
	}
	cfg.Style.Background = 0
	cfg.Style.Border = -1

	return &ErrorWidget{
		BaseWidget:  NewBaseWidget(cfg),
		message:     message,
		flashState:  true,
		lastFlash:   time.Now(),
		flashPeriod: 500 * time.Millisecond,
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

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Only draw if flash state is on
	if !w.flashState {
		return img, nil
	}

	c := color.Gray{Y: 255} // White foreground for error display

	// Try different layout combinations from largest to smallest
	// Each combination: (iconSet, font, margin)
	type layoutOption struct {
		iconSet *glyphs.GlyphSet
		font    *glyphs.GlyphSet
		margin  int
	}

	options := []layoutOption{
		{glyphs.CommonIcons24x24, glyphs.Font5x7, 3},
		{glyphs.CommonIcons16x16, glyphs.Font5x7, 3},
		{glyphs.CommonIcons12x12, glyphs.Font5x7, 2},
		{glyphs.CommonIcons12x12, glyphs.Font3x5, 2},
		{nil, glyphs.Font5x7, 2}, // No icons, 5x7 font only
		{nil, glyphs.Font3x5, 1}, // No icons, 3x5 font only
	}

	for _, opt := range options {
		if w.tryRenderLayout(img, pos.W, pos.H, opt.iconSet, opt.font, opt.margin, c) {
			return img, nil
		}
	}

	// Ultimate fallback: just draw a centered warning icon (smallest available)
	w.renderIconOnly(img, pos.W, pos.H, c)

	return img, nil
}

// tryRenderLayout attempts to render with the given layout options
// Returns true if the layout fits, false otherwise
func (w *ErrorWidget) tryRenderLayout(img *image.Gray, width, height int, iconSet, font *glyphs.GlyphSet, margin int, c color.Gray) bool {
	textWidth := glyphs.MeasureText(w.message, font)
	fontHeight := font.GlyphHeight
	centerY := height / 2

	if iconSet == nil {
		// Text only mode
		// Check if text fits
		if textWidth > width-margin*2 || fontHeight > height {
			return false
		}

		textX := (width - textWidth) / 2
		textY := centerY - fontHeight/2
		glyphs.DrawText(img, w.message, textX, textY, font, c)
		return true
	}

	// Icons + text mode
	icon := glyphs.GetIcon(iconSet, "warning")
	if icon == nil {
		return false
	}

	// Check if icons fit vertically
	if icon.Height > height {
		return false
	}

	// Calculate total width needed: margin + icon + margin + text + margin + icon + margin
	totalWidth := margin + icon.Width + margin + textWidth + margin + icon.Width + margin
	if totalWidth > width {
		return false
	}

	// Everything fits - render it
	leftX := margin
	rightX := width - margin - icon.Width
	iconY := centerY - icon.Height/2

	// Draw icons
	glyphs.DrawGlyph(img, icon, leftX, iconY, c)
	glyphs.DrawGlyph(img, icon, rightX, iconY, c)

	// Draw text centered between icons
	textAreaStart := leftX + icon.Width + margin
	textAreaWidth := rightX - textAreaStart - margin
	textX := textAreaStart + (textAreaWidth-textWidth)/2
	textY := centerY - fontHeight/2

	glyphs.DrawText(img, w.message, textX, textY, font, c)
	return true
}

// renderIconOnly draws just a warning icon centered (ultimate fallback)
func (w *ErrorWidget) renderIconOnly(img *image.Gray, width, height int, c color.Gray) {
	// Try icons from smallest to largest that fit
	iconSets := []*glyphs.GlyphSet{
		glyphs.CommonIcons12x12,
		glyphs.CommonIcons16x16,
		glyphs.CommonIcons24x24,
	}

	for _, iconSet := range iconSets {
		icon := glyphs.GetIcon(iconSet, "warning")
		if icon != nil && icon.Width <= width && icon.Height <= height {
			x := (width - icon.Width) / 2
			y := (height - icon.Height) / 2
			glyphs.DrawGlyph(img, icon, x, y, c)
			return
		}
	}
}
