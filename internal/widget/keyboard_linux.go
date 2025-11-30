//go:build linux

package widget

import (
	"fmt"
	"image"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// KeyboardWidget stub for Linux
type KeyboardWidget struct {
	*BaseWidget
}

// NewKeyboardWidget is not supported on Linux
func NewKeyboardWidget(cfg config.WidgetConfig) (*KeyboardWidget, error) {
	return nil, fmt.Errorf("keyboard widget is not supported on Linux")
}

// Update stub
func (w *KeyboardWidget) Update() error {
	return nil
}

// Render stub
func (w *KeyboardWidget) Render() (image.Image, error) {
	return nil, fmt.Errorf("keyboard widget is not supported on Linux")
}
