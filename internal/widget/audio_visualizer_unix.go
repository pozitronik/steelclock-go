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
	updateInterval time.Duration
}

// NewAudioVisualizerWidget creates a stub widget that displays an error
func NewAudioVisualizerWidget(cfg config.WidgetConfig) (Widget, error) {
	// Extract style (handle nil pointer)
	style := config.StyleConfig{}
	if cfg.Style != nil {
		style = *cfg.Style
	}

	return &AudioVisualizerWidget{
		id:             cfg.ID,
		position:       cfg.Position,
		style:          style,
		updateInterval: time.Duration(cfg.UpdateInterval * float64(time.Second)),
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
	bgColor := uint8(0)
	if w.style.Background >= 0 {
		bgColor = uint8(w.style.Background)
	}
	img := bitmap.NewGrayscaleImage(w.position.W, w.position.H, bgColor)

	// Draw error message
	errorMsg := "AUDIO\nVISUALIZER\nWINDOWS\nONLY"
	face, _ := bitmap.LoadFont("", 8)
	if face != nil {
		bitmap.DrawAlignedText(img, errorMsg, face, "center", "center", 2)
	}

	if w.style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(w.style.Border))
	}

	return img, fmt.Errorf("audio visualizer is only supported on Windows")
}
