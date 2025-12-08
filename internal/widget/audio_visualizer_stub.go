//go:build !windows && !linux

package widget

import (
	"fmt"
	"image"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func init() {
	Register("audio_visualizer", func(cfg config.WidgetConfig) (Widget, error) {
		return NewAudioVisualizerWidget(cfg)
	})
}

// AudioCaptureWCA stub for unsupported platforms
type AudioCaptureWCA struct{}

// GetSharedAudioCapture returns an error on unsupported platforms
func GetSharedAudioCapture() (*AudioCaptureWCA, error) {
	return nil, fmt.Errorf("audio capture is not supported on this platform")
}

// AudioVisualizerWidget stub for unsupported platforms (not Windows or Linux)
type AudioVisualizerWidget struct {
	*BaseWidget
	errorWidget *ErrorWidget
}

// NewAudioVisualizerWidget creates a stub widget that displays an error
func NewAudioVisualizerWidget(cfg config.WidgetConfig) (Widget, error) {
	// Set default update interval for audio visualizer (33ms = ~30fps)
	if cfg.UpdateInterval <= 0 {
		cfg.UpdateInterval = 0.033
	}

	base := NewBaseWidget(cfg)
	pos := base.GetPosition()

	return &AudioVisualizerWidget{
		BaseWidget:  base,
		errorWidget: NewErrorWidget(pos.W, pos.H, "UNSUPPORTED"),
	}, nil
}

func (w *AudioVisualizerWidget) Update() error {
	if w.errorWidget != nil {
		return w.errorWidget.Update()
	}
	return nil
}

func (w *AudioVisualizerWidget) Render() (image.Image, error) {
	if w.errorWidget != nil {
		return w.errorWidget.Render()
	}
	return nil, nil
}
