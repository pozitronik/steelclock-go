package compositor

import (
	"testing"
)

func TestNewFrameDeduplicator(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		d := NewFrameDeduplicator(true)
		if !d.IsEnabled() {
			t.Error("expected deduplicator to be enabled")
		}
	})

	t.Run("disabled", func(t *testing.T) {
		d := NewFrameDeduplicator(false)
		if d.IsEnabled() {
			t.Error("expected deduplicator to be disabled")
		}
	})
}

func TestFrameDeduplicator_HasChanged_Disabled(t *testing.T) {
	d := NewFrameDeduplicator(false)

	data := map[string][]byte{
		"key1": {0x00, 0x01, 0x02},
	}

	// When disabled, always returns true
	if !d.HasChanged(data) {
		t.Error("disabled deduplicator should always return true")
	}

	d.Update(data)

	// Still returns true even after update
	if !d.HasChanged(data) {
		t.Error("disabled deduplicator should always return true")
	}
}

func TestFrameDeduplicator_HasChanged_Enabled(t *testing.T) {
	d := NewFrameDeduplicator(true)

	data1 := map[string][]byte{
		"key1": {0x00, 0x01, 0x02},
	}

	// First check - no previous data, should report changed
	if !d.HasChanged(data1) {
		t.Error("first frame should be detected as changed")
	}

	d.Update(data1)

	// Same data - should report unchanged
	if d.HasChanged(data1) {
		t.Error("identical frame should not be detected as changed")
	}

	// Different data - should report changed
	data2 := map[string][]byte{
		"key1": {0x00, 0x01, 0x03}, // Last byte different
	}
	if !d.HasChanged(data2) {
		t.Error("different frame should be detected as changed")
	}
}

func TestFrameDeduplicator_HasChanged_MultipleResolutions(t *testing.T) {
	d := NewFrameDeduplicator(true)

	data := map[string][]byte{
		"res-128x40": {0x00, 0x01},
		"res-128x52": {0x02, 0x03},
	}

	d.Update(data)

	// Same data for all resolutions
	if d.HasChanged(data) {
		t.Error("identical frames should not be detected as changed")
	}

	// Change only one resolution
	dataPartialChange := map[string][]byte{
		"res-128x40": {0x00, 0x01}, // Same
		"res-128x52": {0xFF, 0xFF}, // Different
	}
	if !d.HasChanged(dataPartialChange) {
		t.Error("partial change should be detected")
	}
}

func TestFrameDeduplicator_Update(t *testing.T) {
	d := NewFrameDeduplicator(true)

	data1 := map[string][]byte{
		"key1": {0x00, 0x01},
	}
	d.Update(data1)

	// Verify the data was stored (by checking HasChanged)
	if d.HasChanged(data1) {
		t.Error("after update, same data should not be detected as changed")
	}

	// Update with new data
	data2 := map[string][]byte{
		"key1": {0xFF, 0xFF},
	}
	d.Update(data2)

	// Old data should now be detected as changed
	if !d.HasChanged(data1) {
		t.Error("after update with new data, old data should be detected as changed")
	}
}

func TestFrameDeduplicator_Update_Disabled(t *testing.T) {
	d := NewFrameDeduplicator(false)

	data := map[string][]byte{
		"key1": {0x00, 0x01},
	}

	// Update should be a no-op when disabled
	d.Update(data)

	// HasChanged should still return true
	if !d.HasChanged(data) {
		t.Error("disabled deduplicator should always return true")
	}
}

func TestFrameDeduplicator_Reset(t *testing.T) {
	d := NewFrameDeduplicator(true)

	data := map[string][]byte{
		"key1": {0x00, 0x01},
	}
	d.Update(data)

	// Before reset - unchanged
	if d.HasChanged(data) {
		t.Error("before reset, same data should not be changed")
	}

	d.Reset()

	// After reset - should be detected as changed (no stored data)
	if !d.HasChanged(data) {
		t.Error("after reset, data should be detected as changed")
	}
}

func TestFrameDeduplicator_DeepCopy(t *testing.T) {
	d := NewFrameDeduplicator(true)

	original := []byte{0x00, 0x01, 0x02}
	data := map[string][]byte{
		"key1": original,
	}
	d.Update(data)

	// Modify original slice
	original[0] = 0xFF

	// The stored data should not be affected
	if d.HasChanged(map[string][]byte{"key1": {0x00, 0x01, 0x02}}) {
		t.Error("deduplicator should store a copy, not reference")
	}
}
