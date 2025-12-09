//go:build linux

package keyboardlayout

import (
	"image"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

func init() {
	widget.Register("keyboard_layout", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Widget stub for Linux - displays error via ErrorWidget
type Widget struct {
	*widget.BaseWidget
	errorWidget *widget.ErrorWidget
}

// New creates a stub widget that displays unsupported message
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	pos := base.GetPosition()

	return &Widget{
		BaseWidget:  base,
		errorWidget: widget.NewErrorWidget(pos.W, pos.H, "UNSUPPORTED"),
	}, nil
}

// Update delegates to error widget
func (w *Widget) Update() error {
	if w.errorWidget != nil {
		return w.errorWidget.Update()
	}
	return nil
}

// Render delegates to error widget
func (w *Widget) Render() (image.Image, error) {
	if w.errorWidget != nil {
		return w.errorWidget.Render()
	}
	return nil, nil
}
