package shared

import (
	"testing"
	"time"
)

func TestNewTextScroller(t *testing.T) {
	cfg := ScrollerConfig{
		Speed:     100,
		Mode:      ScrollContinuous,
		Direction: ScrollLeft,
		Gap:       10,
		PauseMs:   500,
	}
	s := NewTextScroller(cfg)

	if s == nil {
		t.Fatal("NewTextScroller returned nil")
	}
	if s.GetOffset() != 0 {
		t.Errorf("GetOffset() = %f, want 0", s.GetOffset())
	}
	if s.GetConfig().Speed != 100 {
		t.Errorf("Config.Speed = %f, want 100", s.GetConfig().Speed)
	}
}

func TestTextScroller_ContentFitsContainer(t *testing.T) {
	cfg := ScrollerConfig{Speed: 100, Mode: ScrollContinuous, Direction: ScrollLeft}
	s := NewTextScroller(cfg)

	// Content (50) fits in container (100) - no scrolling
	offset := s.Update(50, 100)

	if offset != 0 {
		t.Errorf("Update() = %f, want 0 (content fits)", offset)
	}
}

func TestTextScroller_ContinuousLeft(t *testing.T) {
	cfg := ScrollerConfig{
		Speed:     100, // 100 pixels per second
		Mode:      ScrollContinuous,
		Direction: ScrollLeft,
		Gap:       10,
	}
	s := NewTextScroller(cfg)

	baseTime := time.Now()
	contentSize := 200
	containerSize := 100

	// After 1 second at 100 px/s, should scroll 100 pixels
	futureTime := baseTime.Add(1 * time.Second)
	offset := s.UpdateWithTime(futureTime, contentSize, containerSize)

	if offset < 95 || offset > 105 {
		t.Errorf("offset after 1s = %f, want ~100", offset)
	}

	// After scrolling full content + gap, should wrap
	// totalSize = 200 + 10 = 210
	wrapTime := baseTime.Add(3 * time.Second) // 300 pixels, should wrap
	s.Reset()
	s.lastUpdate = baseTime
	offset = s.UpdateWithTime(wrapTime, contentSize, containerSize)

	if offset >= 210 {
		t.Errorf("offset after 3s = %f, should have wrapped (totalSize=210)", offset)
	}
}

func TestTextScroller_ContinuousRight(t *testing.T) {
	cfg := ScrollerConfig{
		Speed:     100,
		Mode:      ScrollContinuous,
		Direction: ScrollRight,
		Gap:       10,
	}
	s := NewTextScroller(cfg)

	baseTime := time.Now()

	futureTime := baseTime.Add(1 * time.Second)
	offset := s.UpdateWithTime(futureTime, 200, 100)

	// Right direction should give negative offset
	if offset > -95 || offset < -105 {
		t.Errorf("offset after 1s = %f, want ~-100", offset)
	}
}

func TestTextScroller_Bounce(t *testing.T) {
	cfg := ScrollerConfig{
		Speed:     100,
		Mode:      ScrollBounce,
		Direction: ScrollLeft,
		PauseMs:   0, // No pause for easier testing
	}
	s := NewTextScroller(cfg)

	contentSize := 150
	containerSize := 100
	maxOffset := float64(contentSize - containerSize) // 50

	baseTime := time.Now()

	// Scroll to max offset
	time1 := baseTime.Add(500 * time.Millisecond) // 50 pixels
	offset := s.UpdateWithTime(time1, contentSize, containerSize)
	if offset < 45 || offset > 55 {
		t.Errorf("offset at 500ms = %f, want ~50", offset)
	}

	// Should bounce back
	time2 := baseTime.Add(1 * time.Second) // Another 50 pixels, should be near start
	offset = s.UpdateWithTime(time2, contentSize, containerSize)
	if offset < 0 || offset > maxOffset {
		t.Errorf("offset after bounce = %f, should be between 0 and %f", offset, maxOffset)
	}
}

func TestTextScroller_BounceWithPause(t *testing.T) {
	cfg := ScrollerConfig{
		Speed:   1000, // Fast speed
		Mode:    ScrollBounce,
		PauseMs: 100,
	}
	s := NewTextScroller(cfg)

	baseTime := time.Now()

	// Scroll to end (should trigger pause)
	time1 := baseTime.Add(100 * time.Millisecond) // 100 pixels, exceeds maxOffset
	s.UpdateWithTime(time1, 150, 100)

	// During pause, offset should not change
	pauseOffset := s.GetOffset()
	time2 := time1.Add(50 * time.Millisecond) // Still in pause
	s.UpdateWithTime(time2, 150, 100)

	if s.GetOffset() != pauseOffset {
		t.Errorf("offset changed during pause: %f -> %f", pauseOffset, s.GetOffset())
	}

	// After pause, should continue
	time3 := time1.Add(150 * time.Millisecond) // Past pause
	s.UpdateWithTime(time3, 150, 100)

	if s.GetOffset() == pauseOffset {
		t.Error("offset didn't change after pause ended")
	}
}

func TestTextScroller_PauseEnds(t *testing.T) {
	cfg := ScrollerConfig{
		Speed:     100,
		Mode:      ScrollPauseEnds,
		Direction: ScrollLeft,
		PauseMs:   0, // No pause for testing
	}
	s := NewTextScroller(cfg)

	contentSize := 150
	containerSize := 100
	maxOffset := float64(contentSize - containerSize) // 50

	baseTime := time.Now()

	// Scroll past max offset - should reset to 0
	time1 := baseTime.Add(600 * time.Millisecond) // 60 pixels, exceeds maxOffset
	offset := s.UpdateWithTime(time1, contentSize, containerSize)

	if offset != 0 {
		t.Errorf("offset after exceeding max = %f, want 0 (reset)", offset)
	}

	_ = maxOffset // Suppress unused variable warning
}

func TestTextScroller_Reset(t *testing.T) {
	cfg := ScrollerConfig{Speed: 100, Mode: ScrollBounce}
	s := NewTextScroller(cfg)

	// Scroll some
	s.SetOffset(50)
	s.bounceDir = -1

	s.Reset()

	if s.GetOffset() != 0 {
		t.Errorf("GetOffset() = %f after Reset, want 0", s.GetOffset())
	}
	if s.bounceDir != 1 {
		t.Errorf("bounceDir = %d after Reset, want 1", s.bounceDir)
	}
}

func TestTextScroller_SetOffset(t *testing.T) {
	cfg := ScrollerConfig{Speed: 100, Mode: ScrollContinuous}
	s := NewTextScroller(cfg)

	s.SetOffset(42.5)

	if s.GetOffset() != 42.5 {
		t.Errorf("GetOffset() = %f, want 42.5", s.GetOffset())
	}
}

func TestTextScroller_SetSpeed(t *testing.T) {
	cfg := ScrollerConfig{Speed: 100, Mode: ScrollContinuous}
	s := NewTextScroller(cfg)

	s.SetSpeed(200)

	if s.GetConfig().Speed != 200 {
		t.Errorf("Config.Speed = %f, want 200", s.GetConfig().Speed)
	}
}

func TestTextScroller_IsHorizontal(t *testing.T) {
	tests := []struct {
		direction ScrollDirection
		wantH     bool
		wantV     bool
	}{
		{ScrollLeft, true, false},
		{ScrollRight, true, false},
		{ScrollUp, false, true},
		{ScrollDown, false, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.direction), func(t *testing.T) {
			cfg := ScrollerConfig{Direction: tt.direction}
			s := NewTextScroller(cfg)

			if s.IsHorizontal() != tt.wantH {
				t.Errorf("IsHorizontal() = %v, want %v", s.IsHorizontal(), tt.wantH)
			}
			if s.IsVertical() != tt.wantV {
				t.Errorf("IsVertical() = %v, want %v", s.IsVertical(), tt.wantV)
			}
		})
	}
}

func TestTextScroller_IsPaused(t *testing.T) {
	cfg := ScrollerConfig{
		Speed:   1000,
		Mode:    ScrollBounce,
		PauseMs: 100,
	}
	s := NewTextScroller(cfg)

	// Initially not paused
	if s.IsPaused() {
		t.Error("IsPaused() = true initially, want false")
	}

	// Manually trigger pause by setting pauseUntil to a future time
	s.pauseUntil = time.Now().Add(50 * time.Millisecond)

	if !s.IsPaused() {
		t.Error("IsPaused() = false when pauseUntil is in future, want true")
	}

	// Wait for pause to expire
	time.Sleep(60 * time.Millisecond)

	if s.IsPaused() {
		t.Error("IsPaused() = true after pauseUntil passed, want false")
	}
}

func TestTextScroller_VerticalScroll(t *testing.T) {
	cfg := ScrollerConfig{
		Speed:     100,
		Mode:      ScrollContinuous,
		Direction: ScrollUp,
		Gap:       5,
	}
	s := NewTextScroller(cfg)

	baseTime := time.Now()

	// Scroll up should increase offset
	time1 := baseTime.Add(500 * time.Millisecond)
	offset := s.UpdateWithTime(time1, 100, 50)

	if offset < 45 || offset > 55 {
		t.Errorf("offset after 500ms up = %f, want ~50", offset)
	}

	// Test down direction
	cfg.Direction = ScrollDown
	s = NewTextScroller(cfg)

	offset = s.UpdateWithTime(time1, 100, 50)

	if offset > -45 || offset < -55 {
		t.Errorf("offset after 500ms down = %f, want ~-50", offset)
	}
}
