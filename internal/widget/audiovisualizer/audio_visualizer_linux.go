//go:build linux

package audiovisualizer

import (
	"image"
	"image/color"
	"log"
	"math"
	"math/cmplx"
	"sync"
	"time"

	"github.com/mjibson/go-dsp/fft"
	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
)

func init() {
	widget.Register("audio_visualizer", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// frequencyCompensationCurve defines frequency-dependent gain to balance spectrum display
var frequencyCompensationCurve = []struct {
	maxFreq float64
	gain    float64
}{
	{100, 0.6},
	{250, 0.85},
	{500, 1.0},
	{1000, 1.5},
	{2000, 2.0},
	{4000, 3.0},
	{8000, 4.0},
	{99999, 5.0},
}

// AudioCaptureWCA wraps AudioCaptureLinux for API compatibility
type AudioCaptureWCA struct {
	capture *AudioCaptureLinux
}

// GetSharedAudioCapture returns the shared audio capture instance
func GetSharedAudioCapture() (*AudioCaptureWCA, error) {
	capture, err := GetSharedAudioCaptureLinux()
	if err != nil {
		return nil, err
	}
	return &AudioCaptureWCA{capture: capture}, nil
}

// ReinitializeSharedAudioCapture reinitializes the shared audio capture
func ReinitializeSharedAudioCapture() error {
	return ReinitializeSharedAudioCaptureLinux()
}

// ReadSamples returns current audio samples
func (ac *AudioCaptureWCA) ReadSamples() ([]float32, []float32, error) {
	if ac.capture == nil {
		return nil, nil, nil
	}
	return ac.capture.ReadSamples()
}

// Close stops audio capture
func (ac *AudioCaptureWCA) Close() {
	if ac.capture != nil {
		ac.capture.Close()
	}
}

// Widget displays real-time spectrum analyzer or oscilloscope
type Widget struct {
	*widget.BaseWidget
	audioCapture *AudioCaptureLinux
	mu           sync.Mutex

	// Display settings
	displayMode string

	// Error state management
	errorWidget    *widget.ErrorWidget // Error widget proxy (nil = normal operation)
	errorCount     int                 // Consecutive errors
	errorThreshold int                 // Errors before entering error state
	startupTime    time.Time           // For startup grace period

	// Spectrum settings
	frequencyScale         string
	frequencyCompensation  bool
	spectrumDynamicScaling float64
	spectrumDynamicWindow  float64
	smoothing              float64
	peakHold               bool
	peakHoldTime           float64
	barStyle               string
	fillColor              uint8
	barCount               int

	// Oscilloscope settings
	sampleCount       int
	channelMode       string
	waveformStyle     string
	leftChannelColor  uint8
	rightChannelColor uint8

	// Audio data buffers
	audioData      []float32
	audioDataLeft  []float32
	audioDataRight []float32

	// Spectrum analysis state
	spectrumData        []float64
	peakValues          []float64
	peakTimestamps      []time.Time
	smoothedValues      []float64
	barEnergyHistory    [][]float64
	barEnergyIndex      int
	barEnergyWindowSize int
	lastUpdateTime      time.Time
}

// New creates a new audio visualizer widget
func New(cfg config.WidgetConfig) (widget.Widget, error) {
	// Initialize audio capture
	audioCapture, err := GetSharedAudioCaptureLinux()

	if err != nil {
		log.Printf("[AUDIO-VIS-LINUX] Audio capture error: %v", err)
	} else if audioCapture != nil && audioCapture.IsRunning() {
		log.Printf("[AUDIO-VIS-LINUX] Real audio capture initialized")
	} else {
		log.Printf("[AUDIO-VIS-LINUX] Audio capture not running")
	}

	// Set default update interval
	if cfg.UpdateInterval <= 0 {
		cfg.UpdateInterval = 0.033
	}

	base := widget.NewBaseWidget(cfg)

	// Display mode
	displayMode := cfg.Mode
	if displayMode == "" {
		displayMode = shared.AudioDisplayModeSpectrum
	}

	// Spectrum settings
	barCount := 32
	frequencyScale := shared.AudioFrequencyScaleLogarithmic
	frequencyCompensation := true
	spectrumDynamicScaling := 0.0
	spectrumDynamicWindow := 0.5
	smoothing := 0.5
	barStyle := shared.AudioBarStyleBars

	if cfg.Spectrum != nil {
		if cfg.Spectrum.Bars > 0 {
			barCount = cfg.Spectrum.Bars
		}
		if cfg.Spectrum.Scale != "" {
			frequencyScale = cfg.Spectrum.Scale
		}
		frequencyCompensation = cfg.Spectrum.FrequencyCompensation
		if cfg.Spectrum.Smoothing > 0 {
			smoothing = cfg.Spectrum.Smoothing
		}
		if cfg.Spectrum.Style != "" {
			barStyle = cfg.Spectrum.Style
		}
		if cfg.Spectrum.DynamicScaling != nil {
			if cfg.Spectrum.DynamicScaling.Strength > 0 {
				spectrumDynamicScaling = cfg.Spectrum.DynamicScaling.Strength
			}
			if cfg.Spectrum.DynamicScaling.Window > 0 {
				spectrumDynamicWindow = cfg.Spectrum.DynamicScaling.Window
			}
		}
	}

	if barCount > cfg.Position.W {
		barCount = cfg.Position.W
	}

	// Peak settings
	peakHold := true
	peakHoldTime := 1.0

	if cfg.Spectrum != nil && cfg.Spectrum.Peak != nil {
		peakHold = cfg.Spectrum.Peak.Enabled
		if cfg.Spectrum.Peak.HoldTime > 0 {
			peakHoldTime = cfg.Spectrum.Peak.HoldTime
		}
	}

	// Oscilloscope settings
	sampleCount := 256
	channelMode := shared.AudioChannelModeMono
	waveformStyle := shared.AudioWaveformStyleLine

	if cfg.Oscilloscope != nil {
		if cfg.Oscilloscope.Samples > 0 {
			sampleCount = cfg.Oscilloscope.Samples
		}
		if cfg.Oscilloscope.Style != "" {
			waveformStyle = cfg.Oscilloscope.Style
		}
	}

	if cfg.Channel != "" {
		channelMode = cfg.Channel
	}

	// Colors
	fillColor := 255
	leftChannelColor := 255
	rightChannelColor := 200

	switch displayMode {
	case shared.AudioDisplayModeSpectrum:
		if cfg.Spectrum != nil && cfg.Spectrum.Colors != nil {
			if cfg.Spectrum.Colors.Fill != nil {
				fillColor = *cfg.Spectrum.Colors.Fill
			}
			if cfg.Spectrum.Colors.Left != nil {
				leftChannelColor = *cfg.Spectrum.Colors.Left
			}
			if cfg.Spectrum.Colors.Right != nil {
				rightChannelColor = *cfg.Spectrum.Colors.Right
			}
		}
	case shared.AudioDisplayModeOscilloscope:
		if cfg.Oscilloscope != nil && cfg.Oscilloscope.Colors != nil {
			if cfg.Oscilloscope.Colors.Fill != nil {
				fillColor = *cfg.Oscilloscope.Colors.Fill
			}
			if cfg.Oscilloscope.Colors.Left != nil {
				leftChannelColor = *cfg.Oscilloscope.Colors.Left
			}
			if cfg.Oscilloscope.Colors.Right != nil {
				rightChannelColor = *cfg.Oscilloscope.Colors.Right
			}
		}
	}

	windowSize := int(spectrumDynamicWindow/cfg.UpdateInterval) + 1
	if windowSize < 2 {
		windowSize = 2
	}

	energyHistory := make([][]float64, barCount)
	for i := range energyHistory {
		energyHistory[i] = make([]float64, windowSize)
	}

	w := &Widget{
		BaseWidget:             base,
		audioCapture:           audioCapture,
		displayMode:            displayMode,
		frequencyScale:         frequencyScale,
		frequencyCompensation:  frequencyCompensation,
		spectrumDynamicScaling: spectrumDynamicScaling,
		spectrumDynamicWindow:  spectrumDynamicWindow,
		smoothing:              smoothing,
		peakHold:               peakHold,
		peakHoldTime:           peakHoldTime,
		barStyle:               barStyle,
		fillColor:              uint8(fillColor),
		barCount:               barCount,
		sampleCount:            sampleCount,
		channelMode:            channelMode,
		waveformStyle:          waveformStyle,
		leftChannelColor:       uint8(leftChannelColor),
		rightChannelColor:      uint8(rightChannelColor),
		spectrumData:           make([]float64, barCount),
		peakValues:             make([]float64, barCount),
		peakTimestamps:         make([]time.Time, barCount),
		smoothedValues:         make([]float64, barCount),
		barEnergyHistory:       energyHistory,
		barEnergyIndex:         0,
		barEnergyWindowSize:    windowSize,
		audioData:              make([]float32, 0, 4096),
		audioDataLeft:          make([]float32, 0, 4096),
		audioDataRight:         make([]float32, 0, 4096),
		errorThreshold:         30, // ~1 second at 30fps
		startupTime:            time.Now(),
	}

	// Enter error state immediately if audio capture failed to initialize
	if audioCapture == nil || !audioCapture.IsRunning() {
		// Only show error after startup grace period (audio tools may take time to start)
		// For immediate failures, we'll detect in Update()
		if err != nil {
			pos := w.GetPosition()
			w.errorWidget = widget.NewErrorWidget(pos.W, pos.H, "NO AUDIO")
			log.Printf("[AUDIO-VIS-LINUX] Entering error state: audio capture unavailable")
		}
	}

	return w, nil
}

// Update reads audio data and updates visualization
func (w *Widget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Delegate to error widget if in error state
	if w.errorWidget != nil {
		return w.errorWidget.Update()
	}

	w.lastUpdateTime = time.Now()

	// Check audio capture health
	if w.audioCapture == nil || !w.audioCapture.IsRunning() {
		// Grace period: don't count errors during startup (audio tools may take time)
		if time.Since(w.startupTime) > 3*time.Second {
			w.errorCount++
			if w.errorCount >= w.errorThreshold {
				pos := w.GetPosition()
				w.errorWidget = widget.NewErrorWidget(pos.W, pos.H, "NO AUDIO")
				log.Printf("[AUDIO-VIS-LINUX] Entering error state after %d consecutive failures", w.errorCount)
			}
		}
		return nil
	}

	// Reset error count on successful capture access
	w.errorCount = 0

	left, right := w.audioCapture.GetRecentSamples(4096)
	if len(left) == 0 {
		return nil
	}

	// Combine stereo to mono for spectrum analysis
	samples := make([]float32, len(left))
	for i := range left {
		samples[i] = (left[i] + right[i]) / 2.0
	}

	// Update audio buffers
	maxSamples := 8192
	w.audioData = append(w.audioData, samples...)
	if len(w.audioData) > maxSamples {
		w.audioData = w.audioData[len(w.audioData)-maxSamples:]
	}

	w.audioDataLeft = append(w.audioDataLeft, left...)
	w.audioDataRight = append(w.audioDataRight, right...)
	if len(w.audioDataLeft) > maxSamples {
		w.audioDataLeft = w.audioDataLeft[len(w.audioDataLeft)-maxSamples:]
		w.audioDataRight = w.audioDataRight[len(w.audioDataRight)-maxSamples:]
	}

	// Process for spectrum mode
	if w.displayMode == shared.AudioDisplayModeSpectrum && len(w.audioData) >= 2048 {
		w.updateSpectrum(w.audioData)
	}

	return nil
}

// updateSpectrum performs FFT and updates spectrum data
func (w *Widget) updateSpectrum(samples []float32) {
	fftSize := 2048
	if len(samples) < fftSize {
		return
	}

	fftSamples := samples[len(samples)-fftSize:]

	// Remove DC offset
	var mean float32
	for _, s := range fftSamples {
		mean += s
	}
	mean /= float32(len(fftSamples))

	// Convert to complex and apply window
	input := make([]complex128, fftSize)
	for i := 0; i < fftSize; i++ {
		sample := float64(fftSamples[i] - mean)
		window := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(fftSize-1)))
		input[i] = complex(sample*window, 0)
	}

	// Perform FFT
	output := fft.FFT(input)

	// Calculate magnitudes
	halfSize := fftSize / 2
	magnitudes := make([]float64, halfSize)

	for i := 0; i < halfSize; i++ {
		magnitudes[i] = cmplx.Abs(output[i]) / float64(fftSize)
	}
	magnitudes[0] = 0.0 // Zero DC

	// Normalize
	maxRawMag := 0.0
	for i := 1; i < halfSize; i++ {
		if magnitudes[i] > maxRawMag {
			maxRawMag = magnitudes[i]
		}
	}

	if maxRawMag > 0.0001 {
		targetMax := 0.75
		normFactor := targetMax / maxRawMag
		for i := 0; i < halfSize; i++ {
			magnitudes[i] = math.Min(1.0, magnitudes[i]*normFactor)
		}
	}

	// Map to bars
	barCount := len(w.spectrumData)
	if w.frequencyScale == shared.AudioFrequencyScaleLogarithmic {
		w.mapFrequenciesLogarithmic(magnitudes, barCount)
	} else {
		w.mapFrequenciesLinear(magnitudes, barCount)
	}

	// Store energy history
	for i := 0; i < barCount; i++ {
		w.barEnergyHistory[i][w.barEnergyIndex] = w.spectrumData[i]
	}
	w.barEnergyIndex = (w.barEnergyIndex + 1) % w.barEnergyWindowSize

	// Apply smoothing
	dt := time.Since(w.lastUpdateTime).Seconds()
	if dt > 0 && w.smoothing > 0 {
		for i := range w.smoothedValues {
			alpha := 1.0 - math.Pow(w.smoothing, dt*30)
			w.smoothedValues[i] = alpha*w.spectrumData[i] + (1-alpha)*w.smoothedValues[i]
		}
	} else {
		copy(w.smoothedValues, w.spectrumData)
	}

	// Update peak hold
	if w.peakHold {
		now := time.Now()
		for i := range w.peakValues {
			if w.smoothedValues[i] > w.peakValues[i] {
				w.peakValues[i] = w.smoothedValues[i]
				w.peakTimestamps[i] = now
			} else {
				elapsed := now.Sub(w.peakTimestamps[i]).Seconds()
				if elapsed > w.peakHoldTime {
					decayRate := 0.3
					dt := time.Since(w.lastUpdateTime).Seconds()
					w.peakValues[i] *= 1.0 - decayRate*dt
					if w.peakValues[i] < w.smoothedValues[i] {
						w.peakValues[i] = w.smoothedValues[i]
					}
					if w.peakValues[i] < 0.01 {
						w.peakValues[i] = 0
					}
				}
			}
		}
	}
}

// mapFrequenciesLogarithmic maps FFT bins to bars using logarithmic scale
func (w *Widget) mapFrequenciesLogarithmic(magnitudes []float64, barCount int) {
	minFreq := 40.0
	maxFreq := 20000.0
	sampleRate := 48000.0
	freqPerBin := sampleRate / float64(len(magnitudes)*2)

	for i := 0; i < barCount; i++ {
		ratio := float64(i) / float64(barCount)
		freqStart := minFreq * math.Pow(maxFreq/minFreq, ratio)
		freqEnd := minFreq * math.Pow(maxFreq/minFreq, (float64(i+1))/float64(barCount))

		binStart := int(freqStart / freqPerBin)
		binEnd := int(freqEnd / freqPerBin)

		if binEnd >= len(magnitudes) {
			binEnd = len(magnitudes) - 1
		}
		if binStart >= len(magnitudes) {
			binStart = len(magnitudes) - 1
		}

		peakValue := 0.0
		for j := binStart; j <= binEnd && j < len(magnitudes); j++ {
			if magnitudes[j] > peakValue {
				peakValue = magnitudes[j]
			}
		}

		if w.frequencyCompensation {
			centerFreq := (freqStart + freqEnd) / 2.0
			for _, curve := range frequencyCompensationCurve {
				if centerFreq < curve.maxFreq {
					peakValue = math.Min(1.0, peakValue*curve.gain)
					break
				}
			}
		}

		w.spectrumData[i] = peakValue
	}
}

// mapFrequenciesLinear maps FFT bins to bars using linear scale
func (w *Widget) mapFrequenciesLinear(magnitudes []float64, barCount int) {
	binsPerBar := len(magnitudes) / barCount
	sampleRate := 48000.0
	freqPerBin := sampleRate / float64(len(magnitudes)*2)

	for i := 0; i < barCount; i++ {
		start := i * binsPerBar
		end := start + binsPerBar
		if end > len(magnitudes) {
			end = len(magnitudes)
		}

		peakValue := 0.0
		for j := start; j < end; j++ {
			if magnitudes[j] > peakValue {
				peakValue = magnitudes[j]
			}
		}

		if w.frequencyCompensation {
			centerFreq := (float64(start) + float64(end)) / 2.0 * freqPerBin
			for _, curve := range frequencyCompensationCurve {
				if centerFreq < curve.maxFreq {
					peakValue = math.Min(1.0, peakValue*curve.gain)
					break
				}
			}
		}

		w.spectrumData[i] = peakValue
	}
}

// Render draws the visualization
func (w *Widget) Render() (image.Image, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Delegate to error widget if in error state
	if w.errorWidget != nil {
		return w.errorWidget.Render()
	}

	// Create canvas with background
	img := w.CreateCanvas()

	if w.displayMode == shared.AudioDisplayModeSpectrum {
		w.renderSpectrum(img)
	} else if w.displayMode == shared.AudioDisplayModeOscilloscope {
		w.renderOscilloscope(img)
	}

	w.ApplyBorder(img)

	return img, nil
}

// renderSpectrum draws spectrum analyzer bars
func (w *Widget) renderSpectrum(img *image.Gray) {
	barCount := len(w.spectrumData)
	if barCount == 0 {
		return
	}

	pos := w.GetPosition()
	width := pos.W
	height := pos.H
	barWidth := width / barCount
	gap := 0
	if w.barStyle == shared.AudioBarStyleBars && barWidth > 2 {
		gap = 1
	}

	for i := 0; i < barCount; i++ {
		magnitude := w.smoothedValues[i]
		if magnitude > 1.0 {
			magnitude = 1.0
		}
		if magnitude < 0.0 {
			magnitude = 0.0
		}

		barHeight := int(magnitude * float64(height))
		if barHeight > height {
			barHeight = height
		}

		x := i * barWidth
		y := height - barHeight

		if w.barStyle == shared.AudioBarStyleBars {
			for py := y; py < height; py++ {
				for px := x; px < x+barWidth-gap && px < width; px++ {
					img.SetGray(px, py, color.Gray{Y: w.fillColor})
				}
			}
		} else {
			for px := x; px < x+barWidth && px < width; px++ {
				if y >= 0 && y < height {
					img.SetGray(px, y, color.Gray{Y: w.fillColor})
				}
			}
		}

		// Peak hold
		if w.peakHold && i < len(w.peakValues) {
			peakMagnitude := w.peakValues[i]
			if peakMagnitude > 1.0 {
				peakMagnitude = 1.0
			}
			if peakMagnitude < 0.0 {
				peakMagnitude = 0.0
			}
			peakY := height - int(peakMagnitude*float64(height))
			if peakY >= 0 && peakY < height {
				for px := x; px < x+barWidth-gap && px < width; px++ {
					img.SetGray(px, peakY, color.Gray{Y: w.fillColor})
				}
			}
		}
	}
}

// renderOscilloscope draws waveform
func (w *Widget) renderOscilloscope(img *image.Gray) {
	pos := w.GetPosition()
	height := pos.H
	width := pos.W
	sampleCount := w.sampleCount

	if len(w.audioData) == 0 {
		return
	}
	if sampleCount > len(w.audioData) {
		sampleCount = len(w.audioData)
	}

	centerY := height / 2
	samples := w.audioData[len(w.audioData)-sampleCount:]

	amplitude := float32(height) / 2.0

	for i := 0; i < sampleCount-1; i++ {
		x1 := i * width / sampleCount
		x2 := (i + 1) * width / sampleCount

		y1 := centerY - int(samples[i]*amplitude)
		y2 := centerY - int(samples[i+1]*amplitude)

		if y1 < 0 {
			y1 = 0
		}
		if y1 >= height {
			y1 = height - 1
		}
		if y2 < 0 {
			y2 = 0
		}
		if y2 >= height {
			y2 = height - 1
		}

		if w.waveformStyle == shared.AudioWaveformStyleLine {
			bitmap.DrawLine(img, x1, y1, x2, y2, color.Gray{Y: w.fillColor})
		} else if w.waveformStyle == shared.AudioWaveformStyleFilled {
			if y1 < centerY {
				for y := y1; y <= centerY && y < height; y++ {
					if x1 >= 0 && x1 < width {
						img.SetGray(x1, y, color.Gray{Y: w.fillColor})
					}
				}
			} else {
				for y := centerY; y <= y1 && y < height; y++ {
					if x1 >= 0 && x1 < width {
						img.SetGray(x1, y, color.Gray{Y: w.fillColor})
					}
				}
			}
		}
	}
}
