//go:build windows && integration

// Audio Visualizer Integration Tests
//
// These tests require:
// - Windows OS with audio hardware
// - COM initialization support
// - Run sequentially (not in parallel) due to COM threading model
//
// To run these tests:
//   go test -tags=integration ./internal/widget -run TestAudioVisualizer
//
// These tests are NOT run by default because:
// 1. They require real audio hardware
// 2. The shared singleton pattern used by AudioVisualizerWidget doesn't work well in parallel test execution
// 3. COM initialization is thread-specific and conflicts in test environments
// 4. They fail in CI environments without audio devices

package widget

import (
	"image"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewAudioVisualizerWidget_Spectrum(t *testing.T) {

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
			BackgroundColor: 0,
			Border:          false,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			DisplayMode:            "spectrum",
			UpdateInterval:         0.033,
			BarCount:               32,
			FrequencyScale:         "logarithmic",
			BarStyle:               "bars",
			Smoothing:              0.7,
			PeakHold:               true,
			PeakHoldTime:           1.0,
			FillColor:              255,
			FrequencyCompensation:  true,
			SpectrumDynamicScaling: 1.0,
		},
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	if widget == nil {
		t.Fatal("NewAudioVisualizerWidget() returned nil")
	}

	if widget.Name() != "test_spectrum" {
		t.Errorf("Name() = %s, want test_spectrum", widget.Name())
	}

	avWidget := widget.(*AudioVisualizerWidget)
	if avWidget.properties.DisplayMode != "spectrum" {
		t.Errorf("DisplayMode = %s, want spectrum", avWidget.properties.DisplayMode)
	}

	if avWidget.properties.BarCount != 32 {
		t.Errorf("BarCount = %d, want 32", avWidget.properties.BarCount)
	}
}

func TestNewAudioVisualizerWidget_Oscilloscope(t *testing.T) {

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
			BackgroundColor: 0,
			Border:          false,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			DisplayMode:       "oscilloscope",
			UpdateInterval:    0.033,
			WaveformStyle:     "line",
			ChannelMode:       "stereo_separated",
			SampleCount:       128,
			LeftChannelColor:  255,
			RightChannelColor: 200,
		},
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	if widget == nil {
		t.Fatal("NewAudioVisualizerWidget() returned nil")
	}

	avWidget := widget.(*AudioVisualizerWidget)
	if avWidget.properties.DisplayMode != "oscilloscope" {
		t.Errorf("DisplayMode = %s, want oscilloscope", avWidget.properties.DisplayMode)
	}

	if avWidget.properties.WaveformStyle != "line" {
		t.Errorf("WaveformStyle = %s, want line", avWidget.properties.WaveformStyle)
	}

	if avWidget.properties.ChannelMode != "stereo_separated" {
		t.Errorf("ChannelMode = %s, want stereo_separated", avWidget.properties.ChannelMode)
	}
}

func TestNewAudioVisualizerWidget_DefaultBarCount(t *testing.T) {

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
			BackgroundColor: 0,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "spectrum",
			// BarCount not specified - should default to 32
		},
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	avWidget := widget.(*AudioVisualizerWidget)

	// Check that barCount was set to default (32)
	if len(avWidget.spectrumData) != 32 {
		t.Errorf("Default barCount spectrumData length = %d, want 32", len(avWidget.spectrumData))
	}

	if len(avWidget.peakValues) != 32 {
		t.Errorf("Default barCount peakValues length = %d, want 32", len(avWidget.peakValues))
	}
}

func TestNewAudioVisualizerWidget_DynamicScalingWindow(t *testing.T) {

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
			wantWindowSize: 16, // 0.5 / 0.033 ≈ 15.15, +1 = 16
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
					BackgroundColor: 0,
				},
				Properties: config.WidgetProperties{
					DisplayMode:                  "spectrum",
					UpdateInterval:               tt.updateInterval,
					BarCount:                     10,
					SpectrumDynamicScalingWindow: tt.scalingWindow,
				},
			}

			widget, err := NewAudioVisualizerWidget(cfg)
			if err != nil {
				t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
			}

			avWidget := widget.(*AudioVisualizerWidget)
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

func TestAudioVisualizerWidget_Render_Spectrum(t *testing.T) {

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
			BackgroundColor: 0,
			Border:          false,
		},
		Properties: config.WidgetProperties{
			DisplayMode:    "spectrum",
			UpdateInterval: 0.033,
			BarCount:       32,
			FillColor:      255,
		},
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	// Render without update (should work with zero data)
	img, err := widget.Render()
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

func TestAudioVisualizerWidget_Render_Oscilloscope(t *testing.T) {

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
			BackgroundColor: 0,
			Border:          false,
		},
		Properties: config.WidgetProperties{
			DisplayMode:       "oscilloscope",
			UpdateInterval:    0.033,
			WaveformStyle:     "line",
			ChannelMode:       "stereo_combined",
			SampleCount:       128,
			LeftChannelColor:  255,
			RightChannelColor: 200,
		},
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	// Render without update (should work with zero data)
	img, err := widget.Render()
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

func TestAudioVisualizerWidget_BarStyles(t *testing.T) {

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
					BackgroundColor: 0,
				},
				Properties: config.WidgetProperties{
					DisplayMode: "spectrum",
					BarCount:    16,
					BarStyle:    style,
					FillColor:   255,
				},
			}

			widget, err := NewAudioVisualizerWidget(cfg)
			if err != nil {
				t.Fatalf("NewAudioVisualizerWidget() with %s error = %v", style, err)
			}

			// Should render without error
			img, err := widget.Render()
			if err != nil {
				t.Errorf("Render() with %s error = %v", style, err)
			}

			if img == nil {
				t.Errorf("Render() with %s returned nil image", style)
			}

			avWidget := widget.(*AudioVisualizerWidget)
			if avWidget.properties.BarStyle != style {
				t.Errorf("BarStyle = %s, want %s", avWidget.properties.BarStyle, style)
			}
		})
	}
}

func TestAudioVisualizerWidget_FrequencyScales(t *testing.T) {

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
					BackgroundColor: 0,
				},
				Properties: config.WidgetProperties{
					DisplayMode:    "spectrum",
					BarCount:       16,
					FrequencyScale: scale,
					FillColor:      255,
				},
			}

			widget, err := NewAudioVisualizerWidget(cfg)
			if err != nil {
				t.Fatalf("NewAudioVisualizerWidget() with %s error = %v", scale, err)
			}

			// Should render without error
			img, err := widget.Render()
			if err != nil {
				t.Errorf("Render() with %s error = %v", scale, err)
			}

			if img == nil {
				t.Errorf("Render() with %s returned nil image", scale)
			}

			avWidget := widget.(*AudioVisualizerWidget)
			if avWidget.properties.FrequencyScale != scale {
				t.Errorf("FrequencyScale = %s, want %s", avWidget.properties.FrequencyScale, scale)
			}
		})
	}
}

func TestAudioVisualizerWidget_WaveformStyles(t *testing.T) {

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
					BackgroundColor: 0,
				},
				Properties: config.WidgetProperties{
					DisplayMode:   "oscilloscope",
					WaveformStyle: style,
					SampleCount:   128,
					FillColor:     255,
				},
			}

			widget, err := NewAudioVisualizerWidget(cfg)
			if err != nil {
				t.Fatalf("NewAudioVisualizerWidget() with %s error = %v", style, err)
			}

			// Should render without error
			img, err := widget.Render()
			if err != nil {
				t.Errorf("Render() with %s error = %v", style, err)
			}

			if img == nil {
				t.Errorf("Render() with %s returned nil image", style)
			}

			avWidget := widget.(*AudioVisualizerWidget)
			if avWidget.properties.WaveformStyle != style {
				t.Errorf("WaveformStyle = %s, want %s", avWidget.properties.WaveformStyle, style)
			}
		})
	}
}

func TestAudioVisualizerWidget_ChannelModes(t *testing.T) {

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
					BackgroundColor: 0,
				},
				Properties: config.WidgetProperties{
					DisplayMode:       "oscilloscope",
					ChannelMode:       mode,
					SampleCount:       128,
					LeftChannelColor:  255,
					RightChannelColor: 200,
				},
			}

			widget, err := NewAudioVisualizerWidget(cfg)
			if err != nil {
				t.Fatalf("NewAudioVisualizerWidget() with %s error = %v", mode, err)
			}

			// Should render without error
			img, err := widget.Render()
			if err != nil {
				t.Errorf("Render() with %s error = %v", mode, err)
			}

			if img == nil {
				t.Errorf("Render() with %s returned nil image", mode)
			}

			avWidget := widget.(*AudioVisualizerWidget)
			if avWidget.properties.ChannelMode != mode {
				t.Errorf("ChannelMode = %s, want %s", avWidget.properties.ChannelMode, mode)
			}
		})
	}
}

func TestAudioVisualizerWidget_DynamicScaling(t *testing.T) {

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
					BackgroundColor: 0,
				},
				Properties: config.WidgetProperties{
					DisplayMode:            "spectrum",
					BarCount:               16,
					SpectrumDynamicScaling: tt.dynamicScaling,
				},
			}

			widget, err := NewAudioVisualizerWidget(cfg)
			if err != nil {
				t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
			}

			avWidget := widget.(*AudioVisualizerWidget)
			if avWidget.properties.SpectrumDynamicScaling != tt.dynamicScaling {
				t.Errorf("SpectrumDynamicScaling = %f, want %f", avWidget.properties.SpectrumDynamicScaling, tt.dynamicScaling)
			}

			// Render should work regardless of dynamic scaling setting
			_, err = widget.Render()
			if err != nil {
				t.Errorf("Render() with dynamic_scaling=%f error = %v", tt.dynamicScaling, err)
			}
		})
	}
}

func TestAudioVisualizerWidget_GetMethods(t *testing.T) {

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
			BackgroundColor: 100,
			Border:          true,
			BorderColor:     200,
		},
		Properties: config.WidgetProperties{
			DisplayMode:    "spectrum",
			UpdateInterval: 0.05,
		},
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	// Test Name()
	if widget.Name() != "test_getters" {
		t.Errorf("Name() = %s, want test_getters", widget.Name())
	}

	// Test GetPosition()
	pos := widget.GetPosition()
	if pos.X != 10 || pos.Y != 20 || pos.W != 128 || pos.H != 40 || pos.ZOrder != 5 {
		t.Errorf("GetPosition() = %+v, want {X:10 Y:20 W:128 H:40 ZOrder:5}", pos)
	}

	// Test GetStyle()
	style := widget.GetStyle()
	if style.BackgroundColor != 100 || !style.Border || style.BorderColor != 200 {
		t.Errorf("GetStyle() = %+v, want {BackgroundColor:100 Border:true BorderColor:200}", style)
	}

	// Test GetUpdateInterval()
	interval := widget.GetUpdateInterval()
	expectedInterval := int64(0.05 * 1e9) // 50ms in nanoseconds
	if interval.Nanoseconds() != expectedInterval {
		t.Errorf("GetUpdateInterval() = %v, want 50ms", interval)
	}
}

func TestAudioVisualizerWidget_ConcurrentAccess(t *testing.T) {

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
			BackgroundColor: 0,
		},
		Properties: config.WidgetProperties{
			DisplayMode:    "spectrum",
			UpdateInterval: 0.033,
			BarCount:       32,
		},
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
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
				_ = widget.Update()
			}
		}(i)
	}

	// Launch renderer goroutines
	for i := 0; i < numRenderers; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < numIterations; j++ {
				_, err := widget.Render()
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

func TestAudioVisualizerWidget_BorderRendering(t *testing.T) {

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
			BackgroundColor: 0,
			Border:          true,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "spectrum",
			BarCount:    16,
		},
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	img, err := widget.Render()
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

func TestAudioVisualizerWidget_FrequencyCompensationToggle(t *testing.T) {

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
					BackgroundColor: 0,
				},
				Properties: config.WidgetProperties{
					DisplayMode:           "spectrum",
					BarCount:              16,
					FrequencyCompensation: tt.enabled,
				},
			}

			widget, err := NewAudioVisualizerWidget(cfg)
			if err != nil {
				t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
			}

			avWidget := widget.(*AudioVisualizerWidget)
			if avWidget.properties.FrequencyCompensation != tt.enabled {
				t.Errorf("FrequencyCompensation = %v, want %v", avWidget.properties.FrequencyCompensation, tt.enabled)
			}

			// Should render without error
			_, err = widget.Render()
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
