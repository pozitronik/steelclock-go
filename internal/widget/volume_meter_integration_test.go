//go:build windows

package widget

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// TestVolumeMeterWidget_BackgroundPolling tests that the background goroutine works
func TestVolumeMeterWidget_BackgroundPolling(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_polling",
		Enabled: config.BoolPtr(true),
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

	// Give the background goroutine time to initialize and read meter
	time.Sleep(200 * time.Millisecond)

	// Check that meter data was populated by background goroutine
	widget.mu.RLock()
	peak := widget.peak
	successCount := widget.successfulCalls
	channelCount := widget.channelCount
	widget.mu.RUnlock()

	if successCount == 0 {
		t.Error("Background goroutine did not make any successful meter calls")
	}

	if peak < 0 || peak > 1.0 {
		t.Errorf("Peak out of range: %.2f", peak)
	}

	if channelCount < 0 {
		t.Errorf("Invalid channel count: %d", channelCount)
	}

	t.Logf("Background polling working: %d successful calls, peak=%.2f, channels=%d",
		successCount, peak, channelCount)

	// Stop the widget
	widget.Stop()

	// Verify it stopped
	time.Sleep(100 * time.Millisecond)
}

// TestVolumeMeterWidget_StopCleanup tests proper cleanup on Stop
func TestVolumeMeterWidget_StopCleanup(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_cleanup",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "gauge",
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	// Stop should clean up COM resources
	widget.Stop()

	// Reader should be nil after cleanup
	if widget.reader != nil {
		t.Error("Meter reader not cleaned up after Stop()")
	}
}

// TestVolumeMeterWidget_LongRunning tests stability over extended period
func TestVolumeMeterWidget_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_longrun",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "vu_meter",
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	// Run for 10 seconds
	duration := 10 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < duration {
		time.Sleep(1 * time.Second)

		widget.mu.RLock()
		successRate := float64(0)
		if widget.totalCalls > 0 {
			successRate = float64(widget.successfulCalls) / float64(widget.totalCalls) * 100
		}
		consecutiveErrors := widget.consecutiveErrors
		widget.mu.RUnlock()

		// Success rate should remain high
		if successRate < 95.0 && widget.totalCalls > 10 {
			t.Errorf("Success rate dropped to %.2f%% after %v", successRate, time.Since(startTime))
		}

		// Should not have many consecutive errors
		if consecutiveErrors > 5 {
			t.Errorf("Too many consecutive errors: %d after %v", consecutiveErrors, time.Since(startTime))
		}
	}

	// Final check
	widget.mu.RLock()
	totalCalls := widget.totalCalls
	successfulCalls := widget.successfulCalls
	failedCalls := widget.failedCalls
	maxDuration := widget.maxCallDuration
	widget.mu.RUnlock()

	t.Logf("Long-running test results:")
	t.Logf("  Total calls: %d", totalCalls)
	t.Logf("  Successful: %d", successfulCalls)
	t.Logf("  Failed: %d", failedCalls)
	t.Logf("  Success rate: %.2f%%", float64(successfulCalls)/float64(totalCalls)*100)
	t.Logf("  Max call duration: %v", maxDuration)

	// Should have made many calls in 10 seconds (at 100ms interval = ~100 calls)
	if totalCalls < 50 {
		t.Errorf("Too few calls made: %d (expected ~100 in 10s)", totalCalls)
	}

	// Success rate should be very high
	if float64(successfulCalls)/float64(totalCalls)*100 < 99.0 {
		t.Errorf("Success rate too low: %.2f%%", float64(successfulCalls)/float64(totalCalls)*100)
	}
}

// TestVolumeMeterWidget_StereoChannels tests stereo channel handling
func TestVolumeMeterWidget_StereoChannels(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_stereo",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "stereo_bars",
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	// Wait for meter data
	time.Sleep(300 * time.Millisecond)

	widget.mu.RLock()
	channelCount := widget.channelCount
	channelPeaks := widget.channelPeaks
	widget.mu.RUnlock()

	t.Logf("Audio device has %d channels", channelCount)

	// Most audio devices have 2 channels (stereo)
	if channelCount > 0 {
		if len(channelPeaks) != channelCount {
			t.Errorf("Channel peaks length = %d, want %d", len(channelPeaks), channelCount)
		}

		// Verify all channel peaks are in valid range
		for i, peak := range channelPeaks {
			if peak < 0 || peak > 1.0 {
				t.Errorf("Channel %d peak = %.2f, should be in [0.0, 1.0]", i, peak)
			}
		}
	}

	// Test rendering stereo mode
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img != nil {
		bounds := img.Bounds()
		t.Logf("Rendered stereo bars: %dx%d", bounds.Dx(), bounds.Dy())
	}
}

// TestVolumeMeterWidget_ClippingIndicator tests clipping detection
func TestVolumeMeterWidget_ClippingIndicator(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_clipping",
		Enabled: config.BoolPtr(true),
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

	// Wait for meter updates
	time.Sleep(500 * time.Millisecond)

	widget.mu.RLock()
	peak := widget.peak
	isClipping := widget.isClipping
	widget.mu.RUnlock()

	t.Logf("Clipping test: peak=%.3f, isClipping=%v (threshold=%.2f)",
		peak, isClipping, widget.clippingThreshold)

	// Clipping detection should work correctly
	if peak >= widget.clippingThreshold && !isClipping {
		t.Error("Clipping should be detected when peak >= threshold")
	}
	if peak < widget.clippingThreshold && isClipping {
		t.Error("Clipping should not be detected when peak < threshold")
	}
}

// TestVolumeMeterWidget_DBScaleDisplay tests dB scale display
func TestVolumeMeterWidget_DBScaleDisplay(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_db_display",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "text",
			UseDBScale:  true,
			FontSize:    10,
		},
	}

	widget, err := NewVolumeMeterWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeMeterWidget() error = %v", err)
	}
	defer widget.Stop()

	// Wait for meter data
	time.Sleep(300 * time.Millisecond)

	widget.mu.RLock()
	peak := widget.peak
	widget.mu.RUnlock()

	t.Logf("dB scale test: linear peak=%.3f", peak)

	// Render in text mode with dB scale
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() with dB scale error = %v", err)
	}

	if img != nil {
		t.Log("Successfully rendered dB scale display")
	}
}

// TestVolumeMeterWidget_NoMemoryLeak tests that repeated creation/destruction doesn't leak
func TestVolumeMeterWidget_NoMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_memleak",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
		},
	}

	// Create and destroy 100 widgets
	for i := 0; i < 100; i++ {
		widget, err := NewVolumeMeterWidget(cfg)
		if err != nil {
			t.Fatalf("NewVolumeMeterWidget() iteration %d error = %v", i, err)
		}

		// Let it run briefly
		time.Sleep(10 * time.Millisecond)

		// Stop and cleanup
		widget.Stop()

		// Brief pause between iterations
		time.Sleep(5 * time.Millisecond)
	}

	// If we got here without crashing or hanging, test passes
	t.Log("Created and destroyed 100 widgets without issues")
}

// TestVolumeMeterWidget_AutoHideOnSilence tests auto-hide when no audio
func TestVolumeMeterWidget_AutoHideOnSilence(t *testing.T) {
	skipIfNoAudioDeviceMeter(t)

	cfg := config.WidgetConfig{
		Type:    "volume_meter",
		ID:      "test_meter_autohide_silence",
		Enabled: config.BoolPtr(true),
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

	// Wait for initial meter reads
	time.Sleep(300 * time.Millisecond)

	widget.mu.RLock()
	hasAudio := widget.hasAudio
	peak := widget.peak
	widget.mu.RUnlock()

	t.Logf("Auto-hide on silence test: hasAudio=%v, peak=%.3f, threshold=%.3f",
		hasAudio, peak, widget.silenceThreshold)

	// If there's audio, widget should be visible (auto-hide triggered)
	if hasAudio {
		img, err := widget.Render()
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		if img == nil {
			t.Error("Widget should be visible when audio is detected")
		} else {
			t.Log("Widget correctly visible when audio detected")
		}

		// Wait for auto-hide timeout
		time.Sleep(1200 * time.Millisecond)

		// Widget should hide if no more audio
		widget.mu.RLock()
		stillHasAudio := widget.hasAudio
		widget.mu.RUnlock()

		if !stillHasAudio {
			img, err = widget.Render()
			if err != nil {
				t.Fatalf("Render() after timeout error = %v", err)
			}

			if img != nil {
				t.Error("Widget should be hidden after silence timeout")
			} else {
				t.Log("Widget correctly hidden after silence timeout")
			}
		}
	} else {
		t.Log("No audio detected - widget should remain hidden")
	}
}
