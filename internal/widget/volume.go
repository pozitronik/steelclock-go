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

	// Diagnostic metrics for COM call monitoring
	totalCalls        int64
	successfulCalls   int64
	failedCalls       int64
	lastSuccessTime   time.Time
	maxCallDuration   time.Duration
	totalCallDuration time.Duration // for calculating average
	consecutiveErrors int

	// Platform-specific volume reader (managed by polling goroutine)
	reader volumeReader
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
		lastVolumeChange:  time.Time{}, // Zero time = widget starts hidden if auto-hide is enabled
		lastSuccessTime:   time.Now(),  // Initialize to prevent false "stuck" detection
		face:              fontFace,
		stopChan:          make(chan struct{}),
	}

	log.Printf("[VOLUME] Widget initialized: id=%s, mode=%s, autoHide=%v, timeout=%.1fs",
		cfg.ID, displayMode, cfg.Properties.AutoHide, autoHideTimeout)

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

	log.Printf("[VOLUME] Background polling goroutine started")

	// Initialize platform-specific volume reader on THIS goroutine
	// On Windows: COM must be initialized on the same thread that uses it
	reader, err := newVolumeReader()
	if err != nil {
		log.Printf("[VOLUME] FATAL: Failed to initialize volume reader: %v", err)
		log.Printf("[VOLUME] Volume widget will not function")
		return
	}
	w.reader = reader

	// Ensure cleanup when goroutine exits
	defer func() {
		if w.reader != nil {
			w.reader.Close()
			w.reader = nil
		}
	}()

	log.Printf("[VOLUME] Volume reader initialized successfully")

	// Poll every 100ms for responsive display
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Health summary ticker (every 60 seconds)
	healthTicker := time.NewTicker(60 * time.Second)
	defer healthTicker.Stop()

	iterationCount := int64(0)
	lastHealthReport := time.Now()

	for {
		select {
		case <-w.stopChan:
			log.Printf("[VOLUME] Background polling stopped - shutdown requested")
			return

		case <-healthTicker.C:
			// Log health metrics summary
			w.mu.RLock()
			avgDuration := time.Duration(0)
			if w.totalCalls > 0 {
				avgDuration = time.Duration(w.totalCallDuration.Nanoseconds() / w.totalCalls)
			}
			timeSinceSuccess := time.Since(w.lastSuccessTime)
			successRate := float64(0)
			if w.totalCalls > 0 {
				successRate = float64(w.successfulCalls) / float64(w.totalCalls) * 100
			}

			log.Printf("[VOLUME] === HEALTH SUMMARY (last 60s) ===")
			log.Printf("[VOLUME] Total calls: %d | Success: %d | Failed: %d | Success rate: %.1f%%",
				w.totalCalls, w.successfulCalls, w.failedCalls, successRate)
			log.Printf("[VOLUME] Avg duration: %v | Max duration: %v", avgDuration, w.maxCallDuration)
			log.Printf("[VOLUME] Last success: %v ago | Consecutive errors: %d",
				timeSinceSuccess, w.consecutiveErrors)
			log.Printf("[VOLUME] Iterations: %d | Uptime: %v", iterationCount, time.Since(lastHealthReport))
			w.mu.RUnlock()

			lastHealthReport = time.Now()

		case <-ticker.C:
			iterationCount++

			// Log every 100th iteration (every 10 seconds at 100ms tick)
			if iterationCount%100 == 0 {
				log.Printf("[VOLUME] Polling iteration #%d", iterationCount)
			}

			// Check for stuck state BEFORE calling getSystemVolume
			w.mu.RLock()
			timeSinceLastSuccess := time.Since(w.lastSuccessTime)
			w.mu.RUnlock()

			if timeSinceLastSuccess > 5*time.Second && iterationCount > 50 {
				log.Printf("[VOLUME] === STUCK DETECTED === No successful update for %v!", timeSinceLastSuccess)
			}

			// Call volume reader with timing
			callStart := time.Now()
			volume, muted, err := w.reader.GetVolume()
			callDuration := time.Since(callStart)

			// Update health metrics
			w.mu.Lock()
			w.totalCalls++
			w.totalCallDuration += callDuration

			if callDuration > w.maxCallDuration {
				w.maxCallDuration = callDuration
				log.Printf("[VOLUME] NEW MAX DURATION: %v (previous max exceeded)", callDuration)
			}

			if err != nil {
				// ERROR PATH - now we log instead of silently ignoring
				w.failedCalls++
				w.consecutiveErrors++
				log.Printf("[VOLUME] ERROR iteration #%d: getSystemVolume() failed after %v: %v (consecutive errors: %d)",
					iterationCount, callDuration, err, w.consecutiveErrors)
				w.mu.Unlock()
				continue
			}

			// SUCCESS PATH
			w.successfulCalls++
			w.lastSuccessTime = time.Now()
			if w.consecutiveErrors > 0 {
				log.Printf("[VOLUME] RECOVERED: Success after %d consecutive errors", w.consecutiveErrors)
				w.consecutiveErrors = 0
			}

			// Log slow calls
			if callDuration > 100*time.Millisecond {
				log.Printf("[VOLUME] WARNING iteration #%d: getSystemVolume() took %v (>100ms)", iterationCount, callDuration)
			}
			if callDuration > 500*time.Millisecond {
				log.Printf("[VOLUME] ERROR iteration #%d: getSystemVolume() took %v (>500ms) - CRITICAL SLOWDOWN!", iterationCount, callDuration)
			}

			// Update cached volume
			changed := volume != w.lastVolume || muted != w.isMuted
			if changed {
				w.lastVolumeChange = time.Now()
				w.lastVolume = volume
				log.Printf("[VOLUME] Volume changed: %.1f%% (muted=%v)", volume, muted)
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
	log.Printf("[VOLUME] Shutdown initiated - stopping background polling")

	close(w.stopChan)
	w.wg.Wait()

	// Log final health metrics
	w.mu.RLock()
	avgDuration := time.Duration(0)
	if w.totalCalls > 0 {
		avgDuration = time.Duration(w.totalCallDuration.Nanoseconds() / w.totalCalls)
	}
	successRate := float64(0)
	if w.totalCalls > 0 {
		successRate = float64(w.successfulCalls) / float64(w.totalCalls) * 100
	}

	log.Printf("[VOLUME] === FINAL HEALTH METRICS ===")
	log.Printf("[VOLUME] Total calls: %d | Success: %d | Failed: %d | Success rate: %.1f%%",
		w.totalCalls, w.successfulCalls, w.failedCalls, successRate)
	log.Printf("[VOLUME] Avg duration: %v | Max duration: %v", avgDuration, w.maxCallDuration)
	log.Printf("[VOLUME] Last volume: %.1f%% (muted=%v)", w.volume, w.isMuted)
	w.mu.RUnlock()

	log.Printf("[VOLUME] Widget shutdown complete")
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
			// Return nil to hide widget and show content below
			return nil, nil
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
