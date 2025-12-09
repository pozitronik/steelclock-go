package compositor

import (
	"fmt"
	"sync"
)

// FrameSender is the interface for sending batched frames.
type FrameSender interface {
	SendMultipleScreenData(eventName string, frames [][]byte) error
}

// FrameBatcher buffers frames and sends them in batches.
// It handles thread-safe buffering and automatic flushing when batch size is reached.
type FrameBatcher struct {
	mu        sync.Mutex
	enabled   bool
	batchSize int
	buffer    [][]byte
	sender    FrameSender
	eventName string
}

// NewFrameBatcher creates a new batcher.
// If enabled is false, Add() returns immediately without buffering.
func NewFrameBatcher(enabled bool, batchSize int, sender FrameSender, eventName string) *FrameBatcher {
	var buffer [][]byte
	if enabled {
		buffer = make([][]byte, 0, batchSize)
	}

	return &FrameBatcher{
		enabled:   enabled,
		batchSize: batchSize,
		buffer:    buffer,
		sender:    sender,
		eventName: eventName,
	}
}

// IsEnabled returns whether batching is active.
func (b *FrameBatcher) IsEnabled() bool {
	return b.enabled
}

// Add adds a frame to the buffer.
// Returns (shouldSendDirectly, error).
// If batching is disabled, returns (true, nil) indicating caller should send directly.
// If batch is full after adding, flushes and returns (false, flushError).
// Otherwise, returns (false, nil) indicating frame was buffered.
func (b *FrameBatcher) Add(frame []byte) (shouldSendDirectly bool, err error) {
	if !b.enabled {
		return true, nil
	}

	b.mu.Lock()
	b.buffer = append(b.buffer, frame)
	shouldFlush := len(b.buffer) >= b.batchSize
	b.mu.Unlock()

	if shouldFlush {
		return false, b.Flush()
	}

	return false, nil
}

// Flush sends all buffered frames immediately.
// Safe to call even if buffer is empty.
func (b *FrameBatcher) Flush() error {
	if !b.enabled {
		return nil
	}

	b.mu.Lock()
	if len(b.buffer) == 0 {
		b.mu.Unlock()
		return nil
	}

	// Copy buffer to send
	framesToSend := make([][]byte, len(b.buffer))
	copy(framesToSend, b.buffer)
	b.buffer = b.buffer[:0] // Clear buffer
	b.mu.Unlock()

	// Send batch
	if err := b.sender.SendMultipleScreenData(b.eventName, framesToSend); err != nil {
		return fmt.Errorf("batch send failed: %w", err)
	}

	return nil
}

// BufferedCount returns the number of frames currently in the buffer.
func (b *FrameBatcher) BufferedCount() int {
	if !b.enabled {
		return 0
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.buffer)
}

// Reset clears the buffer without sending.
func (b *FrameBatcher) Reset() {
	if !b.enabled {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.buffer = b.buffer[:0]
}
