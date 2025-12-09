package shared

import (
	"testing"
	"time"
)

func TestNewBlinkAnimator(t *testing.T) {
	b := NewBlinkAnimator(BlinkAlways, 500*time.Millisecond)

	if b == nil {
		t.Fatal("NewBlinkAnimator returned nil")
	}
	if b.Mode() != BlinkAlways {
		t.Errorf("Mode() = %s, want %s", b.Mode(), BlinkAlways)
	}
	if !b.State() {
		t.Error("State() = false initially, want true")
	}
}

func TestBlinkAnimator_NeverMode(t *testing.T) {
	b := NewBlinkAnimator(BlinkNever, 500*time.Millisecond)

	// Should always return true for ShouldRender
	if !b.ShouldRender() {
		t.Error("ShouldRender() = false for BlinkNever, want true")
	}

	// GetInterval should return 0
	if b.GetInterval(10) != 0 {
		t.Errorf("GetInterval(10) = %v for BlinkNever, want 0", b.GetInterval(10))
	}

	// Update should not change state
	changed := b.Update(10)
	if changed {
		t.Error("Update() returned true for BlinkNever, want false")
	}
}

func TestBlinkAnimator_AlwaysMode(t *testing.T) {
	b := NewBlinkAnimator(BlinkAlways, 10*time.Millisecond)

	// Should use base interval
	if b.GetInterval(0) != 10*time.Millisecond {
		t.Errorf("GetInterval(0) = %v, want 10ms", b.GetInterval(0))
	}
	if b.GetInterval(100) != 10*time.Millisecond {
		t.Errorf("GetInterval(100) = %v, want 10ms (intensity ignored)", b.GetInterval(100))
	}

	// Wait for toggle
	time.Sleep(15 * time.Millisecond)
	b.Update(0)

	if b.State() {
		t.Error("State() = true after interval, want false (toggled)")
	}
}

func TestBlinkAnimator_ProgressiveMode(t *testing.T) {
	b := NewBlinkAnimator(BlinkProgressive, 1000*time.Millisecond)

	// At intensity 0, should not blink
	if b.GetInterval(0) != 0 {
		t.Errorf("GetInterval(0) = %v, want 0 (no blink at 0 intensity)", b.GetInterval(0))
	}

	// At intensity 1, should use base interval
	if b.GetInterval(1) != 1000*time.Millisecond {
		t.Errorf("GetInterval(1) = %v, want 1000ms", b.GetInterval(1))
	}

	// At intensity 5, should be faster
	interval5 := b.GetInterval(5)
	if interval5 >= 1000*time.Millisecond || interval5 < 100*time.Millisecond {
		t.Errorf("GetInterval(5) = %v, want between 100ms and 1000ms", interval5)
	}

	// At intensity 10+, should be at minimum
	interval10 := b.GetInterval(10)
	interval20 := b.GetInterval(20)
	if interval10 < 100*time.Millisecond {
		t.Errorf("GetInterval(10) = %v, want >= 100ms (clamped)", interval10)
	}
	if interval20 != interval10 {
		t.Errorf("GetInterval(20) = %v != GetInterval(10) = %v (should be clamped)", interval20, interval10)
	}
}

func TestBlinkAnimator_Update(t *testing.T) {
	b := NewBlinkAnimator(BlinkAlways, 10*time.Millisecond)

	// Initial state
	if !b.State() {
		t.Error("Initial State() = false, want true")
	}

	// Immediate update should not toggle
	changed := b.Update(0)
	if changed {
		t.Error("Update() changed state immediately, want no change")
	}

	// Wait and update
	time.Sleep(15 * time.Millisecond)
	changed = b.Update(0)
	if !changed {
		t.Error("Update() did not change state after interval")
	}
	if b.State() {
		t.Error("State() = true after toggle, want false")
	}

	// Toggle again
	time.Sleep(15 * time.Millisecond)
	b.Update(0)
	if !b.State() {
		t.Error("State() = false after second toggle, want true")
	}
}

func TestBlinkAnimator_UpdateWithTime(t *testing.T) {
	b := NewBlinkAnimator(BlinkAlways, 100*time.Millisecond)

	baseTime := time.Now()

	// Immediate update should not toggle
	changed := b.UpdateWithTime(baseTime, 0)
	if changed {
		t.Error("UpdateWithTime() changed state immediately")
	}

	// Update with future time should toggle
	futureTime := baseTime.Add(150 * time.Millisecond)
	changed = b.UpdateWithTime(futureTime, 0)
	if !changed {
		t.Error("UpdateWithTime() did not change state with future time")
	}
	if b.State() {
		t.Error("State() = true after toggle, want false")
	}
}

func TestBlinkAnimator_ShouldRender(t *testing.T) {
	tests := []struct {
		mode       BlinkMode
		state      bool
		wantRender bool
	}{
		{BlinkNever, true, true},
		{BlinkNever, false, true}, // Never mode always renders
		{BlinkAlways, true, true},
		{BlinkAlways, false, false},
		{BlinkProgressive, true, true},
		{BlinkProgressive, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			b := NewBlinkAnimator(tt.mode, 500*time.Millisecond)
			b.state = tt.state

			if got := b.ShouldRender(); got != tt.wantRender {
				t.Errorf("ShouldRender() = %v, want %v (mode=%s, state=%v)",
					got, tt.wantRender, tt.mode, tt.state)
			}
		})
	}
}

func TestBlinkAnimator_Reset(t *testing.T) {
	b := NewBlinkAnimator(BlinkAlways, 10*time.Millisecond)

	// Toggle state
	time.Sleep(15 * time.Millisecond)
	b.Update(0)

	if b.State() {
		t.Error("State() should be false before Reset")
	}

	b.Reset()

	if !b.State() {
		t.Error("State() = false after Reset, want true")
	}
}

func TestBlinkAnimator_SetMode(t *testing.T) {
	b := NewBlinkAnimator(BlinkNever, 500*time.Millisecond)

	b.SetMode(BlinkAlways)

	if b.Mode() != BlinkAlways {
		t.Errorf("Mode() = %s after SetMode, want %s", b.Mode(), BlinkAlways)
	}
}
