//go:build linux
// +build linux

package widget

import (
	"fmt"
	"image"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// KeyboardLayoutWidget displays current keyboard layout (Linux stub - shows "N/A")
type KeyboardLayoutWidget struct {
	*BaseWidget
	fontSize      int
	horizAlign    string
	vertAlign     string
	padding       int
	displayFormat string
	fontFace      font.Face
}

// NewKeyboardLayoutWidget creates a new keyboard layout widget
func NewKeyboardLayoutWidget(cfg config.WidgetConfig) (*KeyboardLayoutWidget, error) {
	base := NewBaseWidget(cfg)

	fontSize := cfg.Properties.FontSize
	if fontSize == 0 {
		fontSize = 10
	}

	horizAlign := cfg.Properties.HorizontalAlign
	if horizAlign == "" {
		horizAlign = "center"
	}

	vertAlign := cfg.Properties.VerticalAlign
	if vertAlign == "" {
		vertAlign = "center"
	}

	displayFormat := cfg.Properties.DisplayFormat
	if displayFormat == "" {
		displayFormat = "iso639-1"
	}

	fontFace, err := bitmap.LoadFont(cfg.Properties.Font, fontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	return &KeyboardLayoutWidget{
		BaseWidget:    base,
		fontSize:      fontSize,
		horizAlign:    horizAlign,
		vertAlign:     vertAlign,
		padding:       cfg.Properties.Padding,
		displayFormat: displayFormat,
		fontFace:      fontFace,
	}, nil
}

// Update updates the keyboard layout state (Linux stub)
func (w *KeyboardLayoutWidget) Update() error {
	// Linux stub - no implementation
	return nil
}

// Render creates an image of the keyboard layout widget (Linux stub)
func (w *KeyboardLayoutWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	style := w.GetStyle()

	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	if style.Border {
		bitmap.DrawBorder(img, uint8(style.BorderColor))
	}

	// Show "N/A" on Linux
	bitmap.DrawAlignedText(img, "N/A", w.fontFace, w.horizAlign, w.vertAlign, w.padding)

	return img, nil
}
