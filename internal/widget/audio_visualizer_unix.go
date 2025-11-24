//go:build !windows

package widget

import (
	"fmt"
	"image"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// AudioCaptureWCA stub for non-Windows platforms
type AudioCaptureWCA struct{}

// GetSharedAudioCapture returns an error on Unix platforms
func GetSharedAudioCapture() (*AudioCaptureWCA, error) {
	return nil, fmt.Errorf("audio capture is not supported on this platform (Unix/Linux/macOS)")
}

// AudioVisualizerWidget stub for non-Windows platforms
type AudioVisualizerWidget struct {
	id             string
	position       config.PositionConfig
	style          config.StyleConfig
	properties     config.WidgetProperties
	updateInterval time.Duration
}

// NewAudioVisualizerWidget creates a stub widget that displays an error
func NewAudioVisualizerWidget(cfg config.WidgetConfig) (Widget, error) {
	return &AudioVisualizerWidget{
		id:             cfg.ID,
		position:       cfg.Position,
		style:          cfg.Style,
		properties:     cfg.Properties,
		updateInterval: time.Duration(cfg.Properties.UpdateInterval * float64(time.Second)),
	}, nil
}

func (w *AudioVisualizerWidget) Name() string {
	return w.id
}

func (w *AudioVisualizerWidget) GetUpdateInterval() time.Duration {
	return w.updateInterval
}

func (w *AudioVisualizerWidget) GetPosition() config.PositionConfig {
	return w.position
}

func (w *AudioVisualizerWidget) GetStyle() config.StyleConfig {
	return w.style
}

func (w *AudioVisualizerWidget) Update() error {
	return nil
}

func (w *AudioVisualizerWidget) Render() (image.Image, error) {
	img := bitmap.NewGrayscaleImage(w.position.W, w.position.H, uint8(w.style.BackgroundColor))

	// Draw error message
	errorMsg := "AUDIO\nVISUALIZER\nWINDOWS\nONLY"
	face, _ := bitmap.LoadFont("", 8)
	if face != nil {
		bitmap.DrawAlignedText(img, errorMsg, face, "center", "center", 2)
	}

	if w.style.Border {
		bitmap.DrawBorder(img, uint8(w.style.BorderColor))
	}

	return img, fmt.Errorf("audio visualizer is only supported on Windows")
}
