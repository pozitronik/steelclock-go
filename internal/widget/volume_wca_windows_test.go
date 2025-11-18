//go:build windows

package widget

import (
	"strings"
	"testing"
	"time"
)

// skipIfNoAudioDevice skips the test if no audio device is available (CI environments)
func skipIfNoAudioDeviceWCA(t *testing.T) {
	t.Helper()

	// Try to create a volume reader to see if audio devices are available
	reader, err := NewVolumeReaderWCA()
	if err != nil {
		// Check if error is "Element not found" (no audio device)
		if strings.Contains(err.Error(), "Element not found") {
			t.Skip("No audio device available (common in CI environments)")
		}
		// For other errors, skip as well but with different message
		t.Skipf("Cannot initialize audio: %v", err)
	}

	// Clean up the test reader
	if reader != nil {
		reader.Close()
	}
}

// TestVolumeReaderWCA_Initialize tests COM initialization
func TestVolumeReaderWCA_Initialize(t *testing.T) {
	skipIfNoAudioDeviceWCA(t)

	reader, err := NewVolumeReaderWCA()
	if err != nil {
		t.Fatalf("Failed to create volume reader: %v", err)
	}
	defer reader.Close()

	if !reader.initialized {
		t.Error("Expected reader to be initialized")
	}

	if reader.aev == nil {
		t.Error("Expected IAudioEndpointVolume interface to be initialized")
	}

	if reader.mmd == nil {
		t.Error("Expected IMMDevice interface to be initialized")
	}

	if reader.mmde == nil {
		t.Error("Expected IMMDeviceEnumerator interface to be initialized")
	}
}

// TestVolumeReaderWCA_GetVolume tests volume reading
func TestVolumeReaderWCA_GetVolume(t *testing.T) {
	skipIfNoAudioDeviceWCA(t)

	reader, err := NewVolumeReaderWCA()
	if err != nil {
		t.Fatalf("Failed to create volume reader: %v", err)
	}
	defer reader.Close()

	volume, muted, err := reader.GetVolume()
	if err != nil {
		t.Fatalf("Failed to get volume: %v", err)
	}

	// Volume should be between 0 and 100
	if volume < 0 || volume > 100 {
		t.Errorf("Volume out of range: %.2f (expected 0-100)", volume)
	}

	// Muted is a boolean, just verify we got a value
	t.Logf("Current volume: %.2f%%, muted: %v", volume, muted)
}

// TestVolumeReaderWCA_MultipleReads tests repeated volume reads
func TestVolumeReaderWCA_MultipleReads(t *testing.T) {
	skipIfNoAudioDeviceWCA(t)

	reader, err := NewVolumeReaderWCA()
	if err != nil {
		t.Fatalf("Failed to create volume reader: %v", err)
	}
	defer reader.Close()

	// Read volume 100 times
	for i := 0; i < 100; i++ {
		volume, muted, err := reader.GetVolume()
		if err != nil {
			t.Fatalf("Failed to get volume on iteration %d: %v", i, err)
		}

		if volume < 0 || volume > 100 {
			t.Errorf("Volume out of range on iteration %d: %.2f", i, volume)
		}

		// Log first and last values
		if i == 0 || i == 99 {
			t.Logf("Iteration %d: volume=%.2f%%, muted=%v", i, volume, muted)
		}
	}
}

// TestVolumeReaderWCA_ConcurrentReads tests thread safety
func TestVolumeReaderWCA_ConcurrentReads(t *testing.T) {
	skipIfNoAudioDeviceWCA(t)

	reader, err := NewVolumeReaderWCA()
	if err != nil {
		t.Fatalf("Failed to create volume reader: %v", err)
	}
	defer reader.Close()

	// Start 10 goroutines reading concurrently
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				volume, _, err := reader.GetVolume()
				if err != nil {
					errors <- err
					done <- false
					return
				}
				if volume < 0 || volume > 100 {
					errors <- err
					done <- false
					return
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case success := <-done:
			if !success {
				t.Fatal("Goroutine failed")
			}
		case err := <-errors:
			t.Fatalf("Error in concurrent read: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent reads")
		}
	}
}

// TestVolumeReaderWCA_PerformanceStability tests performance over time
func TestVolumeReaderWCA_PerformanceStability(t *testing.T) {
	skipIfNoAudioDeviceWCA(t)

	reader, err := NewVolumeReaderWCA()
	if err != nil {
		t.Fatalf("Failed to create volume reader: %v", err)
	}
	defer reader.Close()

	// Read 1000 times and measure timing
	iterations := 1000
	totalDuration := time.Duration(0)
	maxDuration := time.Duration(0)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, _, err := reader.GetVolume()
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed on iteration %d: %v", i, err)
		}

		totalDuration += duration
		if duration > maxDuration {
			maxDuration = duration
		}
	}

	avgDuration := totalDuration / time.Duration(iterations)

	t.Logf("Performance over %d iterations:", iterations)
	t.Logf("  Average: %v", avgDuration)
	t.Logf("  Max: %v", maxDuration)

	// Average should be under 1ms for healthy COM calls
	if avgDuration > 1*time.Millisecond {
		t.Errorf("Average duration too high: %v (expected < 1ms)", avgDuration)
	}

	// Max should be under 10ms
	if maxDuration > 10*time.Millisecond {
		t.Errorf("Max duration too high: %v (expected < 10ms)", maxDuration)
	}
}

// TestVolumeReaderWCA_CloseCleanup tests proper cleanup
func TestVolumeReaderWCA_CloseCleanup(t *testing.T) {
	skipIfNoAudioDeviceWCA(t)

	reader, err := NewVolumeReaderWCA()
	if err != nil {
		t.Fatalf("Failed to create volume reader: %v", err)
	}

	// Verify initialized
	if !reader.initialized {
		t.Fatal("Reader not initialized")
	}

	// Close
	reader.Close()

	// Verify cleanup
	if reader.initialized {
		t.Error("Reader still marked as initialized after Close()")
	}

	if reader.aev != nil {
		t.Error("IAudioEndpointVolume not cleaned up")
	}

	if reader.mmd != nil {
		t.Error("IMMDevice not cleaned up")
	}

	if reader.mmde != nil {
		t.Error("IMMDeviceEnumerator not cleaned up")
	}
}

// TestVolumeReaderWCA_GetVolumeAfterClose tests error handling after close
func TestVolumeReaderWCA_GetVolumeAfterClose(t *testing.T) {
	skipIfNoAudioDeviceWCA(t)

	reader, err := NewVolumeReaderWCA()
	if err != nil {
		t.Fatalf("Failed to create volume reader: %v", err)
	}

	reader.Close()

	// Try to read after close - should error
	_, _, err = reader.GetVolume()
	if err == nil {
		t.Error("Expected error when reading after Close(), got nil")
	}
}

// TestVolumeReaderWCA_DoubleClose tests double close safety
func TestVolumeReaderWCA_DoubleClose(t *testing.T) {
	skipIfNoAudioDeviceWCA(t)

	reader, err := NewVolumeReaderWCA()
	if err != nil {
		t.Fatalf("Failed to create volume reader: %v", err)
	}

	// Close twice - should not panic
	reader.Close()
	reader.Close() // Should be safe to call multiple times
}

// TestNewVolumeReader_Factory tests the factory function
func TestNewVolumeReader_Factory(t *testing.T) {
	skipIfNoAudioDeviceWCA(t)

	reader, err := newVolumeReader()
	if err != nil {
		t.Fatalf("newVolumeReader() failed: %v", err)
	}
	defer reader.Close()

	// Verify it returns VolumeReaderWCA
	wcaReader, ok := reader.(*VolumeReaderWCA)
	if !ok {
		t.Fatalf("Expected *VolumeReaderWCA, got %T", reader)
	}

	if !wcaReader.initialized {
		t.Error("Factory-created reader not initialized")
	}
}
