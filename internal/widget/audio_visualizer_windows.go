//go:build windows

package widget

import (
	"fmt"
	"image"
	"image/color"
	"log"
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

		// Calculate RMS before and after for debugging
		var rmsBefore, rmsAfter float64
		var sumSquares float64
		for _, sample := range samples {
			sumSquares += float64(sample * sample)
		}
		rmsBefore = math.Sqrt(sumSquares / float64(len(samples)))

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

		// Calculate RMS after compensation
		sumSquares = 0
		for _, sample := range samples {
			sumSquares += float64(sample * sample)
		}
		rmsAfter = math.Sqrt(sumSquares / float64(len(samples)))

		log.Printf("[VOLUME-COMP] Volume: %.1f%%, Gain: %.2fx, RMS before: %.4f, RMS after: %.4f",
			volumePercent, gainFactor, rmsBefore, rmsAfter)
	}

	// Store samples for oscilloscope mode (keep last 4096 samples)
	maxSamples := 4096
	if len(samples) > maxSamples {
		w.audioData = samples[len(samples)-maxSamples:]
	} else {
		w.audioData = samples
	}

	// Process for spectrum mode (always update, even with silence to decay peaks)
	if w.properties.DisplayMode == "spectrum" {
		w.updateSpectrum(samples)
	}

	w.lastUpdateTime = time.Now()
	return nil
}

// updateSpectrum performs FFT and updates spectrum data
func (w *AudioVisualizerWidget) updateSpectrum(samples []float32) {
	// Need power of 2 samples for FFT
	fftSize := 1024
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
	// Tuned for typical music levels (peaks around 0.5-1.0)
	sensitivity := 80.0
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

	// Debug: Log spectrum data for first few bars (bass region)
	if barCount >= 5 {
		log.Printf("[SPECTRUM-DEBUG] First 5 bars: [%.4f, %.4f, %.4f, %.4f, %.4f]",
			w.spectrumData[0], w.spectrumData[1], w.spectrumData[2], w.spectrumData[3], w.spectrumData[4])
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
					// Gradual decay
					decay := 0.95
					w.peakValues[i] *= decay
				}
			}
		}
	}
}

// mapFrequenciesLogarithmic maps FFT bins to bars using logarithmic scale
func (w *AudioVisualizerWidget) mapFrequenciesLogarithmic(magnitudes []float64, barCount int) {
	// Logarithmic frequency mapping (similar to Winamp)
	minFreq := 20.0    // Hz
	maxFreq := 20000.0 // Hz

	sampleRate := 48000.0 // Assume 48kHz
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

		// Average magnitudes in this frequency range
		sum := 0.0
		count := 0
		for j := binStart; j <= binEnd && j < len(magnitudes); j++ {
			sum += magnitudes[j]
			count++
		}

		if count > 0 {
			w.spectrumData[i] = sum / float64(count)
		} else {
			w.spectrumData[i] = 0
		}
	}
}

// mapFrequenciesLinear maps FFT bins to bars using linear scale
func (w *AudioVisualizerWidget) mapFrequenciesLinear(magnitudes []float64, barCount int) {
	binsPerBar := len(magnitudes) / barCount

	for i := 0; i < barCount; i++ {
		start := i * binsPerBar
		end := start + binsPerBar
		if end > len(magnitudes) {
			end = len(magnitudes)
		}

		// Average magnitudes
		sum := 0.0
		for j := start; j < end; j++ {
			sum += magnitudes[j]
		}
		w.spectrumData[i] = sum / float64(binsPerBar)
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

// NewAudioCaptureWCA creates a new audio capture instance
func NewAudioCaptureWCA() (*AudioCaptureWCA, error) {
	ac := &AudioCaptureWCA{}
	if err := ac.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}
	return ac, nil
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
			log.Printf("[AUDIO-CAPTURE] COM already initialized on this thread")
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

	log.Printf("[AUDIO-CAPTURE] Format: %d Hz, %d channels, %d bits/sample, format tag: %d",
		wfx.NSamplesPerSec, wfx.NChannels, wfx.WBitsPerSample, wfx.WFormatTag)

	// Initialize audio client in loopback mode
	// AUDCLNT_STREAMFLAGS_LOOPBACK = 0x00020000
	const AUDCLNT_STREAMFLAGS_LOOPBACK = 0x00020000
	const refTimesPerSec = 10000000                           // 100ns units
	bufferDuration := wca.REFERENCE_TIME(refTimesPerSec / 50) // 20ms buffer

	if err := audioClientInterface.Initialize(
		wca.AUDCLNT_SHAREMODE_SHARED,
		AUDCLNT_STREAMFLAGS_LOOPBACK,
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
	log.Printf("[AUDIO-CAPTURE] Initialized successfully (sample rate: %d Hz, buffer: %d frames)", ac.sampleRate, ac.bufferSize)
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
			ac.captureClient.ReleaseBuffer(numFramesToRead)
			break
		}

		totalFrames += int(numFramesToRead)

		// Convert samples to float32 (assuming format is float32 stereo)
		// Take left channel only for mono
		dataSlice := (*[1 << 30]float32)(unsafe.Pointer(pData))[:numFramesToRead*2]
		for i := uint32(0); i < numFramesToRead; i++ {
			samples = append(samples, dataSlice[i*2]) // Left channel
		}

		ac.captureClient.ReleaseBuffer(numFramesToRead)
	}

	// Debug: Log sample info periodically
	if len(samples) > 0 && totalFrames > 0 {
		// Calculate RMS to see if there's actual audio
		var sumSquares float64
		for _, s := range samples {
			sumSquares += float64(s * s)
		}
		rms := math.Sqrt(sumSquares / float64(len(samples)))
		if rms > 0.001 { // Only log if there's non-trivial audio
			log.Printf("[AUDIO-CAPTURE] Read %d frames, RMS: %.4f, peak: %.4f", totalFrames, rms, maxAbs(samples))
		}
	}

	return samples, nil
}

// maxAbs returns the maximum absolute value in a slice
func maxAbs(samples []float32) float32 {
	var max float32
	for _, s := range samples {
		abs := s
		if abs < 0 {
			abs = -abs
		}
		if abs > max {
			max = abs
		}
	}
	return max
}

// cleanup releases COM resources
func (ac *AudioCaptureWCA) cleanup() {
	if ac.audioClient != nil {
		ac.audioClient.Stop()
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
