package compositor

import (
	"bytes"
	"sync"
)

// FrameDeduplicator tracks frame state and determines if frames have changed.
// It handles comparison of bitmap data across multiple resolutions.
type FrameDeduplicator struct {
	mu         sync.RWMutex
	enabled    bool
	lastFrames map[string][]byte // key: resolution key (e.g., "image-data-128x40")
}

// NewFrameDeduplicator creates a new deduplicator.
// If enabled is false, HasChanged always returns true.
func NewFrameDeduplicator(enabled bool) *FrameDeduplicator {
	return &FrameDeduplicator{
		enabled:    enabled,
		lastFrames: make(map[string][]byte),
	}
}

// HasChanged checks if any frame in the resolution data differs from the last sent.
// Returns true if frames have changed or if deduplication is disabled.
func (d *FrameDeduplicator) HasChanged(resolutionData map[string][]byte) bool {
	if !d.enabled {
		return true
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	for key, data := range resolutionData {
		if !bytes.Equal(d.lastFrames[key], data) {
			return true
		}
	}

	return false
}

// Update stores the current frame data as the last sent frames.
// Should be called after successful send.
func (d *FrameDeduplicator) Update(resolutionData map[string][]byte) {
	if !d.enabled {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	for key, data := range resolutionData {
		if d.lastFrames[key] == nil {
			d.lastFrames[key] = make([]byte, len(data))
		}
		copy(d.lastFrames[key], data)
	}
}

// Reset clears all stored frame data.
func (d *FrameDeduplicator) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastFrames = make(map[string][]byte)
}

// IsEnabled returns whether deduplication is active.
func (d *FrameDeduplicator) IsEnabled() bool {
	return d.enabled
}
