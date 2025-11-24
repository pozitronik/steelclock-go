//go:build windows

package widget

import (
	"fmt"
	"image"
	"syscall"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

var (
	user32      = syscall.NewLazyDLL("user32.dll")
	getKeyState = user32.NewProc("GetKeyState")
)

const (
	VkCapital = 0x14 // Caps Lock
	VkNumlock = 0x90 // Num Lock
	VkScroll  = 0x91 // Scroll Lock
)

// KeyboardWidget displays lock key status
type KeyboardWidget struct {
	*BaseWidget
	fontSize      int
	horizAlign    string
	vertAlign     string
	padding       int
	spacing       int
	separator     string
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

	// Separator between indicators (defaults to empty string for condensed output)
	separator := ""
	if cfg.Properties.Separator != nil {
		separator = *cfg.Properties.Separator
	}

	// Lock indicator symbols - only apply defaults when config key is omitted (nil)
	// Empty string ("") is respected as intentional empty value
	capsOn := "C"
	if cfg.Properties.CapsLockOn != nil {
		capsOn = *cfg.Properties.CapsLockOn
	}

	capsOff := "c"
	if cfg.Properties.CapsLockOff != nil {
		capsOff = *cfg.Properties.CapsLockOff
	}

	numOn := "N"
	if cfg.Properties.NumLockOn != nil {
		numOn = *cfg.Properties.NumLockOn
	}

	numOff := "n"
	if cfg.Properties.NumLockOff != nil {
		numOff = *cfg.Properties.NumLockOff
	}

	scrollOn := "S"
	if cfg.Properties.ScrollLockOn != nil {
		scrollOn = *cfg.Properties.ScrollLockOn
	}

	scrollOff := "s"
	if cfg.Properties.ScrollLockOff != nil {
		scrollOff = *cfg.Properties.ScrollLockOff
	}

	spacing := cfg.Properties.Spacing
	if spacing == 0 {
		spacing = 2
	}

	// Load font
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
		separator:     separator,
		capsLockOn:    capsOn,
		capsLockOff:   capsOff,
		numLockOn:     numOn,
		numLockOff:    numOff,
		scrollLockOn:  scrollOn,
		scrollLockOff: scrollOff,
		colorOn:       uint8(colorOn),
		colorOff:      uint8(colorOff),
		fontFace:      fontFace,
	}, nil
}

// Update updates the keyboard state
func (w *KeyboardWidget) Update() error {
	w.capsState = isKeyToggled(VkCapital)
	w.numState = isKeyToggled(VkNumlock)
	w.scrollState = isKeyToggled(VkScroll)
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

	// Build indicator text
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
			text += w.separator
		}
		if ind.state {
			text += ind.on
		} else {
			text += ind.off
		}
	}

	// Draw text
	bitmap.DrawAlignedText(img, text, w.fontFace, w.horizAlign, w.vertAlign, w.padding)

	return img, nil
}

// isKeyToggled checks if a toggle key is enabled (Windows only)
func isKeyToggled(vkCode uint32) bool {
	ret, _, _ := getKeyState.Call(uintptr(vkCode))
	// The low-order bit indicates toggle state (1 = on, 0 = off)
	return (ret & 0x1) != 0
}
