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

	// Extract text settings
	fontSize := 10
	fontName := ""
	horizAlign := "center"
	vertAlign := "center"
	padding := 0

	if cfg.Text != nil {
		if cfg.Text.Size > 0 {
			fontSize = cfg.Text.Size
		}
		fontName = cfg.Text.Font
		if cfg.Text.Align != nil {
			if cfg.Text.Align.H != "" {
				horizAlign = cfg.Text.Align.H
			}
			if cfg.Text.Align.V != "" {
				vertAlign = cfg.Text.Align.V
			}
		}
	}

	// Extract padding from style
	if cfg.Style != nil {
		padding = cfg.Style.Padding
	}

	// Display format from config
	displayFormat := cfg.Format
	if displayFormat == "" {
		displayFormat = "iso639-1"
	}

	fontFace, err := bitmap.LoadFont(fontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	return &KeyboardLayoutWidget{
		BaseWidget:    base,
		fontSize:      fontSize,
		horizAlign:    horizAlign,
		vertAlign:     vertAlign,
		padding:       padding,
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
