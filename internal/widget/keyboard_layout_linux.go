//go:build linux

package widget

import (
	"fmt"
	"image"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// NewKeyboardLayoutWidget is not supported on Linux
func NewKeyboardLayoutWidget(cfg config.WidgetConfig) (*KeyboardLayoutWidget, error) {
	return nil, fmt.Errorf("keyboard_layout widget is not supported on Linux")
}

// KeyboardLayoutWidget stub for Linux
type KeyboardLayoutWidget struct {
	*BaseWidget
}

// Update stub
func (w *KeyboardLayoutWidget) Update() error {
	return nil
}

// Render stub
func (w *KeyboardLayoutWidget) Render() (image.Image, error) {
	return nil, fmt.Errorf("keyboard_layout widget is not supported on Linux")
}
