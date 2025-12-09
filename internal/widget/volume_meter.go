package widget

import (
	"fmt"
	"image"
	"log"
	"math"
	"runtime/debug"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
	"golang.org/x/image/font"
)

func init() {
	Register("volume_meter", func(cfg config.WidgetConfig) (Widget, error) {
		return NewVolumeMeterWidget(cfg)
	})
}

// MeterData holds current meter readings
type MeterData struct {
	Peak         float64   // Overall peak value (0.0-1.0)
	ChannelPeaks []float64 // Per-channel peak values
	ChannelCount int       // Number of channels
	IsClipping   bool      // True if peak >= clipping threshold
	HasAudio     bool      // True if peak > silence threshold
}

// meterReader interface abstracts platform-specific audio meter reading
type meterReader interface {
	GetMeterData(clippingThreshold, silenceThreshold float64) (*MeterData, error)
	Close()
}

// reinitializableMeterReader extends meterReader with reinitialize capability
type reinitializableMeterReader interface {
	meterReader
	Reinitialize() error
	NeedsReinitialize() bool
}

// VolumeMeterWidget displays realtime audio output levels
type VolumeMeterWidget struct {
	*BaseWidget
	displayMode      string
	fillColor        uint8
	clippingColor    uint8
	peakColor        uint8
	gaugeColor       uint8
	gaugeNeedleColor uint8
	gaugeShowTicks   bool
	gaugeTicksColor  uint8
	fontName         string
	horizontalAlign  string
	verticalAlign    string
	padding          int

	// Meter configuration
	pollInterval        time.Duration // Configurable internal polling rate
	stereoMode          bool
	stereoDivider       int // Divider color between channels (-1=disabled, 0-255=color)
	useDBScale          bool
	showClipping        bool
	clippingThreshold   float64
	silenceThreshold    float64
	decayRate           float64 // normalized units per second (0.0-1.0/s)
	showPeakHold        bool
	peakHoldTime        time.Duration
	autoHideOnSilence   bool
	autoHideSilenceTime time.Duration

	mu             sync.RWMutex
	peak           float64   // Current overall peak (0.0-1.0)
	displayPeak    float64   // Peak with decay applied for display
	channelPeaks   []float64 // Per-channel peaks
	channelCount   int
	isClipping     bool
	hasAudio       bool
	peakHoldValues []float64   // Held peak values per channel
	peakHoldUntils []time.Time // When to release peak hold per channel
	lastUpdateTime time.Time

	face font.Face

	// Background polling
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Diagnostic metrics
	totalCalls        int64
	successfulCalls   int64
	failedCalls       int64
	lastSuccessTime   time.Time
	maxCallDuration   time.Duration
	consecutiveErrors int
	errorThreshold    int // Threshold before entering error state

	// Error state
	errorWidget *ErrorWidget // Error widget proxy (nil = normal operation)

	// Platform-specific meter reader
	reader meterReader
}

// NewVolumeMeterWidget creates a new volume meter widget
//
//nolint:gocyclo // Complex initialization logic for different display modes
func NewVolumeMeterWidget(cfg config.WidgetConfig) (*VolumeMeterWidget, error) {
	base := NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	// Extract common settings using helper
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()
	barSettings := helper.GetBarSettings()
	gaugeSettings := helper.GetGaugeSettings()

	// Display mode - translate schema mode to internal mode
	displayMode := helper.GetDisplayMode("bar")

	// Handle "bar" mode by checking bar.direction
	if displayMode == "bar" {
		if barSettings.Direction == "vertical" {
			displayMode = "bar_vertical"
		} else {
			displayMode = "bar_horizontal"
		}
	}

	// Validate display mode (internal modes)
	validModes := map[string]bool{
		"text":           true,
		"bar_horizontal": true,
		"bar_vertical":   true,
		"gauge":          true,
	}
	if !validModes[displayMode] {
		return nil, fmt.Errorf("invalid display mode: %s (valid: text, bar, gauge)", displayMode)
	}

	// Extract colors based on active display mode only
	fillColor := 255
	clippingColor := 200
	peakColor := 180

	// Bar mode colors (for bar_horizontal and bar_vertical)
	if displayMode == "bar_horizontal" || displayMode == "bar_vertical" {
		if cfg.Bar != nil && cfg.Bar.Colors != nil {
			if cfg.Bar.Colors.Fill != nil {
				fillColor = *cfg.Bar.Colors.Fill
			}
			if cfg.Bar.Colors.Clipping != nil {
				clippingColor = *cfg.Bar.Colors.Clipping
			}
			if cfg.Bar.Colors.Peak != nil {
				peakColor = *cfg.Bar.Colors.Peak
			}
		}
	}

	// Gauge mode colors
	if displayMode == "gauge" {
		if cfg.Gauge != nil && cfg.Gauge.Colors != nil {
			if cfg.Gauge.Colors.Clipping != nil {
				clippingColor = *cfg.Gauge.Colors.Clipping
			}
			if cfg.Gauge.Colors.Peak != nil {
				peakColor = *cfg.Gauge.Colors.Peak
			}
		}
	}

	// Text mode doesn't use fill colors - text is rendered via font glyphs

	// Clipping settings (no color field - moved to mode colors)
	clippingThreshold := 0.99
	showClipping := false

	if cfg.Clipping != nil {
		showClipping = cfg.Clipping.Enabled
		if cfg.Clipping.Threshold > 0 {
			clippingThreshold = cfg.Clipping.Threshold
		}
	}

	// Stereo settings (includes divider color between channels)
	stereoMode := false
	stereoDivider := 64 // Default: mid-gray divider

	if cfg.Stereo != nil {
		stereoMode = cfg.Stereo.Enabled
		if cfg.Stereo.Divider != nil {
			stereoDivider = *cfg.Stereo.Divider
		}
	}

	// Metering settings
	silenceThreshold := 0.01
	decayRate := 2.0
	useDBScale := false

	if cfg.Metering != nil {
		useDBScale = cfg.Metering.DBScale
		if cfg.Metering.DecayRate > 0 {
			decayRate = cfg.Metering.DecayRate
		}
		if cfg.Metering.SilenceThreshold > 0 {
			silenceThreshold = cfg.Metering.SilenceThreshold
		}
	}

	// Peak hold settings
	peakHoldTime := 1 * time.Second
	showPeakHold := false

	if cfg.Peak != nil {
		showPeakHold = cfg.Peak.Enabled
		if cfg.Peak.HoldTime > 0 {
			peakHoldTime = time.Duration(cfg.Peak.HoldTime * float64(time.Second))
		}
	}

	// Auto-hide settings
	autoHideOnSilence := false
	autoHideSilenceTime := 2 * time.Second

	if cfg.AutoHide != nil {
		autoHideOnSilence = cfg.AutoHide.OnSilence
		if cfg.AutoHide.SilenceTime > 0 {
			autoHideSilenceTime = time.Duration(cfg.AutoHide.SilenceTime * float64(time.Second))
		}
	}

	// Get poll interval from config, fall back to default
	pollInterval := time.Duration(config.DefaultPollInterval * float64(time.Second))
	if cfg.PollInterval > 0 {
		pollInterval = time.Duration(cfg.PollInterval * float64(time.Second))
	}

	// Load font for text mode (ignore error - degrades gracefully)
	fontFace, _ := bitmap.LoadFontForTextMode(displayMode, textSettings.FontName, textSettings.FontSize)

	w := &VolumeMeterWidget{
		BaseWidget:          base,
		displayMode:         displayMode,
		fillColor:           uint8(fillColor),
		clippingColor:       uint8(clippingColor),
		peakColor:           uint8(peakColor),
		gaugeColor:          uint8(gaugeSettings.ArcColor),
		gaugeNeedleColor:    uint8(gaugeSettings.NeedleColor),
		gaugeShowTicks:      gaugeSettings.ShowTicks,
		gaugeTicksColor:     uint8(gaugeSettings.TicksColor),
		fontName:            textSettings.FontName,
		horizontalAlign:     textSettings.HorizAlign,
		verticalAlign:       textSettings.VertAlign,
		padding:             padding,
		pollInterval:        pollInterval,
		stereoMode:          stereoMode,
		stereoDivider:       stereoDivider,
		useDBScale:          useDBScale,
		showClipping:        showClipping,
		clippingThreshold:   clippingThreshold,
		silenceThreshold:    silenceThreshold,
		decayRate:           decayRate,
		showPeakHold:        showPeakHold,
		peakHoldTime:        peakHoldTime,
		autoHideOnSilence:   autoHideOnSilence,
		autoHideSilenceTime: autoHideSilenceTime,
		lastSuccessTime:     time.Now(),
		lastUpdateTime:      time.Now(),
		errorThreshold:      30, // ~3 seconds at 100ms poll interval
		face:                fontFace,
		stopChan:            make(chan struct{}),
	}

	// Start background polling goroutine
	// Note: Reader is created INSIDE the goroutine due to Windows COM thread affinity -
	// COM objects must be created and used on the same thread
	w.wg.Add(1)
	go w.pollMeterBackground()

	return w, nil
}

// pollMeterBackground continuously polls audio meter in a single background goroutine
func (w *VolumeMeterWidget) pollMeterBackground() {
	defer w.wg.Done()

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in polling goroutine: %v\nStack: %s", r, debug.Stack())
		}
	}()

	// Create reader on this goroutine due to Windows COM thread affinity -
	// COM objects must be created and used on the same thread
	reader, err := newMeterReader()
	if err != nil {
		log.Printf("[VOLUME-METER] Failed to initialize meter reader: %v", err)
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
		log.Printf("[VOLUME-METER] Device notifier not available: %v (will rely on polling)", err)
	} else {
		deviceNotifyChan = deviceNotifier.Subscribe()
		defer deviceNotifier.Unsubscribe(deviceNotifyChan)
		log.Printf("[VOLUME-METER] Subscribed to device change notifications")
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Do initial poll immediately (ticker won't fire until first interval)
	w.updateMeter()

	for {
		select {
		case <-w.stopChan:
			return

		case <-deviceNotifyChan:
			// Device change detected - reinitialize reader
			if reinitReader, ok := w.reader.(reinitializableMeterReader); ok {
				log.Printf("[VOLUME-METER] Device change notification received, reinitializing...")
				if err := reinitReader.Reinitialize(); err != nil {
					log.Printf("[VOLUME-METER] Failed to reinitialize after device change: %v", err)
				} else {
					// Reset error state on successful reinit
					w.mu.Lock()
					w.errorWidget = nil
					w.consecutiveErrors = 0
					w.mu.Unlock()
					log.Printf("[VOLUME-METER] Recovered from error state after device change")
				}
			}

		case <-ticker.C:
			w.updateMeter()
		}
	}
}

// updateMeter reads meter data and updates widget state
//
//nolint:gocyclo // Complex state management for stereo/mono channels
func (w *VolumeMeterWidget) updateMeter() {
	startTime := time.Now()
	w.mu.Lock()
	w.totalCalls++
	w.mu.Unlock()

	// Check if reader needs reinitialization (device may have changed)
	if reinitReader, ok := w.reader.(reinitializableMeterReader); ok {
		if reinitReader.NeedsReinitialize() {
			log.Printf("[VOLUME-METER] Reader needs reinitialization, attempting...")
			if err := reinitReader.Reinitialize(); err != nil {
				log.Printf("[VOLUME-METER] Failed to reinitialize: %v", err)
				// Continue anyway, will retry next cycle
			}
		}
	}

	// Read meter data
	data, err := w.reader.GetMeterData(w.clippingThreshold, w.silenceThreshold)
	callDuration := time.Since(startTime)

	w.mu.Lock()
	defer w.mu.Unlock()

	// Update metrics
	if callDuration > w.maxCallDuration {
		w.maxCallDuration = callDuration
	}

	if err != nil {
		// Log first error for debugging
		if w.consecutiveErrors == 0 {
			log.Printf("[VOLUME-METER] Error reading meter: %v", err)
		}
		w.failedCalls++
		w.consecutiveErrors++

		// Enter error state if threshold reached
		if w.consecutiveErrors >= w.errorThreshold && w.errorWidget == nil {
			pos := w.GetPosition()
			w.errorWidget = NewErrorWidget(pos.W, pos.H, "METER ERROR")
			log.Printf("[VOLUME-METER] Entered error state after %d consecutive errors", w.consecutiveErrors)
		}
		return
	}

	// Successful read
	w.successfulCalls++
	w.consecutiveErrors = 0
	w.lastSuccessTime = time.Now()

	// Update state
	now := time.Now()
	timeDelta := now.Sub(w.lastUpdateTime).Seconds()
	w.lastUpdateTime = now

	w.peak = data.Peak
	w.channelPeaks = data.ChannelPeaks
	w.channelCount = data.ChannelCount
	w.isClipping = data.IsClipping
	w.hasAudio = data.HasAudio

	// Apply ballistics to display peak (smooth rise and fall like real VU meter)
	if w.peak > w.displayPeak {
		// Rising: apply rise ballistics (faster than decay, but not instant)
		rise := w.decayRate * 3.0 * timeDelta // Rise 3x faster than fall
		w.displayPeak += rise
		if w.displayPeak > w.peak {
			w.displayPeak = w.peak
		}
	} else if w.displayPeak > w.peak {
		// Falling: apply fall ballistics (decay) - only when displayPeak is above current peak
		decay := w.decayRate * timeDelta
		w.displayPeak -= decay
		if w.displayPeak < w.peak {
			w.displayPeak = w.peak // Don't fall below current peak
		}
	}
	// If w.peak == w.displayPeak, do nothing (already at current level)

	// Clamp to valid range
	if w.displayPeak < 0 {
		w.displayPeak = 0
	}
	if w.displayPeak > 1.0 {
		w.displayPeak = 1.0
	}

	// Peak hold (per-channel)
	if w.showPeakHold {
		// Ensure arrays are sized correctly
		if len(w.peakHoldValues) != len(w.channelPeaks) {
			w.peakHoldValues = make([]float64, len(w.channelPeaks))
			w.peakHoldUntils = make([]time.Time, len(w.channelPeaks))
		}

		// Update peak hold for each channel
		for i, chPeak := range w.channelPeaks {
			if chPeak > w.peakHoldValues[i] {
				w.peakHoldValues[i] = chPeak
				w.peakHoldUntils[i] = now.Add(w.peakHoldTime)
			} else if now.After(w.peakHoldUntils[i]) {
				w.peakHoldValues[i] = 0
			}
		}
	}

	// Auto-hide on silence
	if w.autoHideOnSilence && w.hasAudio {
		w.TriggerAutoHide()
	}
}

// Render renders the volume meter widget
//
//nolint:gocyclo // Multiple display modes require branching logic
func (w *VolumeMeterWidget) Render() (image.Image, error) {
	// Delegate to error widget if in error state
	w.mu.RLock()
	errorWidget := w.errorWidget
	w.mu.RUnlock()
	if errorWidget != nil {
		return errorWidget.Render()
	}

	// Check auto-hide
	if w.ShouldHide() {
		return nil, nil
	}

	w.mu.RLock()
	displayPeak := w.displayPeak // Decayed peak (for main display)
	channelPeaks := make([]float64, len(w.channelPeaks))
	copy(channelPeaks, w.channelPeaks)
	peakHoldValues := make([]float64, len(w.peakHoldValues))
	copy(peakHoldValues, w.peakHoldValues)
	isClipping := w.isClipping
	w.mu.RUnlock()

	// Convert to dB if needed
	if w.useDBScale {
		displayPeak = w.linearToDBNormalized(displayPeak)
		for i := range channelPeaks {
			channelPeaks[i] = w.linearToDBNormalized(channelPeaks[i])
		}
		for i := range peakHoldValues {
			peakHoldValues[i] = w.linearToDBNormalized(peakHoldValues[i])
		}
	}

	// Create canvas with background and border
	img := w.CreateCanvas()
	w.ApplyBorder(img)

	// Check if we should render in stereo mode
	if w.stereoMode && len(channelPeaks) >= 2 {
		// Render stereo version of each mode (uses per-channel peak holds)
		switch w.displayMode {
		case "text":
			w.renderTextStereo(img, channelPeaks, isClipping)
		case "bar_horizontal":
			w.renderBarHorizontalStereo(img, channelPeaks, peakHoldValues, isClipping)
		case "bar_vertical":
			w.renderBarVerticalStereo(img, channelPeaks, peakHoldValues, isClipping)
		case "gauge":
			w.renderGaugeStereo(img, channelPeaks, peakHoldValues, isClipping)
		}
	} else {
		// Render mono version (use max peak hold from all channels)
		monoPeakHold := 0.0
		if len(peakHoldValues) > 0 {
			for _, ph := range peakHoldValues {
				if ph > monoPeakHold {
					monoPeakHold = ph
				}
			}
		}
		switch w.displayMode {
		case "text":
			w.renderText(img, displayPeak, isClipping)
		case "bar_horizontal":
			w.renderBarHorizontal(img, displayPeak, monoPeakHold, isClipping)
		case "bar_vertical":
			w.renderBarVertical(img, displayPeak, monoPeakHold, isClipping)
		case "gauge":
			w.renderGauge(img, displayPeak, monoPeakHold, isClipping)
		}
	}

	return img, nil
}

// linearToDBNormalized converts linear value (0.0-1.0) to dB scale normalized to 0.0-1.0
// -60dB to 0dB mapped to 0.0 to 1.0
func (w *VolumeMeterWidget) linearToDBNormalized(linear float64) float64 {
	if linear <= 0.0 {
		return 0.0
	}
	// Convert to dB: 20 * log10(linear)
	db := 20.0 * math.Log10(linear)
	// Map -60dB to 0dB => 0.0 to 1.0
	const minDB = -60.0
	const maxDB = 0.0
	normalized := (db - minDB) / (maxDB - minDB)
	if normalized < 0 {
		normalized = 0
	}
	if normalized > 1 {
		normalized = 1
	}
	return normalized
}

// renderText renders text display mode
func (w *VolumeMeterWidget) renderText(img *image.Gray, peak float64, isClipping bool) {
	if w.face == nil {
		return
	}

	var text string
	if w.useDBScale {
		// Convert normalized dB back to actual dB for display
		db := (peak * 60.0) - 60.0 // 0.0-1.0 => -60 to 0 dB
		text = fmt.Sprintf("%.1f dB", db)
	} else {
		text = fmt.Sprintf("%.0f%%", peak*100)
	}

	if isClipping && w.showClipping {
		text += " CLIP"
	}

	bitmap.SmartDrawAlignedText(img, text, w.face, w.fontName, w.horizontalAlign, w.verticalAlign, w.padding)
}

// renderBarHorizontal renders horizontal bar display
func (w *VolumeMeterWidget) renderBarHorizontal(img *image.Gray, displayPeak, peakHold float64, isClipping bool) {
	pos := w.GetPosition()
	barWidth := int(float64(pos.W) * displayPeak)

	fillColor := w.fillColor
	if isClipping && w.showClipping {
		fillColor = w.clippingColor
	}

	// Draw filled bar
	if barWidth > 0 {
		bitmap.DrawFilledRectangle(img, 0, 0, barWidth, pos.H, fillColor)
	}

	// Draw peak hold line
	if w.showPeakHold && peakHold > 0 {
		peakX := int(float64(pos.W) * peakHold)
		if peakX < pos.W {
			bitmap.DrawVerticalLine(img, peakX, 0, pos.H-1, w.peakColor)
		}
	}
}

// renderBarVertical renders vertical bar display
func (w *VolumeMeterWidget) renderBarVertical(img *image.Gray, displayPeak, peakHold float64, isClipping bool) {
	pos := w.GetPosition()
	barHeight := int(float64(pos.H) * displayPeak)
	startY := pos.H - barHeight

	fillColor := w.fillColor
	if isClipping && w.showClipping {
		fillColor = w.clippingColor
	}

	// Draw filled bar (from bottom)
	if barHeight > 0 {
		bitmap.DrawFilledRectangle(img, 0, startY, pos.W, barHeight, fillColor)
	}

	// Draw peak hold line
	if w.showPeakHold && peakHold > 0 {
		peakY := pos.H - int(float64(pos.H)*peakHold)
		if peakY >= 0 {
			bitmap.DrawHorizontalLine(img, 0, pos.W-1, peakY, w.peakColor)
		}
	}
}

// renderGauge renders gauge display
func (w *VolumeMeterWidget) renderGauge(img *image.Gray, peak, peakHold float64, isClipping bool) {
	pos := w.GetPosition()

	needleColor := w.gaugeNeedleColor
	if isClipping && w.showClipping {
		needleColor = w.clippingColor
	}

	// DrawGauge expects percentage as 0-100, not 0.0-1.0
	percentage := peak * 100.0
	bitmap.DrawGauge(img, 0, 0, pos.W, pos.H, percentage, w.gaugeColor, needleColor, w.gaugeShowTicks, w.gaugeTicksColor)

	// Draw peak hold mark if enabled
	if w.showPeakHold && peakHold > 0 {
		w.drawGaugePeakHoldMark(img, pos, peakHold)
	}
}

// renderBarHorizontalStereo renders horizontal bars in stereo mode (left/right channels)
//
//nolint:gocyclo // Geometric calculations for stereo bar rendering
func (w *VolumeMeterWidget) renderBarHorizontalStereo(img *image.Gray, channelPeaks []float64, peakHoldValues []float64, isClipping bool) {
	pos := w.GetPosition()

	if len(channelPeaks) < 2 {
		// Not stereo, fall back to mono display
		peak := 0.0
		peakHold := 0.0
		if len(channelPeaks) > 0 {
			peak = channelPeaks[0]
		}
		if len(peakHoldValues) > 0 {
			peakHold = peakHoldValues[0]
		}
		w.renderBarHorizontal(img, peak, peakHold, isClipping)
		return
	}

	// Draw two bars: top half = left, bottom half = right
	halfHeight := pos.H / 2

	fillColor := w.fillColor
	if isClipping && w.showClipping {
		fillColor = w.clippingColor
	}

	// Left channel (top)
	leftWidth := int(float64(pos.W) * channelPeaks[0])
	if leftWidth > 0 {
		bitmap.DrawFilledRectangle(img, 0, 0, leftWidth, halfHeight, fillColor)
	}

	// Right channel (bottom)
	rightWidth := int(float64(pos.W) * channelPeaks[1])
	if rightWidth > 0 {
		bitmap.DrawFilledRectangle(img, 0, halfHeight, rightWidth, pos.H-halfHeight, fillColor)
	}

	// Draw peak hold lines per channel
	if w.showPeakHold && len(peakHoldValues) >= 2 {
		// Left channel peak hold (top half)
		if peakHoldValues[0] > 0 {
			leftPeakX := int(float64(pos.W) * peakHoldValues[0])
			if leftPeakX < pos.W {
				bitmap.DrawVerticalLine(img, leftPeakX, 0, halfHeight-1, w.peakColor)
			}
		}

		// Right channel peak hold (bottom half)
		if peakHoldValues[1] > 0 {
			rightPeakX := int(float64(pos.W) * peakHoldValues[1])
			if rightPeakX < pos.W {
				bitmap.DrawVerticalLine(img, rightPeakX, halfHeight, pos.H-1, w.peakColor)
			}
		}
	}

	// Draw separator between channels (if enabled)
	if w.stereoDivider >= 0 {
		bitmap.DrawHorizontalLine(img, 0, pos.W-1, halfHeight, uint8(w.stereoDivider))
	}
}

// renderTextStereo renders text display in stereo mode showing L/R channel values
func (w *VolumeMeterWidget) renderTextStereo(img *image.Gray, channelPeaks []float64, isClipping bool) {
	if w.face == nil {
		return
	}

	if len(channelPeaks) < 2 {
		// Fall back to mono display
		peak := 0.0
		if len(channelPeaks) > 0 {
			peak = channelPeaks[0]
		}
		w.renderText(img, peak, isClipping)
		return
	}

	var text string
	if w.useDBScale {
		// Convert normalized dB back to actual dB for display
		leftDB := (channelPeaks[0] * 60.0) - 60.0
		rightDB := (channelPeaks[1] * 60.0) - 60.0
		text = fmt.Sprintf("L:%.1f R:%.1f dB", leftDB, rightDB)
	} else {
		text = fmt.Sprintf("L:%.0f%% R:%.0f%%", channelPeaks[0]*100, channelPeaks[1]*100)
	}

	if isClipping && w.showClipping {
		text += " CLIP"
	}

	bitmap.SmartDrawAlignedText(img, text, w.face, w.fontName, w.horizontalAlign, w.verticalAlign, w.padding)

	// Draw separator between channels (if enabled)
	if w.stereoDivider >= 0 {
		pos := w.GetPosition()
		halfWidth := pos.W / 2
		bitmap.DrawVerticalLine(img, halfWidth, 0, pos.H-1, uint8(w.stereoDivider))
	}
}

// renderBarVerticalStereo renders vertical bars in stereo mode (left/right channels)
//
//nolint:gocyclo // Geometric calculations for stereo bar rendering
func (w *VolumeMeterWidget) renderBarVerticalStereo(img *image.Gray, channelPeaks []float64, peakHoldValues []float64, isClipping bool) {
	pos := w.GetPosition()

	if len(channelPeaks) < 2 {
		// Not stereo, fall back to mono display
		peak := 0.0
		peakHold := 0.0
		if len(channelPeaks) > 0 {
			peak = channelPeaks[0]
		}
		if len(peakHoldValues) > 0 {
			peakHold = peakHoldValues[0]
		}
		w.renderBarVertical(img, peak, peakHold, isClipping)
		return
	}

	// Draw two bars: left half = left channel, right half = right channel
	halfWidth := pos.W / 2

	fillColor := w.fillColor
	if isClipping && w.showClipping {
		fillColor = w.clippingColor
	}

	// Left channel (left half)
	leftHeight := int(float64(pos.H) * channelPeaks[0])
	if leftHeight > 0 {
		bitmap.DrawFilledRectangle(img, 0, pos.H-leftHeight, halfWidth, leftHeight, fillColor)
	}

	// Right channel (right half)
	rightHeight := int(float64(pos.H) * channelPeaks[1])
	if rightHeight > 0 {
		bitmap.DrawFilledRectangle(img, halfWidth, pos.H-rightHeight, pos.W-halfWidth, rightHeight, fillColor)
	}

	// Draw peak hold lines per channel
	if w.showPeakHold && len(peakHoldValues) >= 2 {
		// Left channel peak hold (left half)
		if peakHoldValues[0] > 0 {
			leftPeakY := pos.H - int(float64(pos.H)*peakHoldValues[0])
			if leftPeakY >= 0 && leftPeakY < pos.H {
				bitmap.DrawHorizontalLine(img, 0, halfWidth-1, leftPeakY, w.peakColor)
			}
		}

		// Right channel peak hold (right half)
		if peakHoldValues[1] > 0 {
			rightPeakY := pos.H - int(float64(pos.H)*peakHoldValues[1])
			if rightPeakY >= 0 && rightPeakY < pos.H {
				bitmap.DrawHorizontalLine(img, halfWidth, pos.W-1, rightPeakY, w.peakColor)
			}
		}
	}

	// Draw separator between channels (if enabled)
	if w.stereoDivider >= 0 {
		bitmap.DrawVerticalLine(img, halfWidth, 0, pos.H-1, uint8(w.stereoDivider))
	}
}

// renderGaugeStereo renders gauges in stereo mode (left/right channels)
//
//nolint:gocyclo // Geometric calculations for stereo gauge rendering
func (w *VolumeMeterWidget) renderGaugeStereo(img *image.Gray, channelPeaks []float64, peakHoldValues []float64, isClipping bool) {
	pos := w.GetPosition()

	if len(channelPeaks) < 2 {
		// Not stereo, fall back to mono display
		peak := 0.0
		peakHold := 0.0
		if len(channelPeaks) > 0 {
			peak = channelPeaks[0]
		}
		if len(peakHoldValues) > 0 {
			peakHold = peakHoldValues[0]
		}
		w.renderGauge(img, peak, peakHold, isClipping)
		return
	}

	// Draw two gauges side by side using sub-images
	halfWidth := pos.W / 2

	// Create left gauge sub-image
	leftImg := image.NewGray(image.Rect(0, 0, halfWidth-1, pos.H))
	leftGaugePos := config.PositionConfig{
		X: 0,
		Y: 0,
		W: halfWidth - 1,
		H: pos.H,
	}
	leftPercentage := channelPeaks[0] * 100.0
	leftNeedleColor := w.gaugeNeedleColor
	if isClipping && w.showClipping {
		leftNeedleColor = w.clippingColor
	}
	bitmap.DrawGauge(leftImg, 0, 0, leftGaugePos.W, leftGaugePos.H, leftPercentage, w.gaugeColor, leftNeedleColor, w.gaugeShowTicks, w.gaugeTicksColor)

	// Draw left channel peak hold mark
	if w.showPeakHold && len(peakHoldValues) >= 1 && peakHoldValues[0] > 0 {
		w.drawGaugePeakHoldMark(leftImg, leftGaugePos, peakHoldValues[0])
	}

	// Copy left gauge to main image
	for y := 0; y < pos.H; y++ {
		for x := 0; x < halfWidth-1; x++ {
			img.SetGray(x, y, leftImg.GrayAt(x, y))
		}
	}

	// Create right gauge sub-image
	rightImg := image.NewGray(image.Rect(0, 0, halfWidth-1, pos.H))
	rightGaugePos := config.PositionConfig{
		X: 0,
		Y: 0,
		W: halfWidth - 1,
		H: pos.H,
	}
	rightPercentage := channelPeaks[1] * 100.0
	rightNeedleColor := w.gaugeNeedleColor
	if isClipping && w.showClipping {
		rightNeedleColor = w.clippingColor
	}
	bitmap.DrawGauge(rightImg, 0, 0, rightGaugePos.W, rightGaugePos.H, rightPercentage, w.gaugeColor, rightNeedleColor, w.gaugeShowTicks, w.gaugeTicksColor)

	// Draw right channel peak hold mark
	if w.showPeakHold && len(peakHoldValues) >= 2 && peakHoldValues[1] > 0 {
		w.drawGaugePeakHoldMark(rightImg, rightGaugePos, peakHoldValues[1])
	}

	// Copy right gauge to main image (offset by halfWidth + 1)
	for y := 0; y < pos.H; y++ {
		for x := 0; x < halfWidth-1; x++ {
			img.SetGray(halfWidth+1+x, y, rightImg.GrayAt(x, y))
		}
	}

	// Draw separator between channels (if enabled)
	if w.stereoDivider >= 0 {
		bitmap.DrawVerticalLine(img, halfWidth, 0, pos.H-1, uint8(w.stereoDivider))
	}
}

// drawGaugePeakHoldMark draws a small mark on the gauge arc at the peak hold position
func (w *VolumeMeterWidget) drawGaugePeakHoldMark(img *image.Gray, pos config.PositionConfig, peakHold float64) {
	centerX := pos.W / 2
	centerY := pos.H - 3

	// Calculate radius (same as in DrawGauge)
	radius := pos.H - 6
	if pos.W/2 < radius {
		radius = pos.W/2 - 3
	}

	bitmap.DrawGaugePeakHoldMark(img, centerX, centerY, radius, peakHold, 5, w.peakColor)
}

// Update is called periodically but just returns immediately
// All meter polling happens in the background goroutine
func (w *VolumeMeterWidget) Update() error {
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
func (w *VolumeMeterWidget) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}
