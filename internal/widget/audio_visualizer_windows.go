//go:build windows

package widget

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/cmplx"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/mjibson/go-dsp/fft"
	"github.com/moutend/go-wca/pkg/wca"
	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// frequencyCompensationCurve defines frequency-dependent gain to balance spectrum display
// Each entry: max frequency (Hz) and gain multiplier
var frequencyCompensationCurve = []struct {
	maxFreq float64
	gain    float64
}{
	{100, 0.5},    // Sub-bass: attenuate
	{250, 0.8},    // Bass: slight attenuation
	{500, 1.0},    // Low-mids: neutral
	{1000, 2.0},   // Mids: moderate boost
	{2000, 3.5},   // Upper mids: strong boost
	{4000, 5.0},   // Highs: very strong boost
	{8000, 7.0},   // Upper highs: extreme boost
	{99999, 10.0}, // Very high frequencies: maximum boost
}

// AudioVisualizerWidget displays real-time spectrum analyzer or oscilloscope
type AudioVisualizerWidget struct {
	id             string
	position       config.PositionConfig
	style          config.StyleConfig
	properties     config.WidgetProperties
	updateInterval time.Duration
	audioCapture   *AudioCaptureWCA
	volumeReader   *VolumeReaderWCA
	mu             sync.Mutex
	audioData      []float32 // Latest audio samples
	spectrumData   []float64 // Spectrum magnitudes
	peakValues     []float64 // Peak hold values
	peakTimestamps []time.Time
	smoothedValues []float64 // Smoothed spectrum values
	lastUpdateTime time.Time
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

	barCount := cfg.Properties.BarCount
	if barCount == 0 {
		barCount = 32
	}

	w := &AudioVisualizerWidget{
		id:             cfg.ID,
		position:       cfg.Position,
		style:          cfg.Style,
		properties:     cfg.Properties,
		updateInterval: time.Duration(cfg.Properties.UpdateInterval * float64(time.Second)),
		audioCapture:   capture,
		volumeReader:   volumeReader,
		spectrumData:   make([]float64, barCount),
		peakValues:     make([]float64, barCount),
		peakTimestamps: make([]time.Time, barCount),
		smoothedValues: make([]float64, barCount),
		audioData:      make([]float32, 0, 4096),
	}

	return w, nil
}

func (w *AudioVisualizerWidget) Name() string {
	return w.id
}

func (w *AudioVisualizerWidget) GetUpdateInterval() time.Duration {
	return w.updateInterval
}

func (w *AudioVisualizerWidget) GetPosition() config.PositionConfig {
	return w.position
}

func (w *AudioVisualizerWidget) GetStyle() config.StyleConfig {
	return w.style
}

// Update captures audio and processes it
func (w *AudioVisualizerWidget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Capture audio samples
	samples, err := w.audioCapture.ReadSamples()
	if err != nil {
		return fmt.Errorf("failed to read audio samples: %w", err)
	}

	// If no samples, create silent buffer to allow peaks to decay
	if len(samples) == 0 {
		samples = make([]float32, 1024) // Silent buffer (all zeros)
	}

	// Apply volume compensation
	// WASAPI loopback captures audio AFTER system volume is applied,
	// so we need to compensate to show visualization independent of volume level
	volumePercent, _, err := w.volumeReader.GetVolume()
	if err == nil && volumePercent > 1.0 { // Avoid division by very small numbers (1% minimum)
		// Multiply by (100.0 / volumePercent) to restore original signal
		// Example: at 30% volume, multiply by 100/30 = 3.33x
		gainFactor := float32(100.0 / volumePercent)

		// Apply gain to all samples
		for i := range samples {
			samples[i] *= gainFactor
			// Clamp to -1.0 to +1.0 range
			if samples[i] > 1.0 {
				samples[i] = 1.0
			}
			if samples[i] < -1.0 {
				samples[i] = -1.0
			}
		}
	}

	// Accumulate samples into buffer (keep last 8192 samples for FFT)
	maxSamples := 8192
	w.audioData = append(w.audioData, samples...)
	if len(w.audioData) > maxSamples {
		w.audioData = w.audioData[len(w.audioData)-maxSamples:]
	}

	// Process for spectrum mode (always update, even with silence to decay peaks)
	// Use accumulated audioData buffer, not just current samples
	if w.properties.DisplayMode == "spectrum" {
		w.updateSpectrum(w.audioData)
	}

	w.lastUpdateTime = time.Now()
	return nil
}

// updateSpectrum performs FFT and updates spectrum data
func (w *AudioVisualizerWidget) updateSpectrum(samples []float32) {
	// Need power of 2 samples for FFT
	// Use 8192 for better low-frequency resolution (5.86 Hz per bin at 48kHz)
	fftSize := 8192
	if len(samples) < fftSize {
		return
	}

	// Take last fftSize samples and convert to complex
	fftSamples := samples[len(samples)-fftSize:]
	input := make([]complex128, fftSize)
	for i := 0; i < fftSize; i++ {
		// Apply Hann window
		window := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(fftSize-1)))
		input[i] = complex(float64(fftSamples[i])*window, 0)
	}

	// Perform FFT
	output := fft.FFT(input)

	// Calculate magnitudes for first half (positive frequencies)
	halfSize := fftSize / 2
	magnitudes := make([]float64, halfSize)
	// Sensitivity multiplier for visualization
	// Balanced for music playback - not too high (clipping on loud parts) or too low (weak display)
	sensitivity := 70.0
	for i := 0; i < halfSize; i++ {
		// Calculate magnitude and apply sensitivity
		mag := cmplx.Abs(output[i]) / float64(fftSize)
		magnitudes[i] = math.Min(1.0, mag*sensitivity)
	}

	// Map frequencies to bars
	barCount := len(w.spectrumData)

	if w.properties.FrequencyScale == "logarithmic" {
		w.mapFrequenciesLogarithmic(magnitudes, barCount)
	} else {
		w.mapFrequenciesLinear(magnitudes, barCount)
	}

	// Apply smoothing
	smoothing := w.properties.Smoothing
	dt := time.Since(w.lastUpdateTime).Seconds()
	if dt > 0 && smoothing > 0 {
		for i := range w.smoothedValues {
			// Exponential moving average with time-based decay
			alpha := 1.0 - math.Pow(smoothing, dt*30) // Adjust for frame rate
			w.smoothedValues[i] = alpha*w.spectrumData[i] + (1-alpha)*w.smoothedValues[i]
		}
	} else {
		copy(w.smoothedValues, w.spectrumData)
	}

	// Update peak hold
	if w.properties.PeakHold {
		now := time.Now()
		for i := range w.peakValues {
			if w.smoothedValues[i] > w.peakValues[i] {
				w.peakValues[i] = w.smoothedValues[i]
				w.peakTimestamps[i] = now
			} else {
				// Decay peak if hold time expired
				elapsed := now.Sub(w.peakTimestamps[i]).Seconds()
				if elapsed > w.properties.PeakHoldTime {
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
			if w.properties.FrequencyCompensation {
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
		if w.properties.FrequencyCompensation {
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
func (w *AudioVisualizerWidget) Render() (image.Image, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	img := bitmap.NewGrayscaleImage(w.position.W, w.position.H, uint8(w.style.BackgroundColor))

	if w.properties.DisplayMode == "spectrum" {
		w.renderSpectrum(img)
	} else if w.properties.DisplayMode == "oscilloscope" {
		w.renderOscilloscope(img)
	}

	if w.style.Border {
		bitmap.DrawBorder(img, uint8(w.style.BorderColor))
	}

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
	if w.properties.BarStyle == "bars" && barWidth > 2 {
		gap = 1
	}

	fillColor := uint8(w.properties.FillColor)

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
		if w.properties.BarStyle == "bars" {
			for py := y; py < height; py++ {
				for px := x; px < x+barWidth-gap && px < width; px++ {
					img.SetGray(px, py, color.Gray{Y: fillColor})
				}
			}
		} else {
			// Line style - draw top pixel of each bar
			for px := x; px < x+barWidth && px < width; px++ {
				if y >= 0 && y < height {
					img.SetGray(px, y, color.Gray{Y: fillColor})
				}
			}
		}

		// Draw peak hold
		if w.properties.PeakHold && i < len(w.peakValues) {
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
					img.SetGray(px, peakY, color.Gray{Y: fillColor})
				}
			}
		}
	}
}

// renderOscilloscope draws waveform
func (w *AudioVisualizerWidget) renderOscilloscope(img *image.Gray) {
	if len(w.audioData) == 0 {
		return
	}

	height := w.position.H
	sampleCount := w.properties.SampleCount
	if sampleCount > len(w.audioData) {
		sampleCount = len(w.audioData)
	}

	centerY := height / 2
	samples := w.audioData[len(w.audioData)-sampleCount:]

	if w.properties.ChannelMode == "mono" || w.properties.ChannelMode == "stereo_combined" {
		// Draw single waveform (center)
		fillColor := uint8(w.properties.FillColor)
		w.drawWaveform(img, samples, 0, height, centerY, fillColor)
	} else if w.properties.ChannelMode == "stereo_separated" {
		// Draw two waveforms (split)
		leftColor := uint8(w.properties.LeftChannelColor)
		rightColor := uint8(w.properties.RightChannelColor)

		// Top half - left channel
		w.drawWaveform(img, samples, 0, height/2, height/4, leftColor)

		// Bottom half - right channel
		w.drawWaveform(img, samples, height/2, height, height*3/4, rightColor)
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

		if w.properties.WaveformStyle == "line" {
			bitmap.DrawLine(img, x1, y1, x2, y2, color.Gray{Y: fillColor})
		} else if w.properties.WaveformStyle == "filled" {
			// Draw vertical line from center to sample
			if y1 < centerY {
				for y := y1; y <= centerY && y < yEnd; y++ {
					if x1 >= 0 && x1 < width {
						img.SetGray(x1, y, color.Gray{Y: fillColor})
					}
				}
			} else {
				for y := centerY; y <= y1 && y < yEnd; y++ {
					if x1 >= 0 && x1 < width {
						img.SetGray(x1, y, color.Gray{Y: fillColor})
					}
				}
			}
		}
	}
}

// Shared audio capture instance (singleton)
var (
	sharedAudioCapture     *AudioCaptureWCA
	sharedAudioCaptureOnce sync.Once
	sharedAudioCaptureErr  error
)

// GetSharedAudioCapture returns the shared AudioCaptureWCA instance
func GetSharedAudioCapture() (*AudioCaptureWCA, error) {
	sharedAudioCaptureOnce.Do(func() {
		ac := &AudioCaptureWCA{}
		if err := ac.initialize(); err != nil {
			sharedAudioCaptureErr = fmt.Errorf("failed to initialize: %w", err)
			return
		}
		sharedAudioCapture = ac
	})

	if sharedAudioCaptureErr != nil {
		return nil, sharedAudioCaptureErr
	}

	return sharedAudioCapture, nil
}

// AudioCaptureWCA captures audio using Windows Core Audio API in loopback mode
type AudioCaptureWCA struct {
	mu             sync.Mutex
	initialized    bool
	comInitialized bool
	threadLocked   bool
	audioClient    *wca.IAudioClient
	captureClient  *wca.IAudioCaptureClient
	mmd            *wca.IMMDevice
	mmde           *wca.IMMDeviceEnumerator
	bufferSize     uint32
	sampleRate     uint32
}

// initialize sets up WASAPI loopback capture
func (ac *AudioCaptureWCA) initialize() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Lock thread for COM
	runtime.LockOSThread()
	ac.threadLocked = true

	// Initialize COM
	err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "Incorrect function." || errMsg == "Cannot change thread mode after it is set." {
			// COM already initialized on this thread
		} else {
			runtime.UnlockOSThread()
			ac.threadLocked = false
			return fmt.Errorf("CoInitializeEx failed: %w", err)
		}
	} else {
		ac.comInitialized = true
	}

	// Create device enumerator
	var mmde *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		if ac.comInitialized {
			ole.CoUninitialize()
		}
		if ac.threadLocked {
			runtime.UnlockOSThread()
			ac.threadLocked = false
		}
		return fmt.Errorf("CoCreateInstance failed: %w", err)
	}
	ac.mmde = mmde

	// Get default audio endpoint (render for loopback)
	var mmd *wca.IMMDevice
	if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
		ac.cleanup()
		return fmt.Errorf("GetDefaultAudioEndpoint failed: %w", err)
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

// ReadSamples reads available audio samples (mono float32)
func (ac *AudioCaptureWCA) ReadSamples() ([]float32, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.initialized {
		return nil, fmt.Errorf("not initialized")
	}

	samples := make([]float32, 0, 4096)
	totalFrames := 0

	// Read all available packets
	for {
		var numFramesToRead uint32
		var flags uint32
		var pData *byte

		err := ac.captureClient.GetBuffer(&pData, &numFramesToRead, &flags, nil, nil)
		if err != nil {
			// No more data available
			break
		}

		if numFramesToRead == 0 {
			_ = ac.captureClient.ReleaseBuffer(numFramesToRead)
			break
		}

		totalFrames += int(numFramesToRead)

		// Convert samples to float32 (assuming format is float32 stereo)
		// Take left channel only for mono
		dataSlice := (*[1 << 30]float32)(unsafe.Pointer(pData))[:numFramesToRead*2]
		for i := uint32(0); i < numFramesToRead; i++ {
			samples = append(samples, dataSlice[i*2]) // Left channel
		}

		_ = ac.captureClient.ReleaseBuffer(numFramesToRead)
	}

	return samples, nil
}

// cleanup releases COM resources
func (ac *AudioCaptureWCA) cleanup() {
	if ac.audioClient != nil {
		_ = ac.audioClient.Stop()
		ac.audioClient.Release()
	}
	if ac.captureClient != nil {
		ac.captureClient.Release()
	}
	if ac.mmd != nil {
		ac.mmd.Release()
	}
	if ac.mmde != nil {
		ac.mmde.Release()
	}
	if ac.comInitialized {
		ole.CoUninitialize()
	}
	if ac.threadLocked {
		runtime.UnlockOSThread()
	}
}

// Close releases resources
func (ac *AudioCaptureWCA) Close() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.cleanup()
}
