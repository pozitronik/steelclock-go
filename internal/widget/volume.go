package widget

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// volumeReader interface abstracts platform-specific volume reading
type volumeReader interface {
	GetVolume() (volume float64, muted bool, err error)
	Close()
}

// VolumeWidget displays system volume level
type VolumeWidget struct {
	*BaseWidget
	displayMode       string
	fillColor         uint8
	barDirection      string
	barBorder         bool
	gaugeColor        uint8
	gaugeNeedleColor  uint8
	gaugeShowTicks    bool
	gaugeTicksColor   uint8
	triangleFillColor uint8
	triangleBorder    bool
	horizAlign        string
	vertAlign         string
	padding           int

	mu         sync.RWMutex
	volume     float64 // 0-100
	isMuted    bool
	lastVolume float64
	face       font.Face

	// Background polling
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Diagnostic metrics for COM call monitoring
	totalCalls        int64
	successfulCalls   int64
	failedCalls       int64
	lastSuccessTime   time.Time
	consecutiveErrors int

	// Platform-specific volume reader (managed by polling goroutine)
	reader volumeReader
}

// NewVolumeWidget creates a new volume widget
func NewVolumeWidget(cfg config.WidgetConfig) (*VolumeWidget, error) {
	base := NewBaseWidget(cfg)

	displayMode := cfg.Mode
	if displayMode == "" {
		displayMode = "bar"
	}

	// Extract colors from mode-specific configs
	fillColor := 255
	gaugeColor := 200
	gaugeNeedleColor := 255
	gaugeShowTicks := true
	gaugeTicksColor := 150
	triangleFillColor := 255

	switch displayMode {
	case "bar":
		if cfg.Bar != nil && cfg.Bar.Colors != nil {
			if cfg.Bar.Colors.Fill != nil {
				fillColor = *cfg.Bar.Colors.Fill
			}
		}
	case "gauge":
		if cfg.Gauge != nil {
			if cfg.Gauge.ShowTicks != nil {
				gaugeShowTicks = *cfg.Gauge.ShowTicks
			}
			if cfg.Gauge.Colors != nil {
				if cfg.Gauge.Colors.Arc != nil {
					gaugeColor = *cfg.Gauge.Colors.Arc
				}
				if cfg.Gauge.Colors.Needle != nil {
					gaugeNeedleColor = *cfg.Gauge.Colors.Needle
				}
				if cfg.Gauge.Colors.Ticks != nil {
					gaugeTicksColor = *cfg.Gauge.Colors.Ticks
				}
			}
		}
	case "triangle":
		if cfg.Triangle != nil && cfg.Triangle.Colors != nil {
			if cfg.Triangle.Colors.Fill != nil {
				triangleFillColor = *cfg.Triangle.Colors.Fill
			}
		}
	}

	// Extract text settings
	fontSize := 10
	fontName := ""
	horizAlign := "center"
	vertAlign := "center"
	padding := 0

	if cfg.Text != nil {
		if cfg.Text.Size > 0 {
			fontSize = cfg.Text.Size
		}
		fontName = cfg.Text.Font
		if cfg.Text.Align != nil {
			if cfg.Text.Align.H != "" {
				horizAlign = cfg.Text.Align.H
			}
			if cfg.Text.Align.V != "" {
				vertAlign = cfg.Text.Align.V
			}
		}
	}

	// Extract padding from style
	if cfg.Style != nil {
		padding = cfg.Style.Padding
	}

	// Extract bar settings
	barDirection := "horizontal"
	barBorder := false
	if cfg.Bar != nil {
		if cfg.Bar.Direction != "" {
			barDirection = cfg.Bar.Direction
		}
		barBorder = cfg.Bar.Border
	}

	// Extract triangle settings
	triangleBorder := false
	if cfg.Triangle != nil {
		triangleBorder = cfg.Triangle.Border
	}

	// Load font for text mode
	var fontFace font.Face
	if displayMode == "text" {
		face, err := bitmap.LoadFont(fontName, fontSize)
		if err == nil {
			fontFace = face
		}
	}

	w := &VolumeWidget{
		BaseWidget:        base,
		displayMode:       displayMode,
		fillColor:         uint8(fillColor),
		barDirection:      barDirection,
		barBorder:         barBorder,
		gaugeColor:        uint8(gaugeColor),
		gaugeNeedleColor:  uint8(gaugeNeedleColor),
		gaugeShowTicks:    gaugeShowTicks,
		gaugeTicksColor:   uint8(gaugeTicksColor),
		triangleFillColor: uint8(triangleFillColor),
		triangleBorder:    triangleBorder,
		horizAlign:        horizAlign,
		vertAlign:         vertAlign,
		padding:           padding,
		lastSuccessTime:   time.Now(), // Initialize to prevent false "stuck" detection
		face:              fontFace,
		stopChan:          make(chan struct{}),
	}

	// Initialize volume reader BEFORE starting goroutine
	// This ensures widget creation fails if no audio device exists (fail-fast pattern)
	reader, err := newVolumeReader()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize volume reader: %w", err)
	}
	w.reader = reader

	// Start single background goroutine for polling volume
	w.wg.Add(1)
	go w.pollVolumeBackground()

	return w, nil
}

// pollVolumeBackground continuously polls system volume in a single background goroutine
// This prevents goroutine accumulation - only ONE goroutine ever exists for polling
func (w *VolumeWidget) pollVolumeBackground() {
	defer w.wg.Done()

	// Panic recovery to prevent app crash
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[VOLUME] PANIC in polling goroutine: %v\nStack: %s", r, debug.Stack())
		}
	}()

	// Reader already initialized in NewVolumeWidget (fail-fast pattern)
	// Ensure cleanup when goroutine exits
	defer func() {
		if w.reader != nil {
			w.reader.Close()
			w.reader = nil
		}
	}()

	// Poll every 100ms for responsive display
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return

		case <-ticker.C:
			// Call volume reader
			volume, muted, err := w.reader.GetVolume()

			w.mu.Lock()
			if err != nil {
				// Update error tracking
				w.failedCalls++
				w.consecutiveErrors++
				w.mu.Unlock()
				continue
			}

			// Update success tracking
			w.successfulCalls++
			w.lastSuccessTime = time.Now()
			w.consecutiveErrors = 0

			// Update cached volume
			changed := volume != w.lastVolume || muted != w.isMuted
			if changed {
				w.lastVolume = volume
				// Trigger auto-hide timer (widget becomes visible)
				w.TriggerAutoHide()
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

	// Check if widget should be hidden (auto-hide mode)
	if w.ShouldHide() {
		// Return nil to hide widget and show content below
		return nil, nil
	}

	// Create base image
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Draw border if enabled (border >= 0 means enabled with that color)
	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	// Render based on display mode
	switch w.displayMode {
	case "text":
		w.renderText(img)
	case "bar":
		if w.barDirection == "vertical" {
			w.renderBarVertical(img, pos, style)
		} else {
			w.renderBarHorizontal(img, pos, style)
		}
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

	// Draw text with configured alignment
	bitmap.DrawAlignedText(img, text, w.face, w.horizAlign, w.vertAlign, w.padding)
}

// renderBarHorizontal renders volume as horizontal bar
func (w *VolumeWidget) renderBarHorizontal(img *image.Gray, pos config.PositionConfig, style config.StyleConfig) {
	padding := 2
	if style.Border >= 0 {
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
	barImg := bitmap.NewGrayscaleImage(width, height, w.GetRenderBackgroundColor())

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
	if style.Border >= 0 {
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
	barImg := bitmap.NewGrayscaleImage(width, height, w.GetRenderBackgroundColor())

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
	bitmap.DrawGauge(img, 0, 0, pos.W, pos.H, w.volume, w.gaugeColor, w.gaugeNeedleColor, w.gaugeShowTicks, w.gaugeTicksColor)

	// Draw mute indicator
	if w.isMuted {
		w.drawMuteIndicator(img, pos)
	}
}

// renderTriangle renders volume as a right-angled triangle that fills from left to right.
// Right angle at bottom-right, hypotenuse from bottom-left to top-right
//
//nolint:gocyclo // Geometric calculations for triangle rendering
func (w *VolumeWidget) renderTriangle(img *image.Gray, pos config.PositionConfig, style config.StyleConfig) {
	padding := 2
	if style.Border >= 0 {
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

	// Draw right-angled triangle from left to right.
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
