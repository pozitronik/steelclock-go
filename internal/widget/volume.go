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

// reinitializableVolumeReader extends volumeReader with reinitialize capability
type reinitializableVolumeReader interface {
	volumeReader
	Reinitialize() error
	NeedsReinitialize() bool
}

// VolumeWidget displays system volume level
type VolumeWidget struct {
	*BaseWidget
	displayMode      string
	fillColor        uint8
	barDirection     string
	barBorder        bool
	gaugeColor       uint8
	gaugeNeedleColor uint8
	gaugeShowTicks   bool
	gaugeTicksColor  uint8
	fontName         string
	horizAlign       string
	vertAlign        string
	padding          int
	pollInterval     time.Duration // Configurable internal polling rate

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
	errorThreshold    int // Threshold before entering error state

	// Error state
	errorWidget *ErrorWidget // Error widget proxy (nil = normal operation)

	// Platform-specific volume reader (managed by polling goroutine)
	reader volumeReader
}

// NewVolumeWidget creates a new volume widget
func NewVolumeWidget(cfg config.WidgetConfig) (*VolumeWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract common settings using helper
	displayMode := helper.GetDisplayMode("bar")
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()
	barSettings := helper.GetBarSettings()
	gaugeSettings := helper.GetGaugeSettings()
	fillColor := helper.GetFillColorForMode(displayMode)

	// Load font for text mode (ignore error - volume widget degrades gracefully)
	fontFace, _ := helper.LoadFontForTextMode(displayMode)

	// Get poll interval from config, fall back to default
	pollInterval := time.Duration(config.DefaultPollInterval * float64(time.Second))
	if cfg.PollInterval > 0 {
		pollInterval = time.Duration(cfg.PollInterval * float64(time.Second))
	}

	w := &VolumeWidget{
		BaseWidget:       base,
		displayMode:      displayMode,
		fillColor:        uint8(fillColor),
		barDirection:     barSettings.Direction,
		barBorder:        barSettings.Border,
		gaugeColor:       uint8(gaugeSettings.ArcColor),
		gaugeNeedleColor: uint8(gaugeSettings.NeedleColor),
		gaugeShowTicks:   gaugeSettings.ShowTicks,
		gaugeTicksColor:  uint8(gaugeSettings.TicksColor),
		fontName:         textSettings.FontName,
		horizAlign:       textSettings.HorizAlign,
		vertAlign:        textSettings.VertAlign,
		padding:          padding,
		pollInterval:     pollInterval,
		lastSuccessTime:  time.Now(), // Initialize to prevent false "stuck" detection
		errorThreshold:   30,         // ~3 seconds at 100ms poll interval
		face:             fontFace,
		stopChan:         make(chan struct{}),
	}

	// Start single background goroutine for polling volume
	// Note: Reader is created INSIDE the goroutine due to Windows COM thread affinity -
	// COM objects must be created and used on the same thread
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

	// Create reader on this goroutine due to Windows COM thread affinity -
	// COM objects must be created and used on the same thread
	reader, err := newVolumeReader()
	if err != nil {
		log.Printf("[VOLUME] Failed to initialize volume reader: %v", err)
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

	// Subscribe to device change notifications (if available)
	var deviceNotifyChan <-chan struct{}
	deviceNotifier, err := GetDeviceNotifier()
	if err != nil {
		log.Printf("[VOLUME] Device notifier not available: %v (will rely on polling)", err)
	} else {
		deviceNotifyChan = deviceNotifier.Subscribe()
		defer deviceNotifier.Unsubscribe(deviceNotifyChan)
		log.Printf("[VOLUME] Subscribed to device change notifications")
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Do initial poll immediately (ticker won't fire until first interval)
	w.pollOnce()

	for {
		select {
		case <-w.stopChan:
			return

		case <-deviceNotifyChan:
			// Device change detected - reinitialize reader
			if reinitReader, ok := w.reader.(reinitializableVolumeReader); ok {
				log.Printf("[VOLUME] Device change notification received, reinitializing...")
				if err := reinitReader.Reinitialize(); err != nil {
					log.Printf("[VOLUME] Failed to reinitialize after device change: %v", err)
				} else {
					// Reset error state on successful reinit
					w.mu.Lock()
					w.errorWidget = nil
					w.consecutiveErrors = 0
					w.mu.Unlock()
					log.Printf("[VOLUME] Recovered from error state after device change")
				}
			}

		case <-ticker.C:
			w.pollOnce()
		}
	}
}

// pollOnce performs a single volume poll and updates the widget state
func (w *VolumeWidget) pollOnce() {
	// Check if reader needs reinitialization (device may have changed)
	if reinitReader, ok := w.reader.(reinitializableVolumeReader); ok {
		if reinitReader.NeedsReinitialize() {
			log.Printf("[VOLUME] Reader needs reinitialization, attempting...")
			if err := reinitReader.Reinitialize(); err != nil {
				log.Printf("[VOLUME] Failed to reinitialize: %v", err)
				// Continue anyway, will retry next cycle
			}
		}
	}

	// Call volume reader
	volume, muted, err := w.reader.GetVolume()

	w.mu.Lock()
	defer w.mu.Unlock()

	w.totalCalls++

	if err != nil {
		// Log first error and state changes for debugging
		if w.consecutiveErrors == 0 {
			log.Printf("[VOLUME] Error reading volume: %v", err)
		}
		w.failedCalls++
		w.consecutiveErrors++

		// Enter error state if threshold reached
		if w.consecutiveErrors >= w.errorThreshold && w.errorWidget == nil {
			pos := w.GetPosition()
			w.errorWidget = NewErrorWidget(pos.W, pos.H, "VOLUME ERROR")
			log.Printf("[VOLUME] Entered error state after %d consecutive errors", w.consecutiveErrors)
		}
		return
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
}

// Update is called periodically but just returns immediately
// All volume polling happens in the background goroutine
func (w *VolumeWidget) Update() error {
	w.mu.RLock()
	errorWidget := w.errorWidget
	w.mu.RUnlock()

	// Delegate to error widget if in error state
	if errorWidget != nil {
		return errorWidget.Update()
	}
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

	// Delegate to error widget if in error state
	if w.errorWidget != nil {
		return w.errorWidget.Render()
	}

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
	default:
		w.renderBarHorizontal(img, pos, style)
	}

	return img, nil
}

// renderText renders volume as text
func (w *VolumeWidget) renderText(img *image.Gray) {
	// Skip rendering if no font available (neither TTF nor internal)
	if w.face == nil && !bitmap.IsInternalFont(w.fontName) {
		return
	}

	text := fmt.Sprintf("%.0f%%", w.volume)
	if w.isMuted {
		text = "MUTE"
	}

	// Draw text with configured alignment
	bitmap.SmartDrawAlignedText(img, text, w.face, w.fontName, w.horizAlign, w.vertAlign, w.padding)
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
		bitmap.DrawCrossPattern(img, 0, 0, pos.W, pos.H, 2, 128)
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
		bitmap.DrawCrossPattern(img, 0, 0, pos.W, pos.H, 2, 128)
	}
}

// renderGauge renders volume as an old-fashioned gauge with needle
func (w *VolumeWidget) renderGauge(img *image.Gray, pos config.PositionConfig) {
	// Use shared gauge drawing function
	bitmap.DrawGauge(img, 0, 0, pos.W, pos.H, w.volume, w.gaugeColor, w.gaugeNeedleColor, w.gaugeShowTicks, w.gaugeTicksColor)

	// Draw mute indicator
	if w.isMuted {
		bitmap.DrawCrossPattern(img, 0, 0, pos.W, pos.H, 2, 128)
	}
}
