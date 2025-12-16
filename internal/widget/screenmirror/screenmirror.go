// Package screenmirror provides a widget that captures and displays screen content.
package screenmirror

import (
	"fmt"
	"image"
	"image/color"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

func init() {
	widget.Register("screen_mirror", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Dither mode constants
const (
	DitherFloydSteinberg = "floyd_steinberg"
	DitherOrdered        = "ordered"
	DitherNone           = "none"
)

// Default configuration values
const (
	defaultFPS        = 15
	defaultScaleMode  = "fit"
	defaultDitherMode = DitherFloydSteinberg
	minFPS            = 1
	maxFPS            = 30
)

// ScreenMirrorConfig holds screen mirror widget configuration.
type ScreenMirrorConfig struct {
	DisplayIndex *int
	DisplayName  string
	Region       *CaptureRegion
	Window       *WindowTarget
	ScaleMode    ScaleMode
	FPS          int
	DitherMode   string
}

// Widget displays captured screen content on the OLED display.
type Widget struct {
	*widget.BaseWidget

	cfg ScreenMirrorConfig

	// Screen capture
	capture ScreenCapture

	// Current frame
	currentImg *image.Gray
	mu         sync.RWMutex

	// Capture loop
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New creates a new screen mirror widget.
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)

	// Parse configuration
	mirrorCfg := parseScreenMirrorConfig(cfg)

	// Build capture configuration
	captureCfg := CaptureConfig{
		DisplayIndex: mirrorCfg.DisplayIndex,
		DisplayName:  mirrorCfg.DisplayName,
		Region:       mirrorCfg.Region,
		Window:       mirrorCfg.Window,
	}

	// Create platform-specific capture
	capture, err := newScreenCapture(captureCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create screen capture: %w", err)
	}

	w := &Widget{
		BaseWidget: base,
		cfg:        mirrorCfg,
		capture:    capture,
		stopCh:     make(chan struct{}),
	}

	// Start capture loop
	w.wg.Add(1)
	go w.captureLoop()

	return w, nil
}

// parseScreenMirrorConfig extracts configuration from WidgetConfig.
func parseScreenMirrorConfig(cfg config.WidgetConfig) ScreenMirrorConfig {
	mirrorCfg := ScreenMirrorConfig{
		ScaleMode:  ScaleMode(defaultScaleMode),
		FPS:        defaultFPS,
		DitherMode: defaultDitherMode,
	}

	if cfg.ScreenMirror != nil {
		sm := cfg.ScreenMirror

		mirrorCfg.DisplayIndex = sm.Display
		mirrorCfg.DisplayName = sm.DisplayName

		if sm.Region != nil {
			mirrorCfg.Region = &CaptureRegion{
				X:      sm.Region.X,
				Y:      sm.Region.Y,
				Width:  sm.Region.Width,
				Height: sm.Region.Height,
			}
		}

		if sm.Window != nil {
			mirrorCfg.Window = &WindowTarget{
				Title:  sm.Window.Title,
				Class:  sm.Window.Class,
				Active: sm.Window.Active,
			}
		}

		if sm.ScaleMode != "" {
			mirrorCfg.ScaleMode = ScaleMode(sm.ScaleMode)
		}

		if sm.FPS > 0 {
			mirrorCfg.FPS = sm.FPS
			if mirrorCfg.FPS < minFPS {
				mirrorCfg.FPS = minFPS
			} else if mirrorCfg.FPS > maxFPS {
				mirrorCfg.FPS = maxFPS
			}
		}

		if sm.DitherMode != "" {
			mirrorCfg.DitherMode = sm.DitherMode
		}
	}

	return mirrorCfg
}

// captureLoop continuously captures frames at the configured FPS.
func (w *Widget) captureLoop() {
	defer w.wg.Done()

	interval := time.Second / time.Duration(w.cfg.FPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.captureFrame()
		}
	}
}

// captureFrame captures a single frame and processes it.
func (w *Widget) captureFrame() {
	if w.capture == nil || !w.capture.IsAvailable() {
		return
	}

	// Capture the frame
	frame, err := w.capture.Capture()
	if err != nil {
		return
	}

	if frame == nil {
		return
	}

	// Get widget dimensions
	contentArea := w.GetContentArea()
	targetWidth := contentArea.Width
	targetHeight := contentArea.Height

	if targetWidth <= 0 || targetHeight <= 0 {
		return
	}

	// Scale the captured image
	scaled := ScaleImage(frame, targetWidth, targetHeight, w.cfg.ScaleMode)

	// Apply dithering if enabled
	var processed *image.Gray
	switch w.cfg.DitherMode {
	case DitherFloydSteinberg:
		processed = bitmap.FloydSteinbergDither(scaled)
	case DitherOrdered:
		// Ordered dithering - use simple threshold pattern
		processed = orderedDither(scaled)
	case DitherNone:
		fallthrough
	default:
		processed = scaled
	}

	// Store the processed frame
	w.mu.Lock()
	w.currentImg = processed
	w.mu.Unlock()
}

// orderedDither applies Bayer matrix ordered dithering.
func orderedDither(src *image.Gray) *image.Gray {
	bounds := src.Bounds()
	dst := image.NewGray(bounds)

	// 4x4 Bayer matrix normalized to 0-255 range
	bayer4x4 := [4][4]uint8{
		{0, 128, 32, 160},
		{192, 64, 224, 96},
		{48, 176, 16, 144},
		{240, 112, 208, 80},
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			oldPixel := src.GrayAt(x, y).Y
			threshold := bayer4x4[y%4][x%4]
			var newPixel uint8
			if oldPixel > threshold {
				newPixel = 255
			} else {
				newPixel = 0
			}
			dst.SetGray(x, y, color.Gray{Y: newPixel})
		}
	}

	return dst
}

// Update is called at the widget's update interval.
func (w *Widget) Update() error {
	// Frame capture is handled in the capture loop goroutine
	return nil
}

// Render creates an image of the current captured content.
func (w *Widget) Render() (image.Image, error) {
	// Check auto-hide
	if w.ShouldHide() {
		return nil, nil
	}

	// Create canvas with background
	img := w.CreateCanvas()
	w.ApplyBorder(img)

	// Get content area
	contentArea := w.GetContentArea()

	// Get current frame
	w.mu.RLock()
	frame := w.currentImg
	w.mu.RUnlock()

	if frame != nil {
		// Draw the captured frame onto the canvas
		for y := 0; y < frame.Bounds().Dy() && y < contentArea.Height; y++ {
			for x := 0; x < frame.Bounds().Dx() && x < contentArea.Width; x++ {
				pixel := frame.GrayAt(x, y)
				img.SetGray(contentArea.X+x, contentArea.Y+y, pixel)
			}
		}
	}

	return img, nil
}

// Stop stops the screen capture and releases resources.
func (w *Widget) Stop() {
	close(w.stopCh)
	w.wg.Wait()

	if w.capture != nil {
		w.capture.Close()
	}
}
