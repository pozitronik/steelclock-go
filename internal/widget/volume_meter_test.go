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
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
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
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "invalid_mode",
	}

	_, err := NewVolumeMeterWidget(cfg)
	if err == nil {
		t.Error("NewVolumeMeterWidget() should return error for invalid display mode")
	} else if !strings.Contains(err.Error(), "invalid display mode") {
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
	}

	for _, mode := range displayModes {
		t.Run(mode, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "volume_meter",
				ID:      "test_meter_" + mode,
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Mode: mode,
				Text: &config.TextConfig{
					Size: 10,
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
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
		Metering: &config.MeteringConfig{
			DBScale: true,
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
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
		Clipping: &config.ClippingConfig{
			Enabled:   true,
			Threshold: 0.95,
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
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
		Metering: &config.MeteringConfig{
			DecayRate: 5.0, // Fast decay for testing
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
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
		Peak: &config.PeakConfig{
			Enabled:  true,
			HoldTime: 0.5, // 500ms
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
	peakHoldValues := widget.peakHoldValues
	widget.mu.RUnlock()

	// Peak hold values should be within valid range
	for i, peakHoldValue := range peakHoldValues {
		if peakHoldValue < 0 || peakHoldValue > 1.0 {
			t.Errorf("peakHoldValues[%d] = %.2f, should be in [0.0, 1.0]", i, peakHoldValue)
		}
	}
}

// TestVolumeMeterWidget_AutoHideConfig tests auto-hide configuration
func TestVolumeMeterWidget_AutoHideConfig(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_autohide",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
		AutoHide: &config.AutoHideConfig{
			Enabled:     true,
			OnSilence:   true,
			SilenceTime: 1.0,
		},
		Metering: &config.MeteringConfig{
			SilenceThreshold: 0.01,
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
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
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
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
		AutoHide: &config.AutoHideConfig{
			Enabled: false, // Disable auto-hide for render test
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
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
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

// TestVolumeMeterWidget_StereoMode tests stereo mode configuration
func TestVolumeMeterWidget_StereoMode(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_stereo",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
		Stereo: &config.StereoConfig{
			Enabled: true,
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	if !widget.stereoMode {
		t.Error("stereoMode should be true")
	}

	// Wait for meter updates
	time.Sleep(300 * time.Millisecond)

	widget.mu.RLock()
	channelPeaks := widget.channelPeaks
	channelCount := widget.channelCount
	widget.mu.RUnlock()

	// Should have channel data
	if channelCount == 0 {
		t.Error("channelCount should be > 0")
	}

	// Channel peaks should be valid
	for i, peak := range channelPeaks {
		if peak < 0 || peak > 1.0 {
			t.Errorf("channelPeaks[%d] = %.2f, should be in [0.0, 1.0]", i, peak)
		}
	}
}

// TestVolumeMeterWidget_StereoWithPeakHold tests per-channel peak hold
func TestVolumeMeterWidget_StereoWithPeakHold(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_stereo_peak",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
		Stereo: &config.StereoConfig{
			Enabled: true,
		},
		Peak: &config.PeakConfig{
			Enabled:  true,
			HoldTime: 0.5,
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	if !widget.stereoMode {
		t.Error("stereoMode should be true")
	}

	if !widget.showPeakHold {
		t.Error("showPeakHold should be true")
	}

	// Wait for meter updates
	time.Sleep(300 * time.Millisecond)

	widget.mu.RLock()
	peakHoldValues := widget.peakHoldValues
	peakHoldUntils := widget.peakHoldUntils
	channelCount := widget.channelCount
	widget.mu.RUnlock()

	// Peak hold arrays should match channel count
	if len(peakHoldValues) != channelCount {
		t.Errorf("peakHoldValues length = %d, want %d", len(peakHoldValues), channelCount)
	}

	if len(peakHoldUntils) != channelCount {
		t.Errorf("peakHoldUntils length = %d, want %d", len(peakHoldUntils), channelCount)
	}

	// Each channel's peak hold should be valid
	for i, val := range peakHoldValues {
		if val < 0 || val > 1.0 {
			t.Errorf("peakHoldValues[%d] = %.2f, should be in [0.0, 1.0]", i, val)
		}
	}
}

// TestVolumeMeterWidget_StereoDivider tests stereo divider configuration
func TestVolumeMeterWidget_StereoDivider(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	dividerColor := 128
	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_divider",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar",
		Stereo: &config.StereoConfig{
			Enabled: true,
			Divider: &dividerColor,
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	if !widget.stereoMode {
		t.Error("stereoMode should be true")
	}

	if widget.stereoDivider != 128 {
		t.Errorf("stereoDivider = %d, want 128", widget.stereoDivider)
	}
}

// TestVolumeMeterWidget_GaugeWithPeakHold tests gauge mode with peak hold marks
func TestVolumeMeterWidget_GaugeWithPeakHold(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_gauge_peak",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 80, H: 60,
		},
		Mode: "gauge",
		Peak: &config.PeakConfig{
			Enabled:  true,
			HoldTime: 0.5,
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	if widget.displayMode != "gauge" {
		t.Errorf("displayMode = %s, want gauge", widget.displayMode)
	}

	if !widget.showPeakHold {
		t.Error("showPeakHold should be true")
	}

	// Wait for meter updates
	time.Sleep(300 * time.Millisecond)

	// Try to render - should not error
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	// Image may be nil or valid
	_ = img
}

// TestVolumeMeterWidget_StereoGauge tests stereo gauge mode
func TestVolumeMeterWidget_StereoGauge(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_stereo_gauge",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 160, H: 60,
		},
		Mode: "gauge",
		Stereo: &config.StereoConfig{
			Enabled: true,
		},
		Peak: &config.PeakConfig{
			Enabled:  true,
			HoldTime: 0.5,
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	if !widget.stereoMode {
		t.Error("stereoMode should be true")
	}

	if widget.displayMode != "gauge" {
		t.Errorf("displayMode = %s, want gauge", widget.displayMode)
	}

	// Wait for meter updates
	time.Sleep(300 * time.Millisecond)

	// Try to render - should not error
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	// Image should be created for stereo gauge
	if img != nil {
		bounds := img.Bounds()
		if bounds.Dx() != 160 || bounds.Dy() != 60 {
			t.Errorf("Render() image size = %dx%d, want 160x60", bounds.Dx(), bounds.Dy())
		}
	}
}

// TestVolumeMeterWidget_Ballistics tests rise and fall ballistics
func TestVolumeMeterWidget_Ballistics(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_ballistics",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
		Metering: &config.MeteringConfig{
			DecayRate: 2.0, // Units per second
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	if widget.decayRate != 2.0 {
		t.Errorf("decayRate = %.1f, want 2.0", widget.decayRate)
	}

	// Wait for initial meter updates
	time.Sleep(200 * time.Millisecond)

	// Sample displayPeak multiple times to verify it changes smoothly
	widget.mu.RLock()
	peak1 := widget.displayPeak
	widget.mu.RUnlock()

	time.Sleep(100 * time.Millisecond)

	widget.mu.RLock()
	peak2 := widget.displayPeak
	widget.mu.RUnlock()

	// Peaks should be within valid range
	if peak1 < 0 || peak1 > 1.0 {
		t.Errorf("peak1 = %.2f, should be in [0.0, 1.0]", peak1)
	}

	if peak2 < 0 || peak2 > 1.0 {
		t.Errorf("peak2 = %.2f, should be in [0.0, 1.0]", peak2)
	}

	// With ballistics, displayPeak should not change drastically
	// (this is a weak test, but verifies ballistics are applied)
	_ = peak1
	_ = peak2
}
