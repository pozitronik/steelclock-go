package compositor

import (
	"errors"
	"sync"
	"testing"
)

// mockFrameSender implements FrameSender for testing
type mockFrameSender struct {
	mu         sync.Mutex
	sendCalls  [][]byte
	sendError  error
	batchCount int
}

func newMockFrameSender() *mockFrameSender {
	return &mockFrameSender{}
}

func (m *mockFrameSender) SendMultipleScreenData(_ string, frames [][]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendError != nil {
		return m.sendError
	}

	m.batchCount++
	for _, f := range frames {
		frameCopy := make([]byte, len(f))
		copy(frameCopy, f)
		m.sendCalls = append(m.sendCalls, frameCopy)
	}
	return nil
}

func (m *mockFrameSender) getBatchCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.batchCount
}

func (m *mockFrameSender) getFrameCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sendCalls)
}

func TestNewFrameBatcher(t *testing.T) {
	sender := newMockFrameSender()

	t.Run("enabled", func(t *testing.T) {
		b := NewFrameBatcher(true, 5, sender, "EVENT")
		if !b.IsEnabled() {
			t.Error("expected batcher to be enabled")
		}
	})

	t.Run("disabled", func(t *testing.T) {
		b := NewFrameBatcher(false, 5, sender, "EVENT")
		if b.IsEnabled() {
			t.Error("expected batcher to be disabled")
		}
	})
}

func TestFrameBatcher_Add_Disabled(t *testing.T) {
	sender := newMockFrameSender()
	b := NewFrameBatcher(false, 5, sender, "EVENT")

	frame := []byte{0x01, 0x02, 0x03}
	shouldSend, err := b.Add(frame)

	if err != nil {
		t.Errorf("Add() error = %v", err)
	}
	if !shouldSend {
		t.Error("Add() should return shouldSendDirectly=true when disabled")
	}
	if b.BufferedCount() != 0 {
		t.Error("BufferedCount should be 0 when disabled")
	}
}

func TestFrameBatcher_Add_Enabled(t *testing.T) {
	sender := newMockFrameSender()
	b := NewFrameBatcher(true, 3, sender, "EVENT")

	// Add first frame - should buffer
	shouldSend, err := b.Add([]byte{0x01})
	if err != nil {
		t.Errorf("Add() error = %v", err)
	}
	if shouldSend {
		t.Error("First Add() should return shouldSendDirectly=false")
	}
	if b.BufferedCount() != 1 {
		t.Errorf("BufferedCount = %d, want 1", b.BufferedCount())
	}

	// Add second frame - should buffer
	shouldSend, err = b.Add([]byte{0x02})
	if err != nil {
		t.Errorf("Add() error = %v", err)
	}
	if shouldSend {
		t.Error("Second Add() should return shouldSendDirectly=false")
	}
	if b.BufferedCount() != 2 {
		t.Errorf("BufferedCount = %d, want 2", b.BufferedCount())
	}

	// Add third frame - should flush (batch size = 3)
	shouldSend, err = b.Add([]byte{0x03})
	if err != nil {
		t.Errorf("Add() error = %v", err)
	}
	if shouldSend {
		t.Error("Third Add() should return shouldSendDirectly=false (flushed)")
	}
	if b.BufferedCount() != 0 {
		t.Errorf("BufferedCount = %d, want 0 after flush", b.BufferedCount())
	}

	// Verify sender received the batch
	if sender.getBatchCount() != 1 {
		t.Errorf("BatchCount = %d, want 1", sender.getBatchCount())
	}
	if sender.getFrameCount() != 3 {
		t.Errorf("FrameCount = %d, want 3", sender.getFrameCount())
	}
}

func TestFrameBatcher_Flush(t *testing.T) {
	sender := newMockFrameSender()
	b := NewFrameBatcher(true, 10, sender, "EVENT")

	// Add some frames
	_, _ = b.Add([]byte{0x01})
	_, _ = b.Add([]byte{0x02})

	if b.BufferedCount() != 2 {
		t.Errorf("BufferedCount = %d, want 2", b.BufferedCount())
	}

	// Manual flush
	err := b.Flush()
	if err != nil {
		t.Errorf("Flush() error = %v", err)
	}

	if b.BufferedCount() != 0 {
		t.Errorf("BufferedCount = %d, want 0 after flush", b.BufferedCount())
	}

	if sender.getFrameCount() != 2 {
		t.Errorf("FrameCount = %d, want 2", sender.getFrameCount())
	}
}

func TestFrameBatcher_Flush_Empty(t *testing.T) {
	sender := newMockFrameSender()
	b := NewFrameBatcher(true, 10, sender, "EVENT")

	// Flush empty buffer - should be safe
	err := b.Flush()
	if err != nil {
		t.Errorf("Flush() error = %v", err)
	}

	if sender.getBatchCount() != 0 {
		t.Error("Should not send when buffer is empty")
	}
}

func TestFrameBatcher_Flush_Disabled(t *testing.T) {
	sender := newMockFrameSender()
	b := NewFrameBatcher(false, 10, sender, "EVENT")

	// Flush when disabled - should be safe no-op
	err := b.Flush()
	if err != nil {
		t.Errorf("Flush() error = %v", err)
	}

	if sender.getBatchCount() != 0 {
		t.Error("Should not send when disabled")
	}
}

func TestFrameBatcher_Flush_Error(t *testing.T) {
	sender := newMockFrameSender()
	sender.sendError = errors.New("send failed")
	b := NewFrameBatcher(true, 10, sender, "EVENT")

	_, _ = b.Add([]byte{0x01})

	err := b.Flush()
	if err == nil {
		t.Error("Flush() should return error")
	}
}

func TestFrameBatcher_Reset(t *testing.T) {
	sender := newMockFrameSender()
	b := NewFrameBatcher(true, 10, sender, "EVENT")

	_, _ = b.Add([]byte{0x01})
	_, _ = b.Add([]byte{0x02})

	b.Reset()

	if b.BufferedCount() != 0 {
		t.Errorf("BufferedCount = %d, want 0 after reset", b.BufferedCount())
	}

	// Flush should not send anything after reset
	_ = b.Flush()
	if sender.getBatchCount() != 0 {
		t.Error("Should not send after reset")
	}
}

func TestFrameBatcher_Reset_Disabled(t *testing.T) {
	sender := newMockFrameSender()
	b := NewFrameBatcher(false, 10, sender, "EVENT")

	// Reset when disabled - should be safe no-op
	b.Reset()

	if b.BufferedCount() != 0 {
		t.Error("BufferedCount should be 0")
	}
}

func TestFrameBatcher_BufferedCount_Disabled(t *testing.T) {
	sender := newMockFrameSender()
	b := NewFrameBatcher(false, 10, sender, "EVENT")

	if b.BufferedCount() != 0 {
		t.Error("BufferedCount should always be 0 when disabled")
	}
}
