//go:build windows

package widget

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/cmplx"
	"sync"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/mjibson/go-dsp/fft"
	"github.com/moutend/go-wca/pkg/wca"
	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
)

func init() {
	Register("audio_visualizer", func(cfg config.WidgetConfig) (Widget, error) {
		return NewAudioVisualizerWidget(cfg)
	})
}

// frequencyCompensationCurve defines frequency-dependent gain to balance spectrum display
// Each entry: max frequency (Hz) and gain multiplier
var frequencyCompensationCurve = []struct {
	maxFreq float64
	gain    float64
}{
	{100, 0.6},   // Sub-bass: attenuate
	{250, 0.85},  // Bass: slight attenuation
	{500, 1.0},   // Low-mids: neutral
	{1000, 1.5},  // Mids: moderate boost
	{2000, 2.0},  // Upper mids: moderate boost
	{4000, 3.0},  // Highs: strong boost
	{8000, 4.0},  // Upper highs: strong boost
	{99999, 5.0}, // Very high frequencies: maximum boost
}

// AudioVisualizerWidget displays real-time spectrum analyzer or oscilloscope
type AudioVisualizerWidget struct {
	*BaseWidget
	audioCapture *AudioCaptureWCA
	volumeReader *VolumeReaderWCA
	mu           sync.Mutex

	// Display settings
	displayMode string

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
	audioData      []float32 // Latest audio samples (mixed for spectrum)
	audioDataLeft  []float32 // Left channel samples (for oscilloscope)
	audioDataRight []float32 // Right channel samples (for oscilloscope)

	// Spectrum analysis state
	spectrumData        []float64 // Spectrum magnitudes
	peakValues          []float64 // Peak hold values
	peakTimestamps      []time.Time
	smoothedValues      []float64   // Smoothed spectrum values
	barEnergyHistory    [][]float64 // Rolling window of bar energies for dynamic scaling
	barEnergyIndex      int         // Circular buffer index for energy history
	barEnergyWindowSize int         // Number of history samples for rolling average
	lastUpdateTime      time.Time

	// Error state tracking
	errorWidget    *ErrorWidget // Error widget proxy (nil = normal operation)
	errorCount     int          // Consecutive error count
	errorThreshold int          // Threshold before entering error state
	startupTime    time.Time    // Widget creation time for grace period

	// Device change notification
	deviceNotifyChan <-chan struct{} // Receives signal on audio device change
}

// NewAudioVisualizerWidget creates a new audio visualizer widget
func NewAudioVisualizerWidget(cfg config.WidgetConfig) (Widget, error) {
	// Get shared audio capture instance
	capture, err := GetSharedAudioCapture()
	if err != nil {
		return nil, fmt.Errorf("failed to get shared audio capture: %w", err)
	}

	// Get shared volume reader for volume compensation
	volumeReader, err := GetSharedVolumeReader()
	if err != nil {
		return nil, fmt.Errorf("failed to get shared volume reader: %w", err)
	}

	// Subscribe to device change notifications
	var deviceNotifyChan <-chan struct{}
	if notifier, err := GetDeviceNotifier(); err == nil {
		deviceNotifyChan = notifier.Subscribe()
	}

	// Set default update interval for audio visualizer (33ms = ~30fps)
	if cfg.UpdateInterval <= 0 {
		cfg.UpdateInterval = 0.033
	}

	// Create base widget
	base := NewBaseWidget(cfg)

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

	// Validate and clamp bar count to widget width
	if barCount > cfg.Position.W {
		log.Printf("audio_visualizer: bars count (%d) exceeds widget width (%d), clamping to %d", barCount, cfg.Position.W, cfg.Position.W)
		barCount = cfg.Position.W
	}

	// Peak settings (spectrum mode only)
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

	// Channel mode from top-level config
	if cfg.Channel != "" {
		channelMode = cfg.Channel
	}

	// Error threshold from config (default: 30 = ~3 seconds at 33ms update interval)
	errorThreshold := 30
	if cfg.ErrorThreshold > 0 {
		errorThreshold = cfg.ErrorThreshold
	}

	// Colors from mode-specific configs
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

	// Calculate energy history window size for dynamic scaling
	windowSize := int(spectrumDynamicWindow/cfg.UpdateInterval) + 1
	if windowSize < 2 {
		windowSize = 2 // Minimum 2 samples
	}

	// Initialize energy history (2D array: [barCount][windowSize])
	energyHistory := make([][]float64, barCount)
	for i := range energyHistory {
		energyHistory[i] = make([]float64, windowSize)
	}

	w := &AudioVisualizerWidget{
		BaseWidget:             base,
		audioCapture:           capture,
		volumeReader:           volumeReader,
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
		errorThreshold:         errorThreshold,
		startupTime:            time.Now(),
		deviceNotifyChan:       deviceNotifyChan,
	}

	return w, nil
}

// Update captures audio and processes it
func (w *AudioVisualizerWidget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check for device change notification (non-blocking)
	if w.deviceNotifyChan != nil {
		select {
		case <-w.deviceNotifyChan:
			// Device changed - reinitialize audio capture
			log.Printf("[AUDIO-VIS] Device change detected, reinitializing...")
			w.audioCapture.cleanup()
			w.audioCapture.initialized = false
			newCapture, err := GetSharedAudioCapture()
			if err != nil {
				log.Printf("[AUDIO-VIS] Failed to reinitialize after device change: %v", err)
				// Don't enter error state immediately - will retry on next update
				return nil
			}
			w.audioCapture = newCapture
			// Reset error state if we were in error
			w.errorWidget = nil
			w.errorCount = 0
			w.startupTime = time.Now() // Reset startup grace period
			log.Printf("[AUDIO-VIS] Reinitialized after device change")
		default:
			// No notification, continue normally
		}
	}

	// If in error state, delegate to error widget
	if w.errorWidget != nil {
		return w.errorWidget.Update()
	}

	// Capture audio samples (both channels)
	leftSamples, rightSamples, err := w.audioCapture.ReadSamples()
	if err != nil {
		// Grace period: ignore errors during first 3 seconds after startup
		// Audio buffer needs time to warm up
		if time.Since(w.startupTime) < 3*time.Second {
			return nil
		}

		w.errorCount++
		if w.errorCount >= w.errorThreshold {
			// Enter error state - create error widget proxy
			pos := w.GetPosition()
			w.errorWidget = NewErrorWidget(pos.W, pos.H, "AUDIO ERROR")
			// Mark singleton as uninitialized so config reload can retry
			w.audioCapture.initialized = false
			log.Printf("[AUDIO-VIS] Audio capture failed after %d consecutive errors: %v", w.errorCount, err)
		}
		return nil
	}

	// Reset error count on successful read
	w.errorCount = 0

	// If no samples, create silent buffers to allow peaks to decay
	if len(leftSamples) == 0 {
		leftSamples = make([]float32, 1024)  // Silent buffer (all zeros)
		rightSamples = make([]float32, 1024) // Silent buffer (all zeros)
	}

	// Apply volume compensation to both channels
	// WASAPI loopback captures audio AFTER system volume is applied,
	// so we need to compensate to show visualization independent of volume level
	volumePercent, _, err := w.volumeReader.GetVolume()
	if err == nil && volumePercent > 1.0 { // Avoid division by very small numbers (1% minimum)
		// Multiply by (100.0 / volumePercent) to restore original signal
		// Example: at 30% volume, multiply by 100/30 = 3.33x
		gainFactor := float32(100.0 / volumePercent)

		// Apply gain to left channel
		for i := range leftSamples {
			leftSamples[i] *= gainFactor
			// Clamp to -1.0 to +1.0 range
			if leftSamples[i] > 1.0 {
				leftSamples[i] = 1.0
			}
			if leftSamples[i] < -1.0 {
				leftSamples[i] = -1.0
			}
		}

		// Apply gain to right channel
		for i := range rightSamples {
			rightSamples[i] *= gainFactor
			// Clamp to -1.0 to +1.0 range
			if rightSamples[i] > 1.0 {
				rightSamples[i] = 1.0
			}
			if rightSamples[i] < -1.0 {
				rightSamples[i] = -1.0
			}
		}
	}

	// Store left and right channels separately for oscilloscope mode
	maxSamples := 8192
	w.audioDataLeft = append(w.audioDataLeft, leftSamples...)
	if len(w.audioDataLeft) > maxSamples {
		w.audioDataLeft = w.audioDataLeft[len(w.audioDataLeft)-maxSamples:]
	}
	w.audioDataRight = append(w.audioDataRight, rightSamples...)
	if len(w.audioDataRight) > maxSamples {
		w.audioDataRight = w.audioDataRight[len(w.audioDataRight)-maxSamples:]
	}

	// Mix both channels for spectrum analysis (average of left and right)
	mixedSamples := make([]float32, len(leftSamples))
	for i := range leftSamples {
		mixedSamples[i] = (leftSamples[i] + rightSamples[i]) / 2.0
	}
	w.audioData = append(w.audioData, mixedSamples...)
	if len(w.audioData) > maxSamples {
		w.audioData = w.audioData[len(w.audioData)-maxSamples:]
	}

	// Process for spectrum mode (always update, even with silence to decay peaks)
	// Use accumulated audioData buffer, not just current samples
	if w.displayMode == shared.AudioDisplayModeSpectrum {
		w.updateSpectrum(w.audioData)
	}

	w.lastUpdateTime = time.Now()
	return nil
}

// updateSpectrum performs FFT and updates spectrum data
func (w *AudioVisualizerWidget) updateSpectrum(samples []float32) {
	// Need power of 2 samples for FFT
	// Use 8192 for better low-frequency resolution
	fftSize := 8192
	if len(samples) < fftSize {
		return
	}

	// Take last fftSize samples
	fftSamples := samples[len(samples)-fftSize:]

	// Remove DC offset (mean) to avoid DC component in FFT
	var mean float32
	for _, s := range fftSamples {
		mean += s
	}
	mean /= float32(len(fftSamples))

	// Convert to complex and apply window
	input := make([]complex128, fftSize)
	for i := 0; i < fftSize; i++ {
		// Remove DC offset and apply Hann window
		sample := float64(fftSamples[i] - mean)
		window := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(fftSize-1)))
		input[i] = complex(sample*window, 0)
	}

	// Perform FFT
	output := fft.FFT(input)

	// Calculate magnitudes for first half (positive frequencies)
	halfSize := fftSize / 2
	magnitudes := make([]float64, halfSize)

	// Calculate raw magnitudes
	for i := 0; i < halfSize; i++ {
		magnitudes[i] = cmplx.Abs(output[i]) / float64(fftSize)
	}

	// Zero out DC component (bin 0) to avoid lighting up all low frequency bars
	magnitudes[0] = 0.0

	// Find the maximum magnitude for normalization
	maxRawMag := 0.0
	for i := 1; i < halfSize; i++ {
		if magnitudes[i] > maxRawMag {
			maxRawMag = magnitudes[i]
		}
	}

	// Normalize to 0-1 range based on actual signal strength
	// Target: strongest frequency should reach 0.7-0.8 (leaving headroom)
	if maxRawMag > 0.0001 {
		targetMax := 0.75
		normFactor := targetMax / maxRawMag
		for i := 0; i < halfSize; i++ {
			magnitudes[i] = math.Min(1.0, magnitudes[i]*normFactor)
		}
	}

	// Map frequencies to bars
	barCount := len(w.spectrumData)

	if w.frequencyScale == shared.AudioFrequencyScaleLogarithmic {
		w.mapFrequenciesLogarithmic(magnitudes, barCount)
	} else {
		w.mapFrequenciesLinear(magnitudes, barCount)
	}

	// Store current spectrum data in energy history for dynamic scaling
	for i := 0; i < barCount; i++ {
		w.barEnergyHistory[i][w.barEnergyIndex] = w.spectrumData[i]
	}
	w.barEnergyIndex = (w.barEnergyIndex + 1) % w.barEnergyWindowSize

	// Apply dynamic scaling if enabled (strength > 0)
	if w.spectrumDynamicScaling > 0 {
		dynamicGains := w.calculateDynamicGain(barCount)
		for i := 0; i < barCount; i++ {
			w.spectrumData[i] *= dynamicGains[i]
			// Clamp to 0.0-1.0 range
			if w.spectrumData[i] > 1.0 {
				w.spectrumData[i] = 1.0
			}
			if w.spectrumData[i] < 0.0 {
				w.spectrumData[i] = 0.0
			}
		}
	}

	// Apply smoothing
	dt := time.Since(w.lastUpdateTime).Seconds()
	if dt > 0 && w.smoothing > 0 {
		for i := range w.smoothedValues {
			// Exponential moving average with time-based decay
			alpha := 1.0 - math.Pow(w.smoothing, dt*30) // Adjust for frame rate
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
				// Decay peak if hold time expired
				elapsed := now.Sub(w.peakTimestamps[i]).Seconds()
				if elapsed > w.peakHoldTime {
					// Faster decay - drop at ~30% per second
					decayRate := 0.3 // 30% decay per second
					dt := time.Since(w.lastUpdateTime).Seconds()
					w.peakValues[i] *= 1.0 - decayRate*dt

					// Clamp to smoothed value (peak should never go below current value)
					if w.peakValues[i] < w.smoothedValues[i] {
						w.peakValues[i] = w.smoothedValues[i]
					}

					// Clear if very small
					if w.peakValues[i] < 0.01 {
						w.peakValues[i] = 0
					}
				}
			}
		}
	}
}

// mapFrequenciesLogarithmic maps FFT bins to bars using logarithmic scale
func (w *AudioVisualizerWidget) mapFrequenciesLogarithmic(magnitudes []float64, barCount int) {
	// Logarithmic frequency mapping (similar to Winamp)
	// Start at 40 Hz to avoid sub-bass issues (narrow bars, spectral leakage)
	minFreq := 40.0    // Hz
	maxFreq := 20000.0 // Hz

	// Get actual sample rate from audio capture
	sampleRate := 48000.0 // Default fallback
	if w.audioCapture != nil {
		w.audioCapture.mu.Lock()
		actualRate := float64(w.audioCapture.sampleRate)
		w.audioCapture.mu.Unlock()
		if actualRate > 0 {
			sampleRate = actualRate
		}
	}
	freqPerBin := sampleRate / float64(len(magnitudes)*2)

	for i := 0; i < barCount; i++ {
		// Calculate frequency range for this bar (logarithmic)
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

		// Find peak magnitude in this frequency range (not average)
		// Spectrum analyzers should show the strongest frequency component, not the average
		peakValue := 0.0
		sum := 0.0
		count := 0
		for j := binStart; j <= binEnd && j < len(magnitudes); j++ {
			if magnitudes[j] > peakValue {
				peakValue = magnitudes[j]
			}
			sum += magnitudes[j]
			count++
		}

		if count > 0 {
			// Apply frequency-dependent gain to compensate for natural bass-heavy energy distribution
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
		} else {
			w.spectrumData[i] = 0
		}
	}
}

// mapFrequenciesLinear maps FFT bins to bars using linear scale
func (w *AudioVisualizerWidget) mapFrequenciesLinear(magnitudes []float64, barCount int) {
	binsPerBar := len(magnitudes) / barCount

	// Get sample rate for frequency calculation
	sampleRate := 48000.0
	if w.audioCapture != nil {
		w.audioCapture.mu.Lock()
		actualRate := float64(w.audioCapture.sampleRate)
		w.audioCapture.mu.Unlock()
		if actualRate > 0 {
			sampleRate = actualRate
		}
	}
	freqPerBin := sampleRate / float64(len(magnitudes)*2)

	for i := 0; i < barCount; i++ {
		start := i * binsPerBar
		end := start + binsPerBar
		if end > len(magnitudes) {
			end = len(magnitudes)
		}

		// Find peak magnitude (not average)
		peakValue := 0.0
		for j := start; j < end; j++ {
			if magnitudes[j] > peakValue {
				peakValue = magnitudes[j]
			}
		}

		// Apply frequency-dependent gain (configurable)
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

// calculateDynamicGain computes per-bar dynamic gain to balance frequency energy
// Returns gain multipliers (1.0-4.0) for each bar based on their rolling average energy
func (w *AudioVisualizerWidget) calculateDynamicGain(barCount int) []float64 {
	gains := make([]float64, barCount)

	// Calculate rolling average energy for each bar
	avgEnergies := make([]float64, barCount)
	for i := 0; i < barCount; i++ {
		sum := 0.0
		count := 0
		for j := 0; j < w.barEnergyWindowSize; j++ {
			sum += w.barEnergyHistory[i][j]
			count++
		}
		if count > 0 {
			avgEnergies[i] = sum / float64(count)
		}
	}

	// Calculate global median/mean energy across all bars
	// Use median to be robust against outliers
	sortedEnergies := make([]float64, barCount)
	copy(sortedEnergies, avgEnergies)
	// Simple bubble sort for median (small array, performance not critical)
	for i := 0; i < len(sortedEnergies); i++ {
		for j := i + 1; j < len(sortedEnergies); j++ {
			if sortedEnergies[i] > sortedEnergies[j] {
				sortedEnergies[i], sortedEnergies[j] = sortedEnergies[j], sortedEnergies[i]
			}
		}
	}
	globalMedian := sortedEnergies[len(sortedEnergies)/2]

	// Avoid division by zero
	if globalMedian < 0.001 {
		// If median is very low, use mean instead
		sum := 0.0
		for _, e := range avgEnergies {
			sum += e
		}
		globalMedian = sum / float64(barCount)
		if globalMedian < 0.001 {
			globalMedian = 0.001
		}
	}

	// Get strength multiplier from config (default 1.0 if not set)
	strength := w.spectrumDynamicScaling
	if strength <= 0 {
		strength = 1.0
	}

	// Calculate per-bar gain
	for i := 0; i < barCount; i++ {
		if avgEnergies[i] < globalMedian && avgEnergies[i] > 0.001 {
			// Bar is below median, apply boost.
			// boost = 1.0 + (globalMedian / avgEnergy - 1.0) * strength
			boost := 1.0 + (globalMedian/avgEnergies[i]-1.0)*strength
			// Clamp to reasonable range: 1.0x to 4.0x
			if boost < 1.0 {
				boost = 1.0
			}
			if boost > 4.0 {
				boost = 4.0
			}
			gains[i] = boost
		} else {
			// Bar is at or above median, no boost needed
			gains[i] = 1.0
		}
	}

	return gains
}

// Render draws the visualization
func (w *AudioVisualizerWidget) Render() (image.Image, error) {
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
func (w *AudioVisualizerWidget) renderSpectrum(img *image.Gray) {
	barCount := len(w.spectrumData)
	if barCount == 0 {
		return
	}

	width := w.position.W
	height := w.position.H
	barWidth := width / barCount
	gap := 0
	if w.barStyle == shared.AudioBarStyleBars && barWidth > 2 {
		gap = 1
	}

	for i := 0; i < barCount; i++ {
		// Magnitude is already normalized to 0.0-1.0 range (dB scale)
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

		// Draw bar
		if w.barStyle == shared.AudioBarStyleBars {
			barW := barWidth - gap
			if x+barW > width {
				barW = width - x
			}
			if barW > 0 && barHeight > 0 {
				bitmap.DrawFilledRectangle(img, x, y, barW, barHeight, w.fillColor)
			}
		} else {
			// Line style - draw top pixel of each bar
			if y >= 0 && y < height {
				endX := x + barWidth - 1
				if endX >= width {
					endX = width - 1
				}
				bitmap.DrawHorizontalLine(img, x, endX, y, w.fillColor)
			}
		}

		// Draw peak hold
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
				endX := x + barWidth - gap - 1
				if endX >= width {
					endX = width - 1
				}
				bitmap.DrawHorizontalLine(img, x, endX, peakY, w.fillColor)
			}
		}
	}
}

// renderOscilloscope draws waveform
func (w *AudioVisualizerWidget) renderOscilloscope(img *image.Gray) {
	height := w.position.H
	sampleCount := w.sampleCount

	if w.channelMode == shared.AudioChannelModeMono || w.channelMode == shared.AudioChannelModeStereoCombined {
		// Use mixed channels for mono/combined modes
		if len(w.audioData) == 0 {
			return
		}
		if sampleCount > len(w.audioData) {
			sampleCount = len(w.audioData)
		}
		centerY := height / 2
		samples := w.audioData[len(w.audioData)-sampleCount:]
		w.drawWaveform(img, samples, 0, height, centerY, w.fillColor)
	} else if w.channelMode == shared.AudioChannelModeStereoSeparated {
		// Use separate left and right channels for stereo_separated mode
		if len(w.audioDataLeft) == 0 || len(w.audioDataRight) == 0 {
			return
		}

		// Get sample count based on available data
		leftSampleCount := sampleCount
		rightSampleCount := sampleCount
		if leftSampleCount > len(w.audioDataLeft) {
			leftSampleCount = len(w.audioDataLeft)
		}
		if rightSampleCount > len(w.audioDataRight) {
			rightSampleCount = len(w.audioDataRight)
		}

		leftSamples := w.audioDataLeft[len(w.audioDataLeft)-leftSampleCount:]
		rightSamples := w.audioDataRight[len(w.audioDataRight)-rightSampleCount:]

		// Top half - actual left channel
		w.drawWaveform(img, leftSamples, 0, height/2, height/4, w.leftChannelColor)

		// Bottom half - actual right channel
		w.drawWaveform(img, rightSamples, height/2, height, height*3/4, w.rightChannelColor)
	}
}

// drawWaveform draws a single waveform in the specified region
func (w *AudioVisualizerWidget) drawWaveform(img *image.Gray, samples []float32, yStart, yEnd, centerY int, fillColor uint8) {
	width := w.position.W
	sampleCount := len(samples)

	// Use full height for amplitude (samples are in -1.0 to +1.0 range)
	// Audio is already normalized by AGC, use actual amplitude
	amplitude := float32(yEnd-yStart) / 2.0

	for i := 0; i < sampleCount-1; i++ {
		x1 := i * width / sampleCount
		x2 := (i + 1) * width / sampleCount

		y1 := centerY - int(samples[i]*amplitude)
		y2 := centerY - int(samples[i+1]*amplitude)

		// Clamp to region
		if y1 < yStart {
			y1 = yStart
		}
		if y1 >= yEnd {
			y1 = yEnd - 1
		}
		if y2 < yStart {
			y2 = yStart
		}
		if y2 >= yEnd {
			y2 = yEnd - 1
		}

		if w.waveformStyle == shared.AudioWaveformStyleLine {
			bitmap.DrawLine(img, x1, y1, x2, y2, color.Gray{Y: fillColor})
		} else if w.waveformStyle == shared.AudioWaveformStyleFilled {
			// Draw vertical line from center to sample
			if x1 >= 0 && x1 < width {
				if y1 < centerY {
					endY := centerY
					if endY >= yEnd {
						endY = yEnd - 1
					}
					bitmap.DrawVerticalLine(img, x1, y1, endY, fillColor)
				} else {
					endY := y1
					if endY >= yEnd {
						endY = yEnd - 1
					}
					bitmap.DrawVerticalLine(img, x1, centerY, endY, fillColor)
				}
			}
		}
	}
}

// Shared audio capture instance (recreatable singleton)
var (
	sharedAudioCapture    *AudioCaptureWCA
	sharedAudioCaptureMu  sync.Mutex
	sharedAudioCaptureErr error
)

// GetSharedAudioCapture returns the shared AudioCaptureWCA instance
// This can recreate the instance if it was previously invalidated
func GetSharedAudioCapture() (*AudioCaptureWCA, error) {
	sharedAudioCaptureMu.Lock()
	defer sharedAudioCaptureMu.Unlock()

	// Return existing instance if valid
	if sharedAudioCapture != nil && sharedAudioCapture.initialized {
		return sharedAudioCapture, nil
	}

	// Create new instance
	ac := &AudioCaptureWCA{}
	if err := ac.initialize(); err != nil {
		sharedAudioCaptureErr = fmt.Errorf("failed to initialize: %w", err)
		return nil, sharedAudioCaptureErr
	}

	sharedAudioCapture = ac
	sharedAudioCaptureErr = nil
	return sharedAudioCapture, nil
}

// AudioCaptureWCA captures audio using Windows Core Audio API in loopback mode
type AudioCaptureWCA struct {
	mu            sync.Mutex
	initialized   bool
	audioClient   *wca.IAudioClient
	captureClient *wca.IAudioCaptureClient
	mmd           *wca.IMMDevice
	mmde          *wca.IMMDeviceEnumerator
	bufferSize    uint32
	sampleRate    uint32
}

// initialize sets up WASAPI loopback capture
func (ac *AudioCaptureWCA) initialize() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Ensure COM is initialized on this thread
	err := EnsureCOMInitialized()
	if err != nil {
		return fmt.Errorf("failed to initialize COM: %w", err)
	}

	// Note: We don't own COM cleanup - it's managed per-thread by EnsureCOMInitialized

	// Create device enumerator
	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		return err
	}
	ac.mmde = mmde

	// Get default audio endpoint (render for loopback)
	mmd, err := GetDefaultRenderDevice(mmde)
	if err != nil {
		ac.cleanup()
		return err
	}
	ac.mmd = mmd

	// Activate IAudioClient
	var audioClientInterface *wca.IAudioClient
	if err := mmd.Activate(wca.IID_IAudioClient, wca.CLSCTX_ALL, nil, &audioClientInterface); err != nil {
		ac.cleanup()
		return fmt.Errorf("Activate IAudioClient failed: %w", err)
	}
	ac.audioClient = audioClientInterface

	// Get mix format
	var wfx *wca.WAVEFORMATEX
	if err := audioClientInterface.GetMixFormat(&wfx); err != nil {
		ac.cleanup()
		return fmt.Errorf("GetMixFormat failed: %w", err)
	}
	ac.sampleRate = wfx.NSamplesPerSec

	// Initialize audio client in loopback mode
	// AudclntStreamflagsLoopback = 0x00020000
	const AudclntStreamflagsLoopback = 0x00020000
	const refTimesPerSec = 10000000                           // 100ns units
	bufferDuration := wca.REFERENCE_TIME(refTimesPerSec / 50) // 20ms buffer

	if err := audioClientInterface.Initialize(
		wca.AUDCLNT_SHAREMODE_SHARED,
		AudclntStreamflagsLoopback,
		bufferDuration,
		0,
		wfx,
		nil,
	); err != nil {
		ole.CoTaskMemFree(uintptr(unsafe.Pointer(wfx)))
		ac.cleanup()
		return fmt.Errorf("Initialize failed: %w", err)
	}

	ole.CoTaskMemFree(uintptr(unsafe.Pointer(wfx)))

	// Get buffer size
	if err := audioClientInterface.GetBufferSize(&ac.bufferSize); err != nil {
		ac.cleanup()
		return fmt.Errorf("GetBufferSize failed: %w", err)
	}

	// Get capture client
	var captureClient *wca.IAudioCaptureClient
	if err := audioClientInterface.GetService(wca.IID_IAudioCaptureClient, &captureClient); err != nil {
		ac.cleanup()
		return fmt.Errorf("GetService IAudioCaptureClient failed: %w", err)
	}
	ac.captureClient = captureClient

	// Start capture
	if err := audioClientInterface.Start(); err != nil {
		ac.cleanup()
		return fmt.Errorf("Start failed: %w", err)
	}

	ac.initialized = true
	return nil
}

// AUDCLNT_S_BUFFER_EMPTY is a success code indicating no audio data available
// This is normal when nothing is playing - not an error
const audclntSBufferEmpty = 0x08890001

// ReadSamples reads available audio samples (stereo float32)
// Returns (leftChannel, rightChannel, error)
func (ac *AudioCaptureWCA) ReadSamples() ([]float32, []float32, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.initialized {
		return nil, nil, fmt.Errorf("not initialized")
	}

	leftSamples := make([]float32, 0, 4096)
	rightSamples := make([]float32, 0, 4096)
	hadRealError := false
	var lastErr error

	// Read all available packets
	for {
		var numFramesToRead uint32
		var flags uint32
		var pData *byte

		err := ac.captureClient.GetBuffer(&pData, &numFramesToRead, &flags, nil, nil)
		if err != nil {
			// Check if this is just "buffer empty" (normal when no audio playing)
			var oleErr *ole.OleError
			if errors.As(err, &oleErr) {
				errCode := uint32(oleErr.Code())
				if errCode == audclntSBufferEmpty {
					// Buffer empty is normal - just means no audio data right now
					break
				}
			}
			// Real error
			hadRealError = true
			lastErr = err
			break
		}

		if numFramesToRead == 0 {
			_ = ac.captureClient.ReleaseBuffer(numFramesToRead)
			break
		}

		// Convert samples to float32 (assuming format is float32 stereo interleaved: L-R-L-R)
		// Extract both left and right channels
		dataSlice := (*[1 << 30]float32)(unsafe.Pointer(pData))[:numFramesToRead*2]
		for i := uint32(0); i < numFramesToRead; i++ {
			leftSamples = append(leftSamples, dataSlice[i*2])     // Left channel
			rightSamples = append(rightSamples, dataSlice[i*2+1]) // Right channel
		}

		_ = ac.captureClient.ReleaseBuffer(numFramesToRead)
	}

	// Return error only if we had a real error (not buffer empty)
	if hadRealError && len(leftSamples) == 0 {
		return nil, nil, fmt.Errorf("audio capture failed: %w", lastErr)
	}

	return leftSamples, rightSamples, nil
}

// cleanup releases COM resources
func (ac *AudioCaptureWCA) cleanup() {
	// Stop audio client before releasing
	if ac.audioClient != nil {
		_ = ac.audioClient.Stop()
	}

	SafeReleaseAudioClient(&ac.audioClient)
	SafeReleaseAudioCaptureClient(&ac.captureClient)
	SafeReleaseMMDevice(&ac.mmd)
	SafeReleaseMMDeviceEnumerator(&ac.mmde)
}

// Close releases resources
func (ac *AudioCaptureWCA) Close() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.cleanup()
}
