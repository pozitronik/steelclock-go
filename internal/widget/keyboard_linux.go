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

// KeyboardWidget displays lock key status (Linux stub - always shows off)
type KeyboardWidget struct {
	*BaseWidget
	fontSize      int
	horizAlign    string
	vertAlign     string
	padding       int
	spacing       int
	capsLockOn    string
	capsLockOff   string
	numLockOn     string
	numLockOff    string
	scrollLockOn  string
	scrollLockOff string
	colorOn       uint8
	colorOff      uint8
	capsState     bool
	numState      bool
	scrollState   bool
	fontFace      font.Face
}

// NewKeyboardWidget creates a new keyboard widget
func NewKeyboardWidget(cfg config.WidgetConfig) (*KeyboardWidget, error) {
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

	colorOn := cfg.Properties.IndicatorColorOn
	if colorOn == 0 {
		colorOn = 255
	}

	colorOff := cfg.Properties.IndicatorColorOff
	if colorOff == 0 {
		colorOff = 100
	}

	capsOn := cfg.Properties.CapsLockOn
	if capsOn == "" {
		capsOn = "C"
	}

	capsOff := cfg.Properties.CapsLockOff
	if capsOff == "" {
		capsOff = "c"
	}

	numOn := cfg.Properties.NumLockOn
	if numOn == "" {
		numOn = "N"
	}

	numOff := cfg.Properties.NumLockOff
	if numOff == "" {
		numOff = "n"
	}

	scrollOn := cfg.Properties.ScrollLockOn
	if scrollOn == "" {
		scrollOn = "S"
	}

	scrollOff := cfg.Properties.ScrollLockOff
	if scrollOff == "" {
		scrollOff = "s"
	}

	spacing := cfg.Properties.Spacing
	if spacing == 0 {
		spacing = 2
	}

	fontFace, err := bitmap.LoadFont(cfg.Properties.Font, fontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	return &KeyboardWidget{
		BaseWidget:    base,
		fontSize:      fontSize,
		horizAlign:    horizAlign,
		vertAlign:     vertAlign,
		padding:       cfg.Properties.Padding,
		spacing:       spacing,
		capsLockOn:    capsOn,
		capsLockOff:   capsOff,
		numLockOn:     numOn,
		numLockOff:    numOff,
		scrollLockOn:  scrollOn,
		scrollLockOff: scrollOff,
		colorOn:       uint8(colorOn),
		colorOff:      uint8(colorOff),
		fontFace:      fontFace,
		capsState:     false,
		numState:      false,
		scrollState:   false,
	}, nil
}

// Update updates the keyboard state (Linux stub - always false)
func (w *KeyboardWidget) Update() error {
	w.capsState = false
	w.numState = false
	w.scrollState = false
	return nil
}

// Render creates an image of the keyboard widget
func (w *KeyboardWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	style := w.GetStyle()

	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	if style.Border {
		bitmap.DrawBorder(img, uint8(style.BorderColor))
	}

	indicators := []struct {
		state bool
		on    string
		off   string
	}{
		{w.capsState, w.capsLockOn, w.capsLockOff},
		{w.numState, w.numLockOn, w.numLockOff},
		{w.scrollState, w.scrollLockOn, w.scrollLockOff},
	}

	text := ""
	for i, ind := range indicators {
		if i > 0 {
			text += " "
		}
		if ind.state {
			text += ind.on
		} else {
			text += ind.off
		}
	}

	bitmap.DrawAlignedText(img, text, w.fontFace, w.horizAlign, w.vertAlign, w.padding)

	return img, nil
}
