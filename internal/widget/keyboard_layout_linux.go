//go:build linux

package widget

import (
	"image"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// KeyboardLayoutWidget stub for Linux - displays error via ErrorWidget
type KeyboardLayoutWidget struct {
	*BaseWidget
	errorWidget *ErrorWidget
}

// NewKeyboardLayoutWidget creates a stub widget that displays unsupported message
func NewKeyboardLayoutWidget(cfg config.WidgetConfig) (*KeyboardLayoutWidget, error) {
	base := NewBaseWidget(cfg)
	pos := base.GetPosition()

	return &KeyboardLayoutWidget{
		BaseWidget:  base,
		errorWidget: NewErrorWidget(pos.W, pos.H, "UNSUPPORTED"),
	}, nil
}

// Update delegates to error widget
func (w *KeyboardLayoutWidget) Update() error {
	if w.errorWidget != nil {
		return w.errorWidget.Update()
	}
	return nil
}

// Render delegates to error widget
func (w *KeyboardLayoutWidget) Render() (image.Image, error) {
	if w.errorWidget != nil {
		return w.errorWidget.Render()
	}
	return nil, nil
}
