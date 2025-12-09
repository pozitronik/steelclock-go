//go:build !windows && !linux

package audiovisualizer

import (
	"fmt"
	"image"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

func init() {
	widget.Register("audio_visualizer", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// AudioCaptureWCA stub for unsupported platforms
type AudioCaptureWCA struct{}

// GetSharedAudioCapture returns an error on unsupported platforms
func GetSharedAudioCapture() (*AudioCaptureWCA, error) {
	return nil, fmt.Errorf("audio capture is not supported on this platform")
}

// Widget stub for unsupported platforms (not Windows or Linux)
type Widget struct {
	*widget.BaseWidget
	errorWidget *widget.ErrorWidget
}

// New creates a stub widget that displays an error
func New(cfg config.WidgetConfig) (widget.Widget, error) {
	// Set default update interval for audio visualizer (33ms = ~30fps)
	if cfg.UpdateInterval <= 0 {
		cfg.UpdateInterval = 0.033
	}

	base := widget.NewBaseWidget(cfg)
	pos := base.GetPosition()

	return &Widget{
		BaseWidget:  base,
		errorWidget: widget.NewErrorWidget(pos.W, pos.H, "UNSUPPORTED"),
	}, nil
}

func (w *Widget) Update() error {
	if w.errorWidget != nil {
		return w.errorWidget.Update()
	}
	return nil
}

func (w *Widget) Render() (image.Image, error) {
	if w.errorWidget != nil {
		return w.errorWidget.Render()
	}
	return nil, nil
}
