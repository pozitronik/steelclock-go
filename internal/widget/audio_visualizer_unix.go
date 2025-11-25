//go:build !windows

package widget

import (
	"fmt"
	"image"

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
	*BaseWidget
}

// NewAudioVisualizerWidget creates a stub widget that displays an error
func NewAudioVisualizerWidget(cfg config.WidgetConfig) (Widget, error) {
	// Set default update interval for audio visualizer (33ms = ~30fps)
	if cfg.UpdateInterval <= 0 {
		cfg.UpdateInterval = 0.033
	}

	return &AudioVisualizerWidget{
		BaseWidget: NewBaseWidget(cfg),
	}, nil
}

func (w *AudioVisualizerWidget) Update() error {
	return nil
}

func (w *AudioVisualizerWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	style := w.GetStyle()

	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Draw error message
	errorMsg := "AUDIO\nVISUALIZER\nWINDOWS\nONLY"
	face, _ := bitmap.LoadFont("", 8)
	if face != nil {
		bitmap.DrawAlignedText(img, errorMsg, face, "center", "center", 2)
	}

	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	return img, fmt.Errorf("audio visualizer is only supported on Windows")
}
