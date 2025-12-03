//go:build linux

package widget

import (
	"image"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// KeyboardWidget stub for Linux - displays error via ErrorWidget
type KeyboardWidget struct {
	*BaseWidget
	errorWidget *ErrorWidget
}

// NewKeyboardWidget creates a stub widget that displays unsupported message
func NewKeyboardWidget(cfg config.WidgetConfig) (*KeyboardWidget, error) {
	base := NewBaseWidget(cfg)
	pos := base.GetPosition()

	return &KeyboardWidget{
		BaseWidget:  base,
		errorWidget: NewErrorWidget(pos.W, pos.H, "UNSUPPORTED"),
	}, nil
}

// Update delegates to error widget
func (w *KeyboardWidget) Update() error {
	if w.errorWidget != nil {
		return w.errorWidget.Update()
	}
	return nil
}

// Render delegates to error widget
func (w *KeyboardWidget) Render() (image.Image, error) {
	if w.errorWidget != nil {
		return w.errorWidget.Render()
	}
	return nil, nil
}
