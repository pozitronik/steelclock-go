package widget

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// VolumeWidget displays system volume level
type VolumeWidget struct {
	*BaseWidget
	displayMode       string
	fillColor         uint8
	barBorder         bool
	autoHide          bool
	autoHideTimeout   float64
	gaugeColor        uint8
	gaugeNeedleColor  uint8
	triangleFillColor uint8
	triangleBorder    bool

	mu               sync.RWMutex
	volume           float64 // 0-100
	isMuted          bool
	lastVolumeChange time.Time
	lastVolume       float64
	face             font.Face

	// Background polling
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewVolumeWidget creates a new volume widget
func NewVolumeWidget(cfg config.WidgetConfig) (*VolumeWidget, error) {
	base := NewBaseWidget(cfg)

	displayMode := cfg.Properties.DisplayMode
	if displayMode == "" {
		displayMode = "bar_horizontal"
	}

	fillColor := cfg.Properties.FillColor
	if fillColor == 0 {
		fillColor = 255
	}

	gaugeColor := cfg.Properties.GaugeColor
	if gaugeColor == 0 {
		gaugeColor = 200
	}

	gaugeNeedleColor := cfg.Properties.GaugeNeedleColor
	if gaugeNeedleColor == 0 {
		gaugeNeedleColor = 255
	}

	triangleFillColor := cfg.Properties.TriangleFillColor
	if triangleFillColor == 0 {
		triangleFillColor = 255
	}

	autoHideTimeout := cfg.Properties.AutoHideTimeout
	if autoHideTimeout == 0 {
		autoHideTimeout = 2.0 // 2 seconds default
	}

	// Load font for text mode
	var fontFace font.Face
	if displayMode == "text" {
		fontSize := cfg.Properties.FontSize
		if fontSize == 0 {
			fontSize = 10
		}
		face, err := bitmap.LoadFont(cfg.Properties.Font, fontSize)
		if err == nil {
			fontFace = face
		}
	}

	w := &VolumeWidget{
		BaseWidget:        base,
		displayMode:       displayMode,
		fillColor:         uint8(fillColor),
		barBorder:         cfg.Properties.BarBorder,
		autoHide:          cfg.Properties.AutoHide,
		autoHideTimeout:   autoHideTimeout,
		gaugeColor:        uint8(gaugeColor),
		gaugeNeedleColor:  uint8(gaugeNeedleColor),
		triangleFillColor: uint8(triangleFillColor),
		triangleBorder:    cfg.Properties.TriangleBorder,
		lastVolumeChange:  time.Now(),
		face:              fontFace,
		stopChan:          make(chan struct{}),
	}

	// Start single background goroutine for polling volume
	w.wg.Add(1)
	go w.pollVolumeBackground()

	return w, nil
}

// pollVolumeBackground continuously polls system volume in a single background goroutine
// This prevents goroutine accumulation - only ONE goroutine ever exists for polling
func (w *VolumeWidget) pollVolumeBackground() {
	defer w.wg.Done()

	log.Printf("Volume widget: background polling started")

	// Poll every 100ms for responsive display
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			log.Printf("Volume widget: background polling stopped")
			return

		case <-ticker.C:
			// Directly call Windows API (synchronous)
			// If it hangs, only THIS goroutine hangs - nothing else is affected
			volume, muted, err := getSystemVolume()

			if err != nil {
				// Ignore errors, keep displaying last known volume
				continue
			}

			// Update cached volume under lock
			w.mu.Lock()
			changed := volume != w.lastVolume || muted != w.isMuted
			if changed {
				w.lastVolumeChange = time.Now()
				w.lastVolume = volume
			}
			w.volume = volume
			w.isMuted = muted
			w.mu.Unlock()
		}
	}
}

// Update is called periodically but just returns immediately
// All volume polling happens in the background goroutine
func (w *VolumeWidget) Update() error {
	// No-op: background goroutine handles all polling
	return nil
}

// Stop stops the background polling goroutine
func (w *VolumeWidget) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

// Render renders the volume widget
func (w *VolumeWidget) Render() (image.Image, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	pos := w.GetPosition()
	style := w.GetStyle()

	// Check if we should hide the widget (auto-hide mode)
	if w.autoHide {
		timeSinceChange := time.Since(w.lastVolumeChange).Seconds()
		if timeSinceChange > w.autoHideTimeout {
			// Return transparent/empty image
			img := bitmap.NewGrayscaleImage(pos.W, pos.H, uint8(style.BackgroundColor))
			return img, nil
		}
	}

	// Create base image
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, uint8(style.BackgroundColor))

	// Draw border if enabled
	if style.Border {
		bitmap.DrawBorder(img, uint8(style.BorderColor))
	}

	// Render based on display mode
	switch w.displayMode {
	case "text":
		w.renderText(img)
	case "bar_horizontal":
		w.renderBarHorizontal(img, pos, style)
	case "bar_vertical":
		w.renderBarVertical(img, pos, style)
	case "gauge":
		w.renderGauge(img, pos)
	case "triangle":
		w.renderTriangle(img, pos, style)
	default:
		w.renderBarHorizontal(img, pos, style)
	}

	return img, nil
}

// renderText renders volume as text
func (w *VolumeWidget) renderText(img *image.Gray) {
	if w.face == nil {
		return
	}

	text := fmt.Sprintf("%.0f%%", w.volume)
	if w.isMuted {
		text = "MUTE"
	}

	// Center the text (draws in white by default)
	bitmap.DrawAlignedText(img, text, w.face, "center", "center", 0)
}

// renderBarHorizontal renders volume as horizontal bar
func (w *VolumeWidget) renderBarHorizontal(img *image.Gray, pos config.PositionConfig, style config.StyleConfig) {
	padding := 2
	if style.Border {
		padding = 3
	}

	x := padding
	y := padding
	width := pos.W - (padding * 2)
	height := pos.H - (padding * 2)

	if width <= 0 || height <= 0 {
		return
	}

	// Draw bar background
	barImg := bitmap.NewGrayscaleImage(width, height, uint8(style.BackgroundColor))

	// Draw fill based on volume
	fillColor := color.Gray{Y: w.fillColor}
	fillWidth := int(float64(width) * (w.volume / 100.0))
	if fillWidth > 0 {
		for dy := 0; dy < height; dy++ {
			for dx := 0; dx < fillWidth; dx++ {
				barImg.Set(dx, dy, fillColor)
			}
		}
	}

	// Draw bar border if enabled
	if w.barBorder {
		bitmap.DrawBorder(barImg, w.fillColor)
	}

	// Copy to main image
	for dy := 0; dy < height && y+dy < pos.H; dy++ {
		for dx := 0; dx < width && x+dx < pos.W; dx++ {
			img.Set(x+dx, y+dy, barImg.GrayAt(dx, dy))
		}
	}

	// Draw mute indicator
	if w.isMuted {
		w.drawMuteIndicator(img, pos)
	}
}

// renderBarVertical renders volume as vertical bar
func (w *VolumeWidget) renderBarVertical(img *image.Gray, pos config.PositionConfig, style config.StyleConfig) {
	padding := 2
	if style.Border {
		padding = 3
	}

	x := padding
	y := padding
	width := pos.W - (padding * 2)
	height := pos.H - (padding * 2)

	if width <= 0 || height <= 0 {
		return
	}

	// Draw bar background
	barImg := bitmap.NewGrayscaleImage(width, height, uint8(style.BackgroundColor))

	// Draw fill based on volume (from bottom)
	fillColor := color.Gray{Y: w.fillColor}
	fillHeight := int(float64(height) * (w.volume / 100.0))
	startY := height - fillHeight

	if fillHeight > 0 {
		for dy := startY; dy < height; dy++ {
			for dx := 0; dx < width; dx++ {
				barImg.Set(dx, dy, fillColor)
			}
		}
	}

	// Draw bar border if enabled
	if w.barBorder {
		bitmap.DrawBorder(barImg, w.fillColor)
	}

	// Copy to main image
	for dy := 0; dy < height && y+dy < pos.H; dy++ {
		for dx := 0; dx < width && x+dx < pos.W; dx++ {
			img.Set(x+dx, y+dy, barImg.GrayAt(dx, dy))
		}
	}

	// Draw mute indicator
	if w.isMuted {
		w.drawMuteIndicator(img, pos)
	}
}

// renderGauge renders volume as an old-fashioned gauge with needle
func (w *VolumeWidget) renderGauge(img *image.Gray, pos config.PositionConfig) {
	// Use shared gauge drawing function
	bitmap.DrawGauge(img, pos, w.volume, w.gaugeColor, w.gaugeNeedleColor)

	// Draw mute indicator
	if w.isMuted {
		w.drawMuteIndicator(img, pos)
	}
}

// renderTriangle renders volume as a right-angled triangle that fills from left to right
// Right angle at bottom-right, hypotenuse from bottom-left to top-right
func (w *VolumeWidget) renderTriangle(img *image.Gray, pos config.PositionConfig, style config.StyleConfig) {
	padding := 2
	if style.Border {
		padding = 3
	}

	availWidth := pos.W - (padding * 2)
	availHeight := pos.H - (padding * 2)

	if availWidth <= 0 || availHeight <= 0 {
		return
	}

	fillColor := color.Gray{Y: w.triangleFillColor}
	borderColor := color.Gray{Y: w.triangleFillColor / 2}

	// Calculate how much of the triangle to fill based on volume (0-100%)
	fillWidth := int(float64(availWidth) * (w.volume / 100.0))

	// Draw right-angled triangle from left to right
	// Right angle at bottom-right corner
	for col := 0; col < availWidth; col++ {
		// Calculate height at this column
		// Height increases linearly from left (0) to right (availHeight)
		colHeight := ((col + 1) * availHeight) / availWidth
		if colHeight < 1 {
			colHeight = 1
		}

		// Start from bottom
		startY := padding + availHeight - colHeight
		x := padding + col

		// Determine if this column should be filled
		shouldFill := col < fillWidth

		// Draw the column from top to bottom
		for y := 0; y < colHeight; y++ {
			actualY := startY + y
			if x >= 0 && x < pos.W && actualY >= 0 && actualY < pos.H {
				if shouldFill {
					img.Set(x, actualY, fillColor)
				} else if w.triangleBorder {
					// Draw border outline for unfilled area
					if y == 0 || y == colHeight-1 {
						img.Set(x, actualY, borderColor)
					}
				}
			}
		}
	}

	// Draw mute indicator
	if w.isMuted {
		w.drawMuteIndicator(img, pos)
	}
}

// drawMuteIndicator draws a mute indicator (X or slash)
func (w *VolumeWidget) drawMuteIndicator(img *image.Gray, pos config.PositionConfig) {
	// Draw a visible mute overlay - semi-transparent effect with pattern
	// Use alternating pattern instead of solid X for better visibility
	muteColor := color.Gray{Y: 128} // Mid-gray for visibility on any background

	// Draw thicker diagonal stripes
	thickness := 2
	for i := 0; i < pos.W && i < pos.H; i++ {
		for t := -thickness; t <= thickness; t++ {
			// Diagonal \
			y1 := i + t
			if y1 >= 0 && y1 < pos.H && i >= 0 && i < pos.W {
				img.Set(i, y1, muteColor)
			}
			// Diagonal /
			y2 := pos.H - i - 1 + t
			if y2 >= 0 && y2 < pos.H && i >= 0 && i < pos.W {
				img.Set(i, y2, muteColor)
			}
		}
	}
}

// getSystemVolume returns the current system volume (0-100) and mute status
// This is a platform-specific function
func getSystemVolume() (volume float64, muted bool, err error) {
	// Get actual system volume from platform-specific implementation
	return getSystemVolumeImpl()
}
