//go:build windows

package audiovisualizer

import (
	"strings"
	"testing"
	"time"
)

// skipIfNoAudioDeviceCapture skips the test if audio device is not available
func skipIfNoAudioDeviceCapture(t *testing.T) {
	t.Helper()

	capture, err := GetSharedAudioCapture()
	if err != nil {
		if strings.Contains(err.Error(), "Element not found") {
			t.Skip("No audio device available (common in CI environments)")
		}
		t.Skipf("Cannot initialize audio capture: %v", err)
	}

	if capture == nil {
		t.Skip("Audio capture returned nil")
	}
}

// TestAudioCaptureWCA_Initialize tests COM initialization for audio capture
func TestAudioCaptureWCA_Initialize(t *testing.T) {
	skipIfNoAudioDeviceCapture(t)

	capture, err := GetSharedAudioCapture()
	if err != nil {
		t.Fatalf("GetSharedAudioCapture() failed: %v", err)
	}

	if !capture.initialized {
		t.Error("Expected capture to be initialized")
	}

	if capture.audioClient == nil {
		t.Error("Expected IAudioClient interface to be initialized")
	}

	if capture.captureClient == nil {
		t.Error("Expected IAudioCaptureClient interface to be initialized")
	}

	if capture.mmd == nil {
		t.Error("Expected IMMDevice interface to be initialized")
	}

	if capture.mmde == nil {
		t.Error("Expected IMMDeviceEnumerator interface to be initialized")
	}

	if capture.sampleRate == 0 {
		t.Error("Expected sampleRate to be set")
	}

	t.Logf("Audio capture initialized: sampleRate=%d, bufferSize=%d", capture.sampleRate, capture.bufferSize)
}

// TestAudioCaptureWCA_ReadSamples tests reading audio samples
func TestAudioCaptureWCA_ReadSamples(t *testing.T) {
	skipIfNoAudioDeviceCapture(t)

	capture, err := GetSharedAudioCapture()
	if err != nil {
		t.Fatalf("GetSharedAudioCapture() failed: %v", err)
	}

	// Read samples - even with silence, should return empty slices without error
	leftSamples, rightSamples, err := capture.ReadSamples()
	if err != nil {
		t.Fatalf("ReadSamples() failed: %v", err)
	}

	// Samples might be empty if no audio is playing
	if leftSamples == nil {
		t.Error("Expected non-nil left samples slice (even if empty)")
	}

	if rightSamples == nil {
		t.Error("Expected non-nil right samples slice (even if empty)")
	}

	// Left and right channels should have same length
	if len(leftSamples) != len(rightSamples) {
		t.Errorf("Channel length mismatch: left=%d, right=%d", len(leftSamples), len(rightSamples))
	}

	// All samples should be in valid range [-1.0, 1.0]
	for i, sample := range leftSamples {
		if sample < -1.0 || sample > 1.0 {
			t.Errorf("Left sample %d out of range: %.3f (expected [-1.0, 1.0])", i, sample)
		}
	}

	for i, sample := range rightSamples {
		if sample < -1.0 || sample > 1.0 {
			t.Errorf("Right sample %d out of range: %.3f (expected [-1.0, 1.0])", i, sample)
		}
	}

	t.Logf("Read %d sample frames from audio capture", len(leftSamples))
}

// TestAudioCaptureWCA_MultipleReads tests repeated audio capture reads
func TestAudioCaptureWCA_MultipleReads(t *testing.T) {
	skipIfNoAudioDeviceCapture(t)

	capture, err := GetSharedAudioCapture()
	if err != nil {
		t.Fatalf("GetSharedAudioCapture() failed: %v", err)
	}

	// Read samples 50 times
	for i := 0; i < 50; i++ {
		leftSamples, rightSamples, err := capture.ReadSamples()
		if err != nil {
			t.Fatalf("ReadSamples() iteration %d failed: %v", i, err)
		}

		if leftSamples == nil || rightSamples == nil {
			t.Fatalf("Iteration %d: got nil samples", i)
		}

		if len(leftSamples) != len(rightSamples) {
			t.Errorf("Iteration %d: channel length mismatch", i)
		}

		// Log first and last iterations
		if i == 0 || i == 49 {
			t.Logf("Iteration %d: read %d sample frames", i, len(leftSamples))
		}

		// Small delay to allow audio buffer to fill
		time.Sleep(10 * time.Millisecond)
	}
}

// TestAudioCaptureWCA_ConcurrentReads tests thread safety of audio reads
func TestAudioCaptureWCA_ConcurrentReads(t *testing.T) {
	skipIfNoAudioDeviceCapture(t)

	capture, err := GetSharedAudioCapture()
	if err != nil {
		t.Fatalf("GetSharedAudioCapture() failed: %v", err)
	}

	// Start 5 goroutines reading concurrently
	done := make(chan bool, 5)
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				leftSamples, rightSamples, err := capture.ReadSamples()
				if err != nil {
					errors <- err
					done <- false
					return
				}
				if leftSamples == nil || rightSamples == nil {
					errors <- err
					done <- false
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		select {
		case success := <-done:
			if !success {
				t.Fatal("Goroutine failed")
			}
		case err := <-errors:
			t.Fatalf("Error in concurrent read: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent reads")
		}
	}
}

// TestAudioCaptureWCA_SampleRate tests that sample rate is valid
func TestAudioCaptureWCA_SampleRate(t *testing.T) {
	skipIfNoAudioDeviceCapture(t)

	capture, err := GetSharedAudioCapture()
	if err != nil {
		t.Fatalf("GetSharedAudioCapture() failed: %v", err)
	}

	// Sample rate should be a common audio rate
	// Common rates: 44100, 48000, 88200, 96000, 176400, 192000
	commonRates := map[uint32]bool{
		44100:  true,
		48000:  true,
		88200:  true,
		96000:  true,
		176400: true,
		192000: true,
	}

	if capture.sampleRate == 0 {
		t.Fatal("Sample rate is 0")
	}

	if !commonRates[capture.sampleRate] {
		t.Logf("Warning: Unusual sample rate: %d Hz (not in common rates)", capture.sampleRate)
	} else {
		t.Logf("Sample rate: %d Hz", capture.sampleRate)
	}
}

// TestAudioCaptureWCA_BufferSize tests that buffer size is valid
func TestAudioCaptureWCA_BufferSize(t *testing.T) {
	skipIfNoAudioDeviceCapture(t)

	capture, err := GetSharedAudioCapture()
	if err != nil {
		t.Fatalf("GetSharedAudioCapture() failed: %v", err)
	}

	if capture.bufferSize == 0 {
		t.Error("Buffer size is 0")
	}

	// Buffer size should be reasonable (typically 480-2048 frames for 20ms at 48kHz)
	if capture.bufferSize < 100 || capture.bufferSize > 10000 {
		t.Logf("Warning: Unusual buffer size: %d frames", capture.bufferSize)
	} else {
		t.Logf("Buffer size: %d frames", capture.bufferSize)
	}

	// Calculate buffer duration in milliseconds
	durationMs := float64(capture.bufferSize) / float64(capture.sampleRate) * 1000.0
	t.Logf("Buffer duration: %.1f ms", durationMs)
}

// TestAudioCaptureWCA_CloseSemantics tests Close() semantics
func TestAudioCaptureWCA_CloseSemantics(t *testing.T) {
	// Note: We can't actually call Close() on the shared singleton
	// because it would affect other tests. This test documents the
	// expected cleanup behavior by verifying the singleton is initialized.
	//
	// The architecture assumes all WCA objects use singletons and are
	// created on the same goroutine. Direct instantiation on different
	// threads violates COM threading assumptions.

	skipIfNoAudioDeviceCapture(t)

	capture, err := GetSharedAudioCapture()
	if err != nil {
		t.Fatalf("GetSharedAudioCapture() failed: %v", err)
	}

	// Verify the singleton is properly initialized
	if !capture.initialized {
		t.Error("Shared capture should be initialized")
	}

	if capture.audioClient == nil {
		t.Error("IAudioClient should be initialized")
	}

	if capture.captureClient == nil {
		t.Error("IAudioCaptureClient should be initialized")
	}

	if capture.mmd == nil {
		t.Error("IMMDevice should be initialized")
	}

	if capture.mmde == nil {
		t.Error("IMMDeviceEnumerator should be initialized")
	}

	// Note: We don't call Close() here because:
	// 1. The singleton is shared across tests
	// 2. COM cleanup happens at process exit
	// 3. SafeRelease functions are tested separately in wca_windows_test.go

	t.Log("Close() would cleanup: audioClient, captureClient, mmd, mmde")
}
