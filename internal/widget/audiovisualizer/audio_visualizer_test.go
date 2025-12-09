//go:build windows && integration

// Audio Visualizer Integration Tests
//
// These tests require:
// - Windows OS with audio hardware
// - COM initialization support
// - Run sequentially (not in parallel) due to COM threading model
//
// To run these tests:
//   go test -tags=integration ./internal/widget/audiovisualizer -run TestNew
//
// These tests are NOT run by default because:
// 1. They require real audio hardware
// 2. The shared singleton pattern used by Widget doesn't work well in parallel test execution
// 3. COM initialization is thread-specific and conflicts in test environments
// 4. They fail in CI environments without audio devices

package audiovisualizer

import (
	"image"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNew_Spectrum(t *testing.T) {

	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_spectrum",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Mode:           "spectrum",
		UpdateInterval: 0.033,
		Spectrum: &config.SpectrumConfig{
			Bars:                  32,
			Scale:                 "logarithmic",
			Style:                 "bars",
			Smoothing:             0.7,
			FrequencyCompensation: true,
			Peak: &config.PeakConfig{
				Enabled:  true,
				HoldTime: 1.0,
			},
			DynamicScaling: &config.DynamicScalingConfig{
				Strength: 1.0,
			},
			Colors: &config.AudioColorsConfig{
				Fill: config.IntPtr(255),
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w == nil {
		t.Fatal("New() returned nil")
	}

	if w.Name() != "test_spectrum" {
		t.Errorf("Name() = %s, want test_spectrum", w.Name())
	}

	avWidget := w.(*Widget)
	if avWidget.displayMode != "spectrum" {
		t.Errorf("displayMode = %s, want spectrum", avWidget.displayMode)
	}

	if avWidget.barCount != 32 {
		t.Errorf("barCount = %d, want 32", avWidget.barCount)
	}
}

func TestNew_Oscilloscope(t *testing.T) {

	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_oscilloscope",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Mode:           "oscilloscope",
		UpdateInterval: 0.033,
		Channel:        "stereo_separated",
		Oscilloscope: &config.OscilloscopeConfig{
			Samples: 128,
			Style:   "line",
			Colors: &config.AudioColorsConfig{
				Left:  config.IntPtr(255),
				Right: config.IntPtr(200),
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w == nil {
		t.Fatal("New() returned nil")
	}

	avWidget := w.(*Widget)
	if avWidget.displayMode != "oscilloscope" {
		t.Errorf("displayMode = %s, want oscilloscope", avWidget.displayMode)
	}

	if avWidget.waveformStyle != "line" {
		t.Errorf("waveformStyle = %s, want line", avWidget.waveformStyle)
	}

	if avWidget.channelMode != "stereo_separated" {
		t.Errorf("channelMode = %s, want stereo_separated", avWidget.channelMode)
	}
}

func TestNew_DefaultBarCount(t *testing.T) {

	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_defaults",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			Background: 0,
		},
		Mode: "spectrum",
		// BarCount not specified - should default to 32
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	avWidget := w.(*Widget)

	// Check that barCount was set to default (32)
	if len(avWidget.spectrumData) != 32 {
		t.Errorf("Default barCount spectrumData length = %d, want 32", len(avWidget.spectrumData))
	}

	if len(avWidget.peakValues) != 32 {
		t.Errorf("Default barCount peakValues length = %d, want 32", len(avWidget.peakValues))
	}
}

func TestNew_DynamicScalingWindow(t *testing.T) {

	tests := []struct {
		name           string
		updateInterval float64
		scalingWindow  float64
		wantWindowSize int
	}{
		{
			name:           "default 0.5s window",
			updateInterval: 0.033,
			scalingWindow:  0.5,
			wantWindowSize: 16, // 0.5 / 0.033 ~ 15.15, +1 = 16
		},
		{
			name:           "1 second window",
			updateInterval: 0.05,
			scalingWindow:  1.0,
			wantWindowSize: 21, // 1.0 / 0.05 = 20, +1 = 21
		},
		{
			name:           "minimum window size",
			updateInterval: 1.0,
			scalingWindow:  0.1,
			wantWindowSize: 2, // Minimum enforced
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "audio_visualizer",
				ID:      "test_window",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 128,
					H: 40,
				},
				Style: config.StyleConfig{
					Background: 0,
				},
				Mode:           "spectrum",
				UpdateInterval: tt.updateInterval,
				Spectrum: &config.SpectrumConfig{
					Bars: 10,
					DynamicScaling: &config.DynamicScalingConfig{
						Window: tt.scalingWindow,
					},
				},
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			avWidget := w.(*Widget)
			if avWidget.barEnergyWindowSize != tt.wantWindowSize {
				t.Errorf("barEnergyWindowSize = %d, want %d", avWidget.barEnergyWindowSize, tt.wantWindowSize)
			}

			// Verify energy history array was initialized correctly
			if len(avWidget.barEnergyHistory) != 10 {
				t.Errorf("barEnergyHistory rows = %d, want 10", len(avWidget.barEnergyHistory))
			}

			for i, row := range avWidget.barEnergyHistory {
				if len(row) != tt.wantWindowSize {
					t.Errorf("barEnergyHistory[%d] columns = %d, want %d", i, len(row), tt.wantWindowSize)
				}
			}
		})
	}
}

func TestWidget_Render_Spectrum(t *testing.T) {

	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_render",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Mode:           "spectrum",
		UpdateInterval: 0.033,
		Spectrum: &config.SpectrumConfig{
			Bars: 32,
			Colors: &config.AudioColorsConfig{
				Fill: config.IntPtr(255),
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Render without update (should work with zero data)
	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	if img.Bounds().Dx() != 128 {
		t.Errorf("image width = %d, want 128", img.Bounds().Dx())
	}

	if img.Bounds().Dy() != 40 {
		t.Errorf("image height = %d, want 40", img.Bounds().Dy())
	}
}

func TestWidget_Render_Oscilloscope(t *testing.T) {

	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_render_osc",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Mode:           "oscilloscope",
		UpdateInterval: 0.033,
		Channel:        "stereo_combined",
		Oscilloscope: &config.OscilloscopeConfig{
			Samples: 128,
			Style:   "line",
			Colors: &config.AudioColorsConfig{
				Left:  config.IntPtr(255),
				Right: config.IntPtr(200),
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Render without update (should work with zero data)
	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	if img.Bounds().Dx() != 128 {
		t.Errorf("image width = %d, want 128", img.Bounds().Dx())
	}

	if img.Bounds().Dy() != 40 {
		t.Errorf("image height = %d, want 40", img.Bounds().Dy())
	}
}

func TestWidget_BarStyles(t *testing.T) {

	barStyles := []string{"bars", "gradient", "dots"}

	for _, style := range barStyles {
		t.Run(style, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "audio_visualizer",
				ID:      "test_bar_style",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 128,
					H: 40,
				},
				Style: config.StyleConfig{
					Background: 0,
				},
				Mode: "spectrum",
				Spectrum: &config.SpectrumConfig{
					Bars:  16,
					Style: style,
					Colors: &config.AudioColorsConfig{
						Fill: config.IntPtr(255),
					},
				},
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() with %s error = %v", style, err)
			}

			// Should render without error
			img, err := w.Render()
			if err != nil {
				t.Errorf("Render() with %s error = %v", style, err)
			}

			if img == nil {
				t.Errorf("Render() with %s returned nil image", style)
			}

			avWidget := w.(*Widget)
			if avWidget.barStyle != style {
				t.Errorf("barStyle = %s, want %s", avWidget.barStyle, style)
			}
		})
	}
}

func TestWidget_FrequencyScales(t *testing.T) {

	scales := []string{"linear", "logarithmic"}

	for _, scale := range scales {
		t.Run(scale, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "audio_visualizer",
				ID:      "test_freq_scale",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 128,
					H: 40,
				},
				Style: config.StyleConfig{
					Background: 0,
				},
				Mode: "spectrum",
				Spectrum: &config.SpectrumConfig{
					Bars:  16,
					Scale: scale,
					Colors: &config.AudioColorsConfig{
						Fill: config.IntPtr(255),
					},
				},
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() with %s error = %v", scale, err)
			}

			// Should render without error
			img, err := w.Render()
			if err != nil {
				t.Errorf("Render() with %s error = %v", scale, err)
			}

			if img == nil {
				t.Errorf("Render() with %s returned nil image", scale)
			}

			avWidget := w.(*Widget)
			if avWidget.frequencyScale != scale {
				t.Errorf("frequencyScale = %s, want %s", avWidget.frequencyScale, scale)
			}
		})
	}
}

func TestWidget_WaveformStyles(t *testing.T) {

	styles := []string{"line", "filled"}

	for _, style := range styles {
		t.Run(style, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "audio_visualizer",
				ID:      "test_waveform_style",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 128,
					H: 40,
				},
				Style: config.StyleConfig{
					Background: 0,
				},
				Mode: "oscilloscope",
				Oscilloscope: &config.OscilloscopeConfig{
					Samples: 128,
					Style:   style,
					Colors: &config.AudioColorsConfig{
						Fill: config.IntPtr(255),
					},
				},
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() with %s error = %v", style, err)
			}

			// Should render without error
			img, err := w.Render()
			if err != nil {
				t.Errorf("Render() with %s error = %v", style, err)
			}

			if img == nil {
				t.Errorf("Render() with %s returned nil image", style)
			}

			avWidget := w.(*Widget)
			if avWidget.waveformStyle != style {
				t.Errorf("waveformStyle = %s, want %s", avWidget.waveformStyle, style)
			}
		})
	}
}

func TestWidget_ChannelModes(t *testing.T) {

	modes := []string{"stereo_combined", "stereo_separated", "left_only", "right_only"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "audio_visualizer",
				ID:      "test_channel_mode",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 128,
					H: 40,
				},
				Style: config.StyleConfig{
					Background: 0,
				},
				Mode:    "oscilloscope",
				Channel: mode,
				Oscilloscope: &config.OscilloscopeConfig{
					Samples: 128,
					Colors: &config.AudioColorsConfig{
						Left:  config.IntPtr(255),
						Right: config.IntPtr(200),
					},
				},
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() with %s error = %v", mode, err)
			}

			// Should render without error
			img, err := w.Render()
			if err != nil {
				t.Errorf("Render() with %s error = %v", mode, err)
			}

			if img == nil {
				t.Errorf("Render() with %s returned nil image", mode)
			}

			avWidget := w.(*Widget)
			if avWidget.channelMode != mode {
				t.Errorf("channelMode = %s, want %s", avWidget.channelMode, mode)
			}
		})
	}
}

func TestWidget_DynamicScaling(t *testing.T) {

	tests := []struct {
		name           string
		dynamicScaling float64
		wantEnabled    bool
	}{
		{
			name:           "disabled (0.0)",
			dynamicScaling: 0.0,
			wantEnabled:    false,
		},
		{
			name:           "moderate (1.0)",
			dynamicScaling: 1.0,
			wantEnabled:    true,
		},
		{
			name:           "strong (2.0)",
			dynamicScaling: 2.0,
			wantEnabled:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "audio_visualizer",
				ID:      "test_dynamic_scaling",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 128,
					H: 40,
				},
				Style: config.StyleConfig{
					Background: 0,
				},
				Mode: "spectrum",
				Spectrum: &config.SpectrumConfig{
					Bars: 16,
					DynamicScaling: &config.DynamicScalingConfig{
						Strength: tt.dynamicScaling,
					},
				},
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			avWidget := w.(*Widget)
			if avWidget.spectrumDynamicScaling != tt.dynamicScaling {
				t.Errorf("spectrumDynamicScaling = %f, want %f", avWidget.spectrumDynamicScaling, tt.dynamicScaling)
			}

			// Render should work regardless of dynamic scaling setting
			_, err = w.Render()
			if err != nil {
				t.Errorf("Render() with dynamic_scaling=%f error = %v", tt.dynamicScaling, err)
			}
		})
	}
}

func TestWidget_GetMethods(t *testing.T) {

	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_getters",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X:      10,
			Y:      20,
			W:      128,
			H:      40,
			ZOrder: 5,
		},
		Style: config.StyleConfig{
			Background: 100,
			Border:     200,
		},
		Mode:           "spectrum",
		UpdateInterval: 0.05,
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test Name()
	if w.Name() != "test_getters" {
		t.Errorf("Name() = %s, want test_getters", w.Name())
	}

	// Test GetPosition()
	pos := w.GetPosition()
	if pos.X != 10 || pos.Y != 20 || pos.W != 128 || pos.H != 40 || pos.ZOrder != 5 {
		t.Errorf("GetPosition() = %+v, want {X:10 Y:20 W:128 H:40 ZOrder:5}", pos)
	}

	// Test GetStyle()
	style := w.GetStyle()
	if style.Background != 100 || style.Border != 200 {
		t.Errorf("GetStyle() = %+v, want {Background:100 Border:200}", style)
	}

	// Test GetUpdateInterval()
	interval := w.GetUpdateInterval()
	expectedInterval := int64(0.05 * 1e9) // 50ms in nanoseconds
	if interval.Nanoseconds() != expectedInterval {
		t.Errorf("GetUpdateInterval() = %v, want 50ms", interval)
	}
}

func TestWidget_ConcurrentAccess(t *testing.T) {

	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_concurrent",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			Background: 0,
		},
		Mode:           "spectrum",
		UpdateInterval: 0.033,
		Spectrum: &config.SpectrumConfig{
			Bars: 32,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Number of concurrent goroutines
	const numUpdaters = 10
	const numRenderers = 10
	const numIterations = 20

	done := make(chan bool, numUpdaters+numRenderers)
	errors := make(chan error, (numUpdaters+numRenderers)*numIterations)

	// Launch updater goroutines
	for i := 0; i < numUpdaters; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < numIterations; j++ {
				// Update might fail in test environment without audio device
				// but shouldn't panic or cause data races
				_ = w.Update()
			}
		}(i)
	}

	// Launch renderer goroutines
	for i := 0; i < numRenderers; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < numIterations; j++ {
				_, err := w.Render()
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numUpdaters+numRenderers; i++ {
		<-done
	}
	close(errors)

	// Check for any errors during rendering (Update errors are acceptable in test env)
	var errCount int
	for err := range errors {
		t.Errorf("Error during concurrent rendering: %v", err)
		errCount++
		if errCount > 5 {
			t.Log("(truncating error list...)")
			break
		}
	}

	t.Log("Concurrent access test completed. Run with -race flag to detect data races.")
}

func TestWidget_BorderRendering(t *testing.T) {

	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_border",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			Background: 0,
			Border:     255,
		},
		Mode: "spectrum",
		Spectrum: &config.SpectrumConfig{
			Bars: 16,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	img, err := w.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	grayImg := img.(*image.Gray)

	// Check that border pixels are non-zero (white border on black background)
	// Top edge
	hasBorderPixels := false
	for x := 0; x < 128; x++ {
		if grayImg.GrayAt(x, 0).Y > 0 {
			hasBorderPixels = true
			break
		}
	}

	if !hasBorderPixels {
		t.Error("Border enabled but no border pixels found on top edge")
	}
}

func TestWidget_FrequencyCompensationToggle(t *testing.T) {

	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "frequency compensation enabled",
			enabled: true,
		},
		{
			name:    "frequency compensation disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "audio_visualizer",
				ID:      "test_freq_comp",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 128,
					H: 40,
				},
				Style: config.StyleConfig{
					Background: 0,
				},
				Mode: "spectrum",
				Spectrum: &config.SpectrumConfig{
					Bars:                  16,
					FrequencyCompensation: tt.enabled,
				},
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			avWidget := w.(*Widget)
			if avWidget.frequencyCompensation != tt.enabled {
				t.Errorf("frequencyCompensation = %v, want %v", avWidget.frequencyCompensation, tt.enabled)
			}

			// Should render without error
			_, err = w.Render()
			if err != nil {
				t.Errorf("Render() error = %v", err)
			}
		})
	}
}

// TestGetSharedAudioCapture_Singleton verifies singleton behavior of audio capture
func TestGetSharedAudioCapture_Singleton(t *testing.T) {
	// Get shared instance twice
	capture1, err1 := GetSharedAudioCapture()
	if err1 != nil {
		t.Fatalf("First GetSharedAudioCapture() failed: %v", err1)
	}

	capture2, err2 := GetSharedAudioCapture()
	if err2 != nil {
		t.Fatalf("Second GetSharedAudioCapture() failed: %v", err2)
	}

	// Should return the same instance
	if capture1 != capture2 {
		t.Error("GetSharedAudioCapture() returned different instances (expected singleton)")
	}

	// Verify it's initialized
	if !capture1.initialized {
		t.Error("Shared audio capture not initialized")
	}
}

// TestGetSharedAudioCapture_ConcurrentCreation tests thread-safe singleton creation
func TestGetSharedAudioCapture_ConcurrentCreation(t *testing.T) {
	// Start 100 goroutines trying to get the shared instance concurrently
	done := make(chan *AudioCaptureWCA, 100)
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		go func() {
			capture, err := GetSharedAudioCapture()
			if err != nil {
				errors <- err
				return
			}
			done <- capture
		}()
	}

	// Collect all results
	var captures []*AudioCaptureWCA
	for i := 0; i < 100; i++ {
		select {
		case capture := <-done:
			captures = append(captures, capture)
		case err := <-errors:
			t.Fatalf("Error getting shared audio capture: %v", err)
		}
	}

	// All captures should be the same instance
	firstCapture := captures[0]
	for i, capture := range captures {
		if capture != firstCapture {
			t.Errorf("Capture %d is different from first capture (expected all same)", i)
		}
	}
}
