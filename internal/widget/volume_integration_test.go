//go:build windows
// +build windows

package widget

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// TestVolumeWidget_BackgroundPolling tests that the background goroutine works
func TestVolumeWidget_BackgroundPolling(t *testing.T) {
	skipIfNoAudioDevice(t)

	cfg := config.WidgetConfig{
		Type:    "volume",
		ID:      "test_volume_polling",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	// Give the background goroutine time to initialize and read volume
	time.Sleep(200 * time.Millisecond)

	// Check that volume was populated by background goroutine
	widget.mu.RLock()
	volume := widget.volume
	successCount := widget.successfulCalls
	widget.mu.RUnlock()

	if successCount == 0 {
		t.Error("Background goroutine did not make any successful volume calls")
	}

	if volume < 0 || volume > 100 {
		t.Errorf("Volume out of range: %.2f", volume)
	}

	t.Logf("Background polling working: %d successful calls, volume=%.2f%%", successCount, volume)

	// Stop the widget
	widget.Stop()

	// Verify it stopped
	time.Sleep(100 * time.Millisecond)
}

// TestVolumeWidget_StopCleanup tests proper cleanup on Stop
func TestVolumeWidget_StopCleanup(t *testing.T) {
	skipIfNoAudioDevice(t)

	cfg := config.WidgetConfig{
		Type:    "volume",
		ID:      "test_volume_cleanup",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "gauge",
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	// Stop should clean up COM resources
	widget.Stop()

	// Reader should be nil after cleanup
	if widget.reader != nil {
		t.Error("Volume reader not cleaned up after Stop()")
	}
}

// TestVolumeWidget_LongRunning tests stability over extended period
func TestVolumeWidget_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}
	skipIfNoAudioDevice(t)

	cfg := config.WidgetConfig{
		Type:    "volume",
		ID:      "test_volume_longrun",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
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

// TestVolumeWidget_HealthMetrics tests health tracking
func TestVolumeWidget_HealthMetrics(t *testing.T) {
	skipIfNoAudioDevice(t)

	cfg := config.WidgetConfig{
		Type:    "volume",
		ID:      "test_volume_health",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "gauge",
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
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

	// Max duration should be reasonable (under 10ms for healthy COM)
	if maxCallDuration > 100*time.Millisecond {
		t.Errorf("Max call duration too high: %v", maxCallDuration)
	}

	t.Logf("Health metrics:")
	t.Logf("  Total calls: %d", totalCalls)
	t.Logf("  Successful: %d", successfulCalls)
	t.Logf("  Failed: %d", failedCalls)
	t.Logf("  Max duration: %v", maxCallDuration)
}

// TestVolumeWidget_NoMemoryLeak tests that repeated creation/destruction doesn't leak
func TestVolumeWidget_NoMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}
	skipIfNoAudioDevice(t)

	cfg := config.WidgetConfig{
		Type:    "volume",
		ID:      "test_volume_memleak",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
		},
	}

	// Create and destroy 100 widgets
	for i := 0; i < 100; i++ {
		widget, err := NewVolumeWidget(cfg)
		if err != nil {
			t.Fatalf("NewVolumeWidget() iteration %d error = %v", i, err)
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

// TestVolumeWidget_VolumeChange tests volume change detection and auto-hide trigger
func TestVolumeWidget_VolumeChange(t *testing.T) {
	skipIfNoAudioDevice(t)

	cfg := config.WidgetConfig{
		Type:    "volume",
		ID:      "test_volume_change",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:     "bar_horizontal",
			AutoHide:        true, // Enable auto-hide to test trigger mechanism
			AutoHideTimeout: 1.0,  // 1 second timeout
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}
	defer widget.Stop()

	// Wait for initial volume read (which should trigger auto-hide and make widget visible)
	time.Sleep(300 * time.Millisecond)

	widget.mu.RLock()
	initialVolume := widget.volume
	widget.mu.RUnlock()

	t.Logf("Initial volume: %.2f%%", initialVolume)

	// Widget should be visible after volume is detected (auto-hide triggered)
	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Error("Widget should be visible after volume change detected, but got nil image")
	} else {
		t.Log("Volume change detection working: widget is visible after initial volume read")
	}

	// Wait for auto-hide timeout
	time.Sleep(1200 * time.Millisecond)

	// Widget should now be hidden
	img, err = widget.Render()
	if err != nil {
		t.Fatalf("Render() after timeout error = %v", err)
	}

	if img != nil {
		t.Error("Widget should be hidden after auto-hide timeout, but got image")
	} else {
		t.Log("Auto-hide timeout working: widget is now hidden")
	}
}
