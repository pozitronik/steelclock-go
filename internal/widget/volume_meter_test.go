package widget

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// skipIfNoAudioDeviceMeter skips the test if audio device is not available
func skipIfNoAudioDeviceMeter(t *testing.T) {
	t.Helper()

	// Volume meter widget is Windows-only
	if runtime.GOOS != "windows" {
		t.Skip("Volume meter widget is Windows-only (requires Windows Core Audio API)")
		return
	}

	// Try to create a meter reader to see if audio devices are available
	reader, err := newMeterReader()
	if err != nil {
		if strings.Contains(err.Error(), "Element not found") {
			t.Skip("No audio device available (common in CI environments)")
		}
		t.Skipf("Cannot initialize audio: %v", err)
	}

	if reader != nil {
		reader.Close()
	}
}

// TestNewVolumeMeterWidget tests widget creation
func TestNewVolumeMeterWidget(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}

	if widget == nil {
		t.Fatal("NewVolumeMeterWidget() returned nil widget")
	}

	// Cleanup
	widget.Stop()
}

// TestVolumeMeterWidget_InvalidDisplayMode tests error handling for invalid display modes
func TestVolumeMeterWidget_InvalidDisplayMode(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "invalid_mode",
		},
	}

	_, err := NewVolumeMeterWidget(cfg)
	if err == nil {
		t.Error("NewVolumeMeterWidget() should return error for invalid display mode")
	}

	if !strings.Contains(err.Error(), "invalid display mode") {
		t.Errorf("Error should mention invalid display mode, got: %v", err)
	}
}

// TestVolumeMeterWidget_AllDisplayModes tests all valid display modes
func TestVolumeMeterWidget_AllDisplayModes(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	displayModes := []string{
		"text",
		"bar_horizontal",
		"bar_vertical",
		"gauge",
		"vu_meter",
		"stereo_bars",
		"stereo_vu",
	}

	for _, mode := range displayModes {
		t.Run(mode, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "volume_meter",
				ID:      "test_meter_" + mode,
				Enabled: true,
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Properties: config.WidgetProperties{
					DisplayMode: mode,
					FontSize:    10,
				},
			}

			widget, err := NewVolumeMeterWidget(cfg)
			if err != nil {
				t.Fatalf("NewVolumeMeterWidget(%s) error = %v", mode, err)
			}

			if widget == nil {
				t.Fatalf("NewVolumeMeterWidget(%s) returned nil", mode)
			}

			// Let it run briefly
			time.Sleep(200 * time.Millisecond)

			// Try to render
			img, err := widget.Render()
			if err != nil {
				t.Errorf("Render(%s) error = %v", mode, err)
			}

			// Image may be nil if auto-hide is active
			_ = img

			widget.Stop()
		})
	}
}

// TestVolumeMeterWidget_DBScale tests dB scale conversion
func TestVolumeMeterWidget_DBScale(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_db",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
			UseDBScale:  true,
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	if !widget.useDBScale {
		t.Error("useDBScale should be true")
	}

	// Test dB conversion
	testCases := []struct {
		linear   float64
		expected float64 // approximate
	}{
		{1.0, 1.0},    // 0 dB = 1.0 normalized
		{0.5, 0.895},  // ~-6 dB
		{0.1, 0.667},  // ~-20 dB
		{0.01, 0.333}, // ~-40 dB
		{0.0, 0.0},    // -inf dB = 0.0
	}

	for _, tc := range testCases {
		result := widget.linearToDBNormalized(tc.linear)
		// Allow 10% tolerance for float comparison
		tolerance := 0.1
		if result < tc.expected-tolerance || result > tc.expected+tolerance {
			t.Errorf("linearToDBNormalized(%.2f) = %.3f, want ~%.3f", tc.linear, result, tc.expected)
		}
	}
}

// TestVolumeMeterWidget_ClippingDetection tests clipping detection
func TestVolumeMeterWidget_ClippingDetection(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_clip",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:       "bar_horizontal",
			ShowClipping:      true,
			ClippingThreshold: 0.95,
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	if !widget.showClipping {
		t.Error("showClipping should be true")
	}

	if widget.clippingThreshold != 0.95 {
		t.Errorf("clippingThreshold = %.2f, want 0.95", widget.clippingThreshold)
	}
}

// TestVolumeMeterWidget_DecayBehavior tests peak decay
func TestVolumeMeterWidget_DecayBehavior(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_decay",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "vu_meter",
			DecayRate:   5.0, // Fast decay for testing
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	if widget.decayRate != 5.0 {
		t.Errorf("decayRate = %.1f, want 5.0", widget.decayRate)
	}

	// Wait for some meter updates
	time.Sleep(300 * time.Millisecond)

	widget.mu.RLock()
	displayPeak := widget.displayPeak
	widget.mu.RUnlock()

	// Display peak should be within valid range
	if displayPeak < 0 || displayPeak > 1.0 {
		t.Errorf("displayPeak = %.2f, should be in [0.0, 1.0]", displayPeak)
	}
}

// TestVolumeMeterWidget_PeakHold tests peak hold functionality
func TestVolumeMeterWidget_PeakHold(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_peak_hold",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:  "vu_meter",
			ShowPeakHold: true,
			PeakHoldTime: 0.5, // 500ms
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	if !widget.showPeakHold {
		t.Error("showPeakHold should be true")
	}

	if widget.peakHoldTime != 500*time.Millisecond {
		t.Errorf("peakHoldTime = %v, want 500ms", widget.peakHoldTime)
	}

	// Wait for meter updates
	time.Sleep(300 * time.Millisecond)

	widget.mu.RLock()
	peakHoldValue := widget.peakHoldValue
	widget.mu.RUnlock()

	// Peak hold value should be within valid range
	if peakHoldValue < 0 || peakHoldValue > 1.0 {
		t.Errorf("peakHoldValue = %.2f, should be in [0.0, 1.0]", peakHoldValue)
	}
}

// TestVolumeMeterWidget_AutoHideConfig tests auto-hide configuration
func TestVolumeMeterWidget_AutoHideConfig(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_autohide",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:         "bar_horizontal",
			AutoHide:            true,
			AutoHideOnSilence:   true,
			AutoHideSilenceTime: 1.0,
			SilenceThreshold:    0.01,
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	if !widget.autoHideOnSilence {
		t.Error("autoHideOnSilence should be true")
	}

	if widget.silenceThreshold != 0.01 {
		t.Errorf("silenceThreshold = %.2f, want 0.01", widget.silenceThreshold)
	}
}

// TestVolumeMeterWidget_Stop tests proper cleanup
func TestVolumeMeterWidget_Stop(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_stop",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}

	// Let it run briefly
	time.Sleep(200 * time.Millisecond)

	widget.mu.RLock()
	totalCallsBefore := widget.totalCalls
	widget.mu.RUnlock()

	// Stop the widget
	widget.Stop()

	// Reader should be cleaned up
	if widget.reader != nil {
		t.Error("Reader should be cleaned up after Stop()")
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// No new calls should be made after Stop()
	widget.mu.RLock()
	totalCallsAfter := widget.totalCalls
	widget.mu.RUnlock()

	if totalCallsAfter != totalCallsBefore {
		t.Errorf("Calls should stop after Stop(), before=%d, after=%d", totalCallsBefore, totalCallsAfter)
	}
}

// TestVolumeMeterWidget_Render tests rendering
func TestVolumeMeterWidget_Render(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_render",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
			AutoHide:    false, // Disable auto-hide for render test
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	// Wait for meter data
	time.Sleep(200 * time.Millisecond)

	// Render should not error
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	// Image may be nil or valid
	if img != nil {
		bounds := img.Bounds()
		if bounds.Dx() != 128 || bounds.Dy() != 40 {
			t.Errorf("Render() image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
		}
	}
}

// TestVolumeMeterWidget_HealthMetrics tests diagnostic metrics
func TestVolumeMeterWidget_HealthMetrics(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_health",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	// Let it accumulate some calls
	time.Sleep(500 * time.Millisecond)

	widget.mu.RLock()
	totalCalls := widget.totalCalls
	successfulCalls := widget.successfulCalls
	failedCalls := widget.failedCalls
	lastSuccessTime := widget.lastSuccessTime
	maxCallDuration := widget.maxCallDuration
	widget.mu.RUnlock()

	// Should have metrics
	if totalCalls == 0 {
		t.Error("No calls recorded")
	}

	if successfulCalls == 0 {
		t.Error("No successful calls recorded")
	}

	// Failed calls should be zero or very low
	if failedCalls > totalCalls/10 {
		t.Errorf("Too many failed calls: %d out of %d", failedCalls, totalCalls)
	}

	// Last success should be recent
	if time.Since(lastSuccessTime) > 1*time.Second {
		t.Errorf("Last success too old: %v ago", time.Since(lastSuccessTime))
	}

	// Max duration should be reasonable
	if maxCallDuration > 100*time.Millisecond {
		t.Logf("Warning: Max call duration high: %v", maxCallDuration)
	}

	t.Logf("Health metrics: total=%d, success=%d, failed=%d, maxDuration=%v",
		totalCalls, successfulCalls, failedCalls, maxCallDuration)
}
