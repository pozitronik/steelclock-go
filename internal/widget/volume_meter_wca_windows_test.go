//go:build windows

package widget

import (
	"strings"
	"testing"
)

// skipIfNoAudioDeviceMeterWCA skips the test if audio device is not available (for WCA tests)
func skipIfNoAudioDeviceMeterWCA(t *testing.T) {
	t.Helper()

	reader, err := NewMeterReaderWCA()
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

// TestNewMeterReaderWCA tests meter reader creation
func TestNewMeterReaderWCA(t *testing.T) {
	skipIfNoAudioDeviceMeterWCA(t)

	reader, err := NewMeterReaderWCA()
	if err != nil {
		t.Fatalf("NewMeterReaderWCA() error = %v", err)
	}

	if reader == nil {
		t.Fatal("NewMeterReaderWCA() returned nil reader")
	}

	defer reader.Close()

	// Verify initialization
	if !reader.initialized {
		t.Error("Reader should be initialized")
	}

	if reader.ami == nil {
		t.Error("IAudioMeterInformation should not be nil")
	}
}

// TestMeterReaderWCA_GetMeterData tests reading meter data
func TestMeterReaderWCA_GetMeterData(t *testing.T) {
	skipIfNoAudioDeviceMeterWCA(t)

	reader, err := NewMeterReaderWCA()
	if err != nil {
		t.Fatalf("NewMeterReaderWCA() error = %v", err)
	}
	defer reader.Close()

	clippingThreshold := 0.99
	silenceThreshold := 0.01

	data, err := reader.GetMeterData(clippingThreshold, silenceThreshold)
	if err != nil {
		t.Fatalf("GetMeterData() error = %v", err)
	}

	if data == nil {
		t.Fatal("GetMeterData() returned nil data")
	}

	// Validate peak value
	if data.Peak < 0 || data.Peak > 1.0 {
		t.Errorf("Peak = %.3f, should be in [0.0, 1.0]", data.Peak)
	}

	// Validate channel count
	if data.ChannelCount < 0 {
		t.Errorf("ChannelCount = %d, should be >= 0", data.ChannelCount)
	}

	// Validate channel peaks
	if len(data.ChannelPeaks) != data.ChannelCount {
		t.Errorf("ChannelPeaks length = %d, want %d", len(data.ChannelPeaks), data.ChannelCount)
	}

	for i, peak := range data.ChannelPeaks {
		if peak < 0 || peak > 1.0 {
			t.Errorf("Channel %d peak = %.3f, should be in [0.0, 1.0]", i, peak)
		}
	}

	t.Logf("Meter data: peak=%.3f, channels=%d, clipping=%v, hasAudio=%v",
		data.Peak, data.ChannelCount, data.IsClipping, data.HasAudio)
}

// TestMeterReaderWCA_MultipleCalls tests repeated meter reads
func TestMeterReaderWCA_MultipleCalls(t *testing.T) {
	skipIfNoAudioDeviceMeterWCA(t)

	reader, err := NewMeterReaderWCA()
	if err != nil {
		t.Fatalf("NewMeterReaderWCA() error = %v", err)
	}
	defer reader.Close()

	clippingThreshold := 0.99
	silenceThreshold := 0.01

	// Make 10 consecutive calls
	for i := 0; i < 10; i++ {
		data, err := reader.GetMeterData(clippingThreshold, silenceThreshold)
		if err != nil {
			t.Errorf("GetMeterData() call %d error = %v", i+1, err)
			continue
		}

		if data == nil {
			t.Errorf("GetMeterData() call %d returned nil data", i+1)
			continue
		}

		// Basic validation
		if data.Peak < 0 || data.Peak > 1.0 {
			t.Errorf("Call %d: Peak = %.3f out of range", i+1, data.Peak)
		}
	}

	t.Log("Successfully made 10 consecutive meter reads")
}

// TestMeterReaderWCA_ClippingDetection tests clipping flag
func TestMeterReaderWCA_ClippingDetection(t *testing.T) {
	skipIfNoAudioDeviceMeterWCA(t)

	reader, err := NewMeterReaderWCA()
	if err != nil {
		t.Fatalf("NewMeterReaderWCA() error = %v", err)
	}
	defer reader.Close()

	// Test with low threshold (most audio should trigger clipping)
	clippingThreshold := 0.01
	silenceThreshold := 0.001

	data, err := reader.GetMeterData(clippingThreshold, silenceThreshold)
	if err != nil {
		t.Fatalf("GetMeterData() error = %v", err)
	}

	// If peak is above threshold, clipping should be true
	if data.Peak >= clippingThreshold && !data.IsClipping {
		t.Errorf("Clipping should be true when peak (%.3f) >= threshold (%.3f)", data.Peak, clippingThreshold)
	}

	if data.Peak < clippingThreshold && data.IsClipping {
		t.Errorf("Clipping should be false when peak (%.3f) < threshold (%.3f)", data.Peak, clippingThreshold)
	}

	t.Logf("Clipping detection working: peak=%.3f, threshold=%.3f, clipping=%v",
		data.Peak, clippingThreshold, data.IsClipping)
}

// TestMeterReaderWCA_SilenceDetection tests silence/audio detection
func TestMeterReaderWCA_SilenceDetection(t *testing.T) {
	skipIfNoAudioDeviceMeterWCA(t)

	reader, err := NewMeterReaderWCA()
	if err != nil {
		t.Fatalf("NewMeterReaderWCA() error = %v", err)
	}
	defer reader.Close()

	clippingThreshold := 0.99
	silenceThreshold := 0.01

	data, err := reader.GetMeterData(clippingThreshold, silenceThreshold)
	if err != nil {
		t.Fatalf("GetMeterData() error = %v", err)
	}

	// If peak is above silence threshold, hasAudio should be true
	if data.Peak > silenceThreshold && !data.HasAudio {
		t.Errorf("HasAudio should be true when peak (%.3f) > threshold (%.3f)", data.Peak, silenceThreshold)
	}

	if data.Peak <= silenceThreshold && data.HasAudio {
		t.Errorf("HasAudio should be false when peak (%.3f) <= threshold (%.3f)", data.Peak, silenceThreshold)
	}

	t.Logf("Silence detection working: peak=%.3f, threshold=%.3f, hasAudio=%v",
		data.Peak, silenceThreshold, data.HasAudio)
}

// TestMeterReaderWCA_ChannelCount tests channel counting
func TestMeterReaderWCA_ChannelCount(t *testing.T) {
	skipIfNoAudioDeviceMeterWCA(t)

	reader, err := NewMeterReaderWCA()
	if err != nil {
		t.Fatalf("NewMeterReaderWCA() error = %v", err)
	}
	defer reader.Close()

	clippingThreshold := 0.99
	silenceThreshold := 0.01

	data, err := reader.GetMeterData(clippingThreshold, silenceThreshold)
	if err != nil {
		t.Fatalf("GetMeterData() error = %v", err)
	}

	// Most audio devices have 2 channels (stereo)
	if data.ChannelCount < 1 {
		t.Errorf("ChannelCount = %d, expected at least 1", data.ChannelCount)
	}

	t.Logf("Audio device has %d channel(s)", data.ChannelCount)

	// Common channel counts: 1 (mono), 2 (stereo), 6 (5.1), 8 (7.1)
	commonCounts := map[int]string{
		1: "Mono",
		2: "Stereo",
		6: "5.1 Surround",
		8: "7.1 Surround",
	}

	if name, ok := commonCounts[data.ChannelCount]; ok {
		t.Logf("Channel configuration: %s", name)
	}
}

// TestMeterReaderWCA_PerChannelPeaks tests individual channel peak values
func TestMeterReaderWCA_PerChannelPeaks(t *testing.T) {
	skipIfNoAudioDeviceMeterWCA(t)

	reader, err := NewMeterReaderWCA()
	if err != nil {
		t.Fatalf("NewMeterReaderWCA() error = %v", err)
	}
	defer reader.Close()

	clippingThreshold := 0.99
	silenceThreshold := 0.01

	data, err := reader.GetMeterData(clippingThreshold, silenceThreshold)
	if err != nil {
		t.Fatalf("GetMeterData() error = %v", err)
	}

	if data.ChannelCount == 0 {
		t.Skip("No channels available")
	}

	// Verify we have peak data for each channel
	if len(data.ChannelPeaks) != data.ChannelCount {
		t.Fatalf("ChannelPeaks length = %d, want %d", len(data.ChannelPeaks), data.ChannelCount)
	}

	// Log individual channel peaks
	for i, peak := range data.ChannelPeaks {
		t.Logf("Channel %d peak: %.3f", i, peak)

		if peak < 0 || peak > 1.0 {
			t.Errorf("Channel %d peak = %.3f out of valid range [0.0, 1.0]", i, peak)
		}
	}

	// Overall peak should be >= max of channel peaks
	maxChannelPeak := 0.0
	for _, peak := range data.ChannelPeaks {
		if peak > maxChannelPeak {
			maxChannelPeak = peak
		}
	}

	if data.Peak < maxChannelPeak-0.01 { // Allow small tolerance
		t.Errorf("Overall peak (%.3f) should be >= max channel peak (%.3f)", data.Peak, maxChannelPeak)
	}
}

// TestMeterReaderWCA_Close tests proper cleanup
func TestMeterReaderWCA_Close(t *testing.T) {
	skipIfNoAudioDeviceMeterWCA(t)

	reader, err := NewMeterReaderWCA()
	if err != nil {
		t.Fatalf("NewMeterReaderWCA() error = %v", err)
	}

	// Close should not panic
	reader.Close()

	// Reader should be uninitialized
	if reader.initialized {
		t.Error("Reader should be uninitialized after Close()")
	}

	// COM objects should be nil
	if reader.ami != nil {
		t.Error("IAudioMeterInformation should be nil after Close()")
	}

	if reader.mmd != nil {
		t.Error("IMMDevice should be nil after Close()")
	}

	if reader.mmde != nil {
		t.Error("IMMDeviceEnumerator should be nil after Close()")
	}

	// Calling Close again should not panic
	reader.Close()
}

// TestMeterReaderWCA_ErrorAfterClose tests that operations fail after Close
func TestMeterReaderWCA_ErrorAfterClose(t *testing.T) {
	skipIfNoAudioDeviceMeterWCA(t)

	reader, err := NewMeterReaderWCA()
	if err != nil {
		t.Fatalf("NewMeterReaderWCA() error = %v", err)
	}

	reader.Close()

	// GetMeterData should error after Close
	_, err = reader.GetMeterData(0.99, 0.01)
	if err == nil {
		t.Error("GetMeterData() should return error after Close()")
	} else if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("Error should mention 'not initialized', got: %v", err)
	}
}
