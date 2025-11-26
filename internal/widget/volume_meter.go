package widget

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"runtime/debug"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

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

// VolumeMeterWidget displays realtime audio output levels
type VolumeMeterWidget struct {
	*BaseWidget
	displayMode       string
	fillColor         uint8
	clippingColor     uint8
	leftChannelColor  uint8
	rightChannelColor uint8
	barBorder         bool
	borderColor       uint8
	gaugeColor        uint8
	gaugeNeedleColor  uint8
	gaugeShowTicks    bool
	gaugeTicksColor   uint8
	horizontalAlign   string
	verticalAlign     string
	padding           int

	// Meter configuration
	stereoMode          bool
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

	// Platform-specific meter reader
	reader meterReader
}

// NewVolumeMeterWidget creates a new volume meter widget
//
//nolint:gocyclo // Complex initialization logic for different display modes
func NewVolumeMeterWidget(cfg config.WidgetConfig) (*VolumeMeterWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

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

	// Extract colors
	fillColor := 255
	clippingColor := 200
	leftChannelColor := 255
	rightChannelColor := 200

	if cfg.Colors != nil {
		if cfg.Colors.Fill != nil {
			fillColor = *cfg.Colors.Fill
		}
	}

	// Clipping settings (includes color)
	clippingThreshold := 0.99
	showClipping := false

	if cfg.Clipping != nil {
		showClipping = cfg.Clipping.Enabled
		if cfg.Clipping.Threshold > 0 {
			clippingThreshold = cfg.Clipping.Threshold
		}
		if cfg.Clipping.Color != nil {
			clippingColor = *cfg.Clipping.Color
		}
	}

	// Stereo settings (includes channel colors)
	stereoMode := false

	if cfg.Stereo != nil {
		stereoMode = cfg.Stereo.Enabled
		if cfg.Stereo.LeftColor != nil {
			leftChannelColor = *cfg.Stereo.LeftColor
		}
		if cfg.Stereo.RightColor != nil {
			rightChannelColor = *cfg.Stereo.RightColor
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

	// Load font for text mode (ignore error - degrades gracefully)
	fontFace, _ := helper.LoadFontForTextMode(displayMode)

	// Border settings (combine bar and style)
	barBorder := barSettings.Border
	style := config.StyleConfig{Border: -1} // Default: disabled
	if cfg.Style != nil {
		style = *cfg.Style
		if style.Border >= 0 {
			barBorder = true
		}
	}

	// Get border color from style config (use the border value itself as color)
	borderColor := style.Border
	if borderColor < 0 {
		borderColor = 255 // Default: bright white if somehow enabled without color
	}

	w := &VolumeMeterWidget{
		BaseWidget:          base,
		displayMode:         displayMode,
		fillColor:           uint8(fillColor),
		clippingColor:       uint8(clippingColor),
		leftChannelColor:    uint8(leftChannelColor),
		rightChannelColor:   uint8(rightChannelColor),
		barBorder:           barBorder,
		borderColor:         uint8(borderColor),
		gaugeColor:          uint8(gaugeSettings.ArcColor),
		gaugeNeedleColor:    uint8(gaugeSettings.NeedleColor),
		gaugeShowTicks:      gaugeSettings.ShowTicks,
		gaugeTicksColor:     uint8(gaugeSettings.TicksColor),
		horizontalAlign:     textSettings.HorizAlign,
		verticalAlign:       textSettings.VertAlign,
		padding:             padding,
		stereoMode:          stereoMode,
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
		face:                fontFace,
		stopChan:            make(chan struct{}),
	}

	// Initialize meter reader BEFORE starting goroutine
	// This ensures widget creation fails if no audio device exists (fail-fast pattern)
	reader, err := newMeterReader()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize meter reader: %w", err)
	}
	w.reader = reader

	// Start background polling goroutine
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

	// Reader already initialized in NewVolumeMeterWidget (fail-fast pattern)
	// Ensure cleanup when goroutine exits
	defer func() {
		if w.reader != nil {
			w.reader.Close()
			w.reader = nil
		}
	}()

	pollInterval := 100 * time.Millisecond // 10Hz default
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
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
		// FIXME: Silent error swallowing - see volume.go for suggested improvements.
		w.failedCalls++
		w.consecutiveErrors++
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

	pos := w.GetPosition()
	img := image.NewGray(image.Rect(0, 0, pos.W, pos.H))

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

	bitmap.DrawAlignedText(img, text, w.face, w.horizontalAlign, w.verticalAlign, w.padding)
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
	for y := 0; y < pos.H; y++ {
		for x := 0; x < barWidth; x++ {
			img.SetGray(x, y, color.Gray{Y: fillColor})
		}
	}

	// Draw peak hold line
	if w.showPeakHold && peakHold > 0 {
		peakX := int(float64(pos.W) * peakHold)
		if peakX < pos.W {
			for y := 0; y < pos.H; y++ {
				img.SetGray(peakX, y, color.Gray{Y: 180})
			}
		}
	}

	// Draw border
	if w.barBorder {
		bitmap.DrawBorder(img, w.borderColor)
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
	for y := startY; y < pos.H; y++ {
		for x := 0; x < pos.W; x++ {
			img.SetGray(x, y, color.Gray{Y: fillColor})
		}
	}

	// Draw peak hold line
	if w.showPeakHold && peakHold > 0 {
		peakY := pos.H - int(float64(pos.H)*peakHold)
		if peakY >= 0 {
			for x := 0; x < pos.W; x++ {
				img.SetGray(x, peakY, color.Gray{Y: 180})
			}
		}
	}

	// Draw border
	if w.barBorder {
		bitmap.DrawBorder(img, w.borderColor)
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

	// Left channel (top)
	leftWidth := int(float64(pos.W) * channelPeaks[0])
	for y := 0; y < halfHeight; y++ {
		for x := 0; x < leftWidth; x++ {
			img.SetGray(x, y, color.Gray{Y: w.leftChannelColor})
		}
	}

	// Right channel (bottom)
	rightWidth := int(float64(pos.W) * channelPeaks[1])
	for y := halfHeight; y < pos.H; y++ {
		for x := 0; x < rightWidth; x++ {
			img.SetGray(x, y, color.Gray{Y: w.rightChannelColor})
		}
	}

	// Draw peak hold lines per channel (slightly dimmer)
	if w.showPeakHold && len(peakHoldValues) >= 2 {
		// Left channel peak hold (top half)
		if peakHoldValues[0] > 0 {
			leftPeakX := int(float64(pos.W) * peakHoldValues[0])
			if leftPeakX < pos.W {
				for y := 0; y < halfHeight; y++ {
					img.SetGray(leftPeakX, y, color.Gray{Y: 180})
				}
			}
		}

		// Right channel peak hold (bottom half)
		if peakHoldValues[1] > 0 {
			rightPeakX := int(float64(pos.W) * peakHoldValues[1])
			if rightPeakX < pos.W {
				for y := halfHeight; y < pos.H; y++ {
					img.SetGray(rightPeakX, y, color.Gray{Y: 180})
				}
			}
		}
	}

	// Draw separator
	for x := 0; x < pos.W; x++ {
		img.SetGray(x, halfHeight, color.Gray{Y: 64})
	}

	// Draw border
	if w.barBorder {
		bitmap.DrawBorder(img, w.borderColor)
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

	bitmap.DrawAlignedText(img, text, w.face, w.horizontalAlign, w.verticalAlign, w.padding)
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

	// Left channel (left half)
	leftHeight := int(float64(pos.H) * channelPeaks[0])
	for y := pos.H - leftHeight; y < pos.H; y++ {
		for x := 0; x < halfWidth; x++ {
			img.SetGray(x, y, color.Gray{Y: w.leftChannelColor})
		}
	}

	// Right channel (right half)
	rightHeight := int(float64(pos.H) * channelPeaks[1])
	for y := pos.H - rightHeight; y < pos.H; y++ {
		for x := halfWidth; x < pos.W; x++ {
			img.SetGray(x, y, color.Gray{Y: w.rightChannelColor})
		}
	}

	// Draw peak hold lines per channel (slightly dimmer)
	if w.showPeakHold && len(peakHoldValues) >= 2 {
		// Left channel peak hold (left half)
		if peakHoldValues[0] > 0 {
			leftPeakY := pos.H - int(float64(pos.H)*peakHoldValues[0])
			if leftPeakY >= 0 && leftPeakY < pos.H {
				for x := 0; x < halfWidth; x++ {
					img.SetGray(x, leftPeakY, color.Gray{Y: 180})
				}
			}
		}

		// Right channel peak hold (right half)
		if peakHoldValues[1] > 0 {
			rightPeakY := pos.H - int(float64(pos.H)*peakHoldValues[1])
			if rightPeakY >= 0 && rightPeakY < pos.H {
				for x := halfWidth; x < pos.W; x++ {
					img.SetGray(x, rightPeakY, color.Gray{Y: 180})
				}
			}
		}
	}

	// Draw separator
	for y := 0; y < pos.H; y++ {
		img.SetGray(halfWidth, y, color.Gray{Y: 64})
	}

	// Draw borders if enabled
	if w.barBorder {
		// Left bar border
		for x := 0; x < halfWidth; x++ {
			img.SetGray(x, 0, color.Gray{Y: w.borderColor})
			img.SetGray(x, pos.H-1, color.Gray{Y: w.borderColor})
		}
		for y := 0; y < pos.H; y++ {
			img.SetGray(0, y, color.Gray{Y: w.borderColor})
			img.SetGray(halfWidth-1, y, color.Gray{Y: w.borderColor})
		}

		// Right bar border
		for x := halfWidth; x < pos.W; x++ {
			img.SetGray(x, 0, color.Gray{Y: w.borderColor})
			img.SetGray(x, pos.H-1, color.Gray{Y: w.borderColor})
		}
		for y := 0; y < pos.H; y++ {
			img.SetGray(halfWidth+1, y, color.Gray{Y: w.borderColor})
			img.SetGray(pos.W-1, y, color.Gray{Y: w.borderColor})
		}
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

	// Draw separator
	for y := 0; y < pos.H; y++ {
		img.SetGray(halfWidth, y, color.Gray{Y: 64})
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

	if radius <= 0 {
		return
	}

	// Calculate angle for peak hold position (180° to 0°)
	angle := 180.0 - (peakHold * 180.0)
	rad := angle * math.Pi / 180.0

	// Draw a small mark extending outward from the gauge arc
	markColor := color.Gray{Y: 180} // Same brightness as peak hold lines in bar modes
	tickLen := 5

	// Outer point (extended beyond arc)
	x1 := centerX + int(float64(radius+tickLen)*math.Cos(rad))
	y1 := centerY - int(float64(radius+tickLen)*math.Sin(rad))

	// Inner point (on the arc)
	x2 := centerX + int(float64(radius)*math.Cos(rad))
	y2 := centerY - int(float64(radius)*math.Sin(rad))

	bitmap.DrawLine(img, x1, y1, x2, y2, markColor)
}

// Update is called periodically but just returns immediately
// All meter polling happens in the background goroutine
func (w *VolumeMeterWidget) Update() error {
	// No-op: background goroutine handles all polling
	return nil
}

// Stop stops the background polling goroutine
func (w *VolumeMeterWidget) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}
