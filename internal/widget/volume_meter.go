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
	gaugeColor        uint8
	gaugeNeedleColor  uint8
	horizontalAlign   string
	verticalAlign     string

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
	peakHoldValue  float64   // Held peak value
	peakHoldUntil  time.Time // When to release peak hold
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
func NewVolumeMeterWidget(cfg config.WidgetConfig) (*VolumeMeterWidget, error) {
	base := NewBaseWidget(cfg)

	displayMode := cfg.Properties.DisplayMode
	if displayMode == "" {
		displayMode = "bar_horizontal"
	}

	// Validate display mode
	validModes := map[string]bool{
		"text":           true,
		"bar_horizontal": true,
		"bar_vertical":   true,
		"gauge":          true,
		"vu_meter":       true,
	}
	if !validModes[displayMode] {
		return nil, fmt.Errorf("invalid display mode: %s (valid: text, bar_horizontal, bar_vertical, gauge, vu_meter)", displayMode)
	}

	fillColor := cfg.Properties.FillColor
	if fillColor == 0 {
		fillColor = 255
	}

	clippingColor := cfg.Properties.ClippingColor
	if clippingColor == 0 {
		clippingColor = 200 // Red in grayscale
	}

	leftChannelColor := cfg.Properties.LeftChannelColor
	if leftChannelColor == 0 {
		leftChannelColor = 255
	}

	rightChannelColor := cfg.Properties.RightChannelColor
	if rightChannelColor == 0 {
		rightChannelColor = 200
	}

	gaugeColor := cfg.Properties.GaugeColor
	if gaugeColor == 0 {
		gaugeColor = 200
	}

	gaugeNeedleColor := cfg.Properties.GaugeNeedleColor
	if gaugeNeedleColor == 0 {
		gaugeNeedleColor = 255
	}

	// Clipping detection
	clippingThreshold := cfg.Properties.ClippingThreshold
	if clippingThreshold == 0 {
		clippingThreshold = 0.99 // Default: 99% of max
	}

	// Silence detection
	silenceThreshold := cfg.Properties.SilenceThreshold
	if silenceThreshold == 0 {
		silenceThreshold = 0.01 // Default: 1% of max
	}

	// Decay rate (normalized units per second)
	decayRate := cfg.Properties.DecayRate
	if decayRate == 0 {
		decayRate = 2.0 // Default: decay 2.0 units/sec (0.5 seconds from 1.0 to 0.0)
	}

	// Peak hold time
	peakHoldTime := time.Duration(cfg.Properties.PeakHoldTime * float64(time.Second))
	if peakHoldTime == 0 {
		peakHoldTime = 1 * time.Second
	}

	// Auto-hide on silence timeout
	autoHideSilenceTime := time.Duration(cfg.Properties.AutoHideSilenceTime * float64(time.Second))
	if autoHideSilenceTime == 0 {
		autoHideSilenceTime = 2 * time.Second
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

	horizontalAlign := cfg.Properties.HorizontalAlign
	if horizontalAlign == "" {
		horizontalAlign = "center"
	}

	verticalAlign := cfg.Properties.VerticalAlign
	if verticalAlign == "" {
		verticalAlign = "center"
	}

	w := &VolumeMeterWidget{
		BaseWidget:          base,
		displayMode:         displayMode,
		fillColor:           uint8(fillColor),
		clippingColor:       uint8(clippingColor),
		leftChannelColor:    uint8(leftChannelColor),
		rightChannelColor:   uint8(rightChannelColor),
		barBorder:           cfg.Properties.BarBorder,
		gaugeColor:          uint8(gaugeColor),
		gaugeNeedleColor:    uint8(gaugeNeedleColor),
		horizontalAlign:     horizontalAlign,
		verticalAlign:       verticalAlign,
		stereoMode:          cfg.Properties.StereoMode,
		useDBScale:          cfg.Properties.UseDBScale,
		showClipping:        cfg.Properties.ShowClipping,
		clippingThreshold:   clippingThreshold,
		silenceThreshold:    silenceThreshold,
		decayRate:           decayRate,
		showPeakHold:        cfg.Properties.ShowPeakHold,
		peakHoldTime:        peakHoldTime,
		autoHideOnSilence:   cfg.Properties.AutoHideOnSilence,
		autoHideSilenceTime: autoHideSilenceTime,
		lastSuccessTime:     time.Now(),
		lastUpdateTime:      time.Now(),
		face:                fontFace,
		stopChan:            make(chan struct{}),
	}

	log.Printf("[METER] Widget initialized: id=%s, mode=%s, dB=%v, clipping=%v, autoHide=%v",
		cfg.ID, displayMode, w.useDBScale, w.showClipping, base.IsAutoHideEnabled())

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
			log.Printf("[METER] PANIC in polling goroutine: %v\nStack: %s", r, debug.Stack())
		}
	}()

	log.Printf("[METER] Background polling goroutine started")

	// Initialize platform-specific meter reader
	reader, err := newMeterReader()
	if err != nil {
		log.Printf("[METER] FATAL: Failed to initialize meter reader: %v", err)
		log.Printf("[METER] Volume meter widget will not function")
		return
	}
	w.reader = reader

	defer func() {
		if w.reader != nil {
			w.reader.Close()
			w.reader = nil
		}
	}()

	log.Printf("[METER] Meter reader initialized successfully")

	pollInterval := 100 * time.Millisecond // 10Hz default
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			log.Printf("[METER] Polling goroutine stopping")
			return
		case <-ticker.C:
			w.updateMeter()
		}
	}
}

// updateMeter reads meter data and updates widget state
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
		w.failedCalls++
		w.consecutiveErrors++
		log.Printf("[METER] Read error (consecutive: %d): %v", w.consecutiveErrors, err)
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

	// Apply decay to display peak
	if w.peak > w.displayPeak {
		// New peak - update immediately
		w.displayPeak = w.peak
	} else {
		// Decay
		decay := w.decayRate * timeDelta
		w.displayPeak -= decay
		if w.displayPeak < w.peak {
			w.displayPeak = w.peak
		}
		if w.displayPeak < 0 {
			w.displayPeak = 0
		}
	}

	// Peak hold
	if w.showPeakHold {
		if w.peak > w.peakHoldValue {
			w.peakHoldValue = w.peak
			w.peakHoldUntil = now.Add(w.peakHoldTime)
		} else if now.After(w.peakHoldUntil) {
			w.peakHoldValue = 0
		}
	}

	// Auto-hide on silence
	if w.autoHideOnSilence && w.hasAudio {
		w.TriggerAutoHide()
	}
}

// Render renders the volume meter widget
func (w *VolumeMeterWidget) Render() (image.Image, error) {
	// Check auto-hide
	if w.ShouldHide() {
		return nil, nil
	}

	w.mu.RLock()
	peak := w.displayPeak
	channelPeaks := make([]float64, len(w.channelPeaks))
	copy(channelPeaks, w.channelPeaks)
	isClipping := w.isClipping
	peakHold := w.peakHoldValue
	w.mu.RUnlock()

	// Convert to dB if needed
	if w.useDBScale {
		peak = w.linearToDBNormalized(peak)
		peakHold = w.linearToDBNormalized(peakHold)
		for i := range channelPeaks {
			channelPeaks[i] = w.linearToDBNormalized(channelPeaks[i])
		}
	}

	pos := w.GetPosition()
	img := image.NewGray(image.Rect(0, 0, pos.W, pos.H))

	// Check if we should render in stereo mode
	if w.stereoMode && len(channelPeaks) >= 2 {
		// Render stereo version of each mode
		switch w.displayMode {
		case "text":
			w.renderTextStereo(img, channelPeaks, isClipping)
		case "bar_horizontal":
			w.renderBarHorizontalStereo(img, channelPeaks, peakHold, isClipping)
		case "bar_vertical":
			w.renderBarVerticalStereo(img, channelPeaks, peakHold, isClipping)
		case "gauge":
			w.renderGaugeStereo(img, channelPeaks, isClipping)
		case "vu_meter":
			w.renderVUMeterStereo(img, channelPeaks, peakHold, isClipping)
		}
	} else {
		// Render mono version
		switch w.displayMode {
		case "text":
			w.renderText(img, peak, isClipping)
		case "bar_horizontal":
			w.renderBarHorizontal(img, peak, peakHold, isClipping)
		case "bar_vertical":
			w.renderBarVertical(img, peak, peakHold, isClipping)
		case "gauge":
			w.renderGauge(img, peak, isClipping)
		case "vu_meter":
			w.renderVUMeter(img, peak, peakHold, isClipping)
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

	bitmap.DrawAlignedText(img, text, w.face, w.horizontalAlign, w.verticalAlign, 0)
}

// renderBarHorizontal renders horizontal bar display
func (w *VolumeMeterWidget) renderBarHorizontal(img *image.Gray, peak, peakHold float64, isClipping bool) {
	pos := w.GetPosition()
	barWidth := int(float64(pos.W) * peak)

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
				img.SetGray(peakX, y, color.Gray{Y: 255})
			}
		}
	}

	// Draw border
	if w.barBorder {
		bitmap.DrawBorder(img, 128)
	}
}

// renderBarVertical renders vertical bar display
func (w *VolumeMeterWidget) renderBarVertical(img *image.Gray, peak, peakHold float64, isClipping bool) {
	pos := w.GetPosition()
	barHeight := int(float64(pos.H) * peak)
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
				img.SetGray(x, peakY, color.Gray{Y: 255})
			}
		}
	}

	// Draw border
	if w.barBorder {
		bitmap.DrawBorder(img, 128)
	}
}

// renderGauge renders gauge display
func (w *VolumeMeterWidget) renderGauge(img *image.Gray, peak float64, isClipping bool) {
	pos := w.GetPosition()

	needleColor := w.gaugeNeedleColor
	if isClipping && w.showClipping {
		needleColor = w.clippingColor
	}

	// DrawGauge expects percentage as 0-100, not 0.0-1.0
	percentage := peak * 100.0
	bitmap.DrawGauge(img, pos, percentage, w.gaugeColor, needleColor)
}

// renderVUMeter renders VU meter style display
func (w *VolumeMeterWidget) renderVUMeter(img *image.Gray, peak, peakHold float64, isClipping bool) {
	// VU meter is like horizontal bar but with color zones
	pos := w.GetPosition()

	// Draw colored zones: green (0-70%), yellow (70-90%), red (90-100%)
	greenWidth := int(float64(pos.W) * 0.7)
	yellowWidth := int(float64(pos.W) * 0.9)

	barWidth := int(float64(pos.W) * peak)

	for y := 0; y < pos.H; y++ {
		for x := 0; x < barWidth; x++ {
			var fillColor uint8
			if x < greenWidth {
				fillColor = 128 // Green zone
			} else if x < yellowWidth {
				fillColor = 192 // Yellow zone
			} else {
				fillColor = 255 // Red zone
			}

			if isClipping && w.showClipping {
				fillColor = w.clippingColor
			}

			img.SetGray(x, y, color.Gray{Y: fillColor})
		}
	}

	// Draw peak hold line
	if w.showPeakHold && peakHold > 0 {
		peakX := int(float64(pos.W) * peakHold)
		if peakX < pos.W {
			for y := 0; y < pos.H; y++ {
				img.SetGray(peakX, y, color.Gray{Y: 255})
			}
		}
	}

	// Draw zone markers (vertical lines)
	for y := 0; y < pos.H; y++ {
		if greenWidth < pos.W {
			img.SetGray(greenWidth, y, color.Gray{Y: 64})
		}
		if yellowWidth < pos.W {
			img.SetGray(yellowWidth, y, color.Gray{Y: 64})
		}
	}
}

// renderBarHorizontalStereo renders horizontal bars in stereo mode (left/right channels)
func (w *VolumeMeterWidget) renderBarHorizontalStereo(img *image.Gray, channelPeaks []float64, peakHold float64, isClipping bool) {
	pos := w.GetPosition()

	if len(channelPeaks) < 2 {
		// Not stereo, fall back to mono display
		peak := 0.0
		if len(channelPeaks) > 0 {
			peak = channelPeaks[0]
		}
		w.renderBarHorizontal(img, peak, 0, isClipping)
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

	// Draw separator
	for x := 0; x < pos.W; x++ {
		img.SetGray(x, halfHeight, color.Gray{Y: 64})
	}
}

// renderVUMeterStereo renders VU meters in stereo mode (left/right channels)
func (w *VolumeMeterWidget) renderVUMeterStereo(img *image.Gray, channelPeaks []float64, peakHold float64, isClipping bool) {
	pos := w.GetPosition()

	if len(channelPeaks) < 2 {
		// Not stereo, fall back to mono VU
		peak := 0.0
		if len(channelPeaks) > 0 {
			peak = channelPeaks[0]
		}
		w.renderVUMeter(img, peak, peakHold, isClipping)
		return
	}

	// Draw two VU meters: top half = left, bottom half = right
	halfHeight := pos.H / 2

	// Helper function to draw a single VU bar
	drawVU := func(startY, height int, peak float64, channelColor uint8) {
		greenWidth := int(float64(pos.W) * 0.7)
		yellowWidth := int(float64(pos.W) * 0.9)
		barWidth := int(float64(pos.W) * peak)

		for y := startY; y < startY+height; y++ {
			for x := 0; x < barWidth; x++ {
				var fillColor uint8
				if x < greenWidth {
					fillColor = channelColor - 64 // Darker for green
				} else if x < yellowWidth {
					fillColor = channelColor - 32 // Medium for yellow
				} else {
					fillColor = channelColor // Full brightness for red
				}

				img.SetGray(x, y, color.Gray{Y: fillColor})
			}
		}
	}

	// Left channel (top)
	drawVU(0, halfHeight, channelPeaks[0], w.leftChannelColor)

	// Right channel (bottom)
	drawVU(halfHeight, halfHeight, channelPeaks[1], w.rightChannelColor)

	// Draw separator
	for x := 0; x < pos.W; x++ {
		img.SetGray(x, halfHeight, color.Gray{Y: 64})
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

	bitmap.DrawAlignedText(img, text, w.face, w.horizontalAlign, w.verticalAlign, 0)
}

// renderBarVerticalStereo renders vertical bars in stereo mode (left/right channels)
func (w *VolumeMeterWidget) renderBarVerticalStereo(img *image.Gray, channelPeaks []float64, peakHold float64, isClipping bool) {
	pos := w.GetPosition()

	if len(channelPeaks) < 2 {
		// Not stereo, fall back to mono display
		peak := 0.0
		if len(channelPeaks) > 0 {
			peak = channelPeaks[0]
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

	// Draw separator
	for y := 0; y < pos.H; y++ {
		img.SetGray(halfWidth, y, color.Gray{Y: 64})
	}

	// Draw borders if enabled
	if w.barBorder {
		// Left bar border
		for x := 0; x < halfWidth; x++ {
			img.SetGray(x, 0, color.Gray{Y: 128})
			img.SetGray(x, pos.H-1, color.Gray{Y: 128})
		}
		for y := 0; y < pos.H; y++ {
			img.SetGray(0, y, color.Gray{Y: 128})
			img.SetGray(halfWidth-1, y, color.Gray{Y: 128})
		}

		// Right bar border
		for x := halfWidth; x < pos.W; x++ {
			img.SetGray(x, 0, color.Gray{Y: 128})
			img.SetGray(x, pos.H-1, color.Gray{Y: 128})
		}
		for y := 0; y < pos.H; y++ {
			img.SetGray(halfWidth+1, y, color.Gray{Y: 128})
			img.SetGray(pos.W-1, y, color.Gray{Y: 128})
		}
	}
}

// renderGaugeStereo renders gauges in stereo mode (left/right channels)
func (w *VolumeMeterWidget) renderGaugeStereo(img *image.Gray, channelPeaks []float64, isClipping bool) {
	pos := w.GetPosition()

	if len(channelPeaks) < 2 {
		// Not stereo, fall back to mono display
		peak := 0.0
		if len(channelPeaks) > 0 {
			peak = channelPeaks[0]
		}
		w.renderGauge(img, peak, isClipping)
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
	bitmap.DrawGauge(leftImg, leftGaugePos, leftPercentage, w.gaugeColor, leftNeedleColor)

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
	bitmap.DrawGauge(rightImg, rightGaugePos, rightPercentage, w.gaugeColor, rightNeedleColor)

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
