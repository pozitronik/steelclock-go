package testutil

import (
	"testing"
	"time"
)

func TestCalculateTimingStats_NotEnoughFrames(t *testing.T) {
	client := NewTestClient()

	stats := client.CalculateTimingStats()
	if stats.FrameCount != 0 {
		t.Errorf("Expected 0 frames, got %d", stats.FrameCount)
	}

	_ = client.SendScreenData("EVENT", make([]byte, 640))
	stats = client.CalculateTimingStats()
	if stats.FrameCount != 1 {
		t.Errorf("Expected 1 frame, got %d", stats.FrameCount)
	}
	if len(stats.Intervals) != 0 {
		t.Error("Should have no intervals with only 1 frame")
	}
}

func TestCalculateTimingStats_MultipleFrames(t *testing.T) {
	client := NewTestClient()

	// Send 5 frames with ~10ms intervals
	for i := 0; i < 5; i++ {
		_ = client.SendScreenData("EVENT", make([]byte, 640))
		time.Sleep(10 * time.Millisecond)
	}

	stats := client.CalculateTimingStats()

	if stats.FrameCount != 5 {
		t.Errorf("Expected 5 frames, got %d", stats.FrameCount)
	}

	if len(stats.Intervals) != 4 {
		t.Errorf("Expected 4 intervals, got %d", len(stats.Intervals))
	}

	// Average should be around 10ms (with some tolerance for test execution)
	if stats.AvgInterval < 5*time.Millisecond || stats.AvgInterval > 50*time.Millisecond {
		t.Errorf("Average interval out of expected range: %v", stats.AvgInterval)
	}

	if stats.AverageFPS <= 0 {
		t.Error("AverageFPS should be positive")
	}
}

func TestVerifyFrameRate(t *testing.T) {
	client := NewTestClient()

	// Send frames at roughly 100ms intervals
	for i := 0; i < 5; i++ {
		_ = client.SendScreenData("EVENT", make([]byte, 640))
		if i < 4 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Should pass with 50% tolerance
	err := client.VerifyFrameRate(100, 50)
	if err != nil {
		t.Errorf("VerifyFrameRate should pass: %v", err)
	}

	// Should fail with very tight tolerance and wrong expected rate
	err = client.VerifyFrameRate(10, 1)
	if err == nil {
		t.Error("VerifyFrameRate should fail with wrong expected rate")
	}
}

func TestVerifyFrameRate_NotEnoughFrames(t *testing.T) {
	client := NewTestClient()

	err := client.VerifyFrameRate(100, 10)
	if err == nil {
		t.Error("Should fail with no frames")
	}
}

func TestVerifyMinimumFrames(t *testing.T) {
	client := NewTestClient()

	// No frames yet
	err := client.VerifyMinimumFrames(1)
	if err == nil {
		t.Error("Should fail with 0 frames")
	}

	// Add some frames
	for i := 0; i < 5; i++ {
		_ = client.SendScreenData("EVENT", make([]byte, 640))
	}

	err = client.VerifyMinimumFrames(5)
	if err != nil {
		t.Errorf("Should pass with exactly 5 frames: %v", err)
	}

	err = client.VerifyMinimumFrames(3)
	if err != nil {
		t.Errorf("Should pass with more than required frames: %v", err)
	}

	err = client.VerifyMinimumFrames(10)
	if err == nil {
		t.Error("Should fail when requiring more frames than available")
	}
}

func TestVerifyFrameCountInRange(t *testing.T) {
	client := NewTestClient()

	for i := 0; i < 5; i++ {
		_ = client.SendScreenData("EVENT", make([]byte, 640))
	}

	err := client.VerifyFrameCountInRange(3, 10)
	if err != nil {
		t.Errorf("Should pass when count is in range: %v", err)
	}

	err = client.VerifyFrameCountInRange(5, 5)
	if err != nil {
		t.Errorf("Should pass when count equals both bounds: %v", err)
	}

	err = client.VerifyFrameCountInRange(6, 10)
	if err == nil {
		t.Error("Should fail when count is below range")
	}

	err = client.VerifyFrameCountInRange(1, 4)
	if err == nil {
		t.Error("Should fail when count is above range")
	}
}

func TestWaitForFrames(t *testing.T) {
	client := NewTestClient()

	// Start goroutine to send frames
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(10 * time.Millisecond)
			_ = client.SendScreenData("EVENT", make([]byte, 640))
		}
	}()

	err := client.WaitForFrames(3, 500*time.Millisecond)
	if err != nil {
		t.Errorf("WaitForFrames should succeed: %v", err)
	}
}

func TestWaitForFrames_Timeout(t *testing.T) {
	client := NewTestClient()

	err := client.WaitForFrames(10, 50*time.Millisecond)
	if err == nil {
		t.Error("WaitForFrames should timeout")
	}
}

func TestWaitForFrameMatching(t *testing.T) {
	client := NewTestClient()

	// Start goroutine to send frames with different first bytes
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(10 * time.Millisecond)
			frame := make([]byte, 640)
			frame[0] = byte(i)
			_ = client.SendScreenData("EVENT", frame)
		}
	}()

	// Wait for frame with first byte == 5
	frame, err := client.WaitForFrameMatching(func(f Frame) bool {
		return f.Data[0] == 5
	}, 500*time.Millisecond)

	if err != nil {
		t.Fatalf("WaitForFrameMatching should succeed: %v", err)
	}
	if frame == nil {
		t.Fatal("Should return matching frame")
	}
	if frame.Data[0] != 5 {
		t.Errorf("Expected frame with data[0]=5, got %d", frame.Data[0])
	}
}

func TestWaitForFrameMatching_Timeout(t *testing.T) {
	client := NewTestClient()

	// Send frames that don't match
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(10 * time.Millisecond)
			_ = client.SendScreenData("EVENT", make([]byte, 640))
		}
	}()

	// Wait for frame that will never exist
	frame, err := client.WaitForFrameMatching(func(f Frame) bool {
		return f.Data[0] == 255
	}, 100*time.Millisecond)

	if err == nil {
		t.Error("Should timeout")
	}
	if frame != nil {
		t.Error("Should not return frame on timeout")
	}
}

func TestWaitForNonBlankFrame(t *testing.T) {
	client := NewTestClient()

	// Send blank frame, then non-blank
	go func() {
		time.Sleep(10 * time.Millisecond)
		_ = client.SendScreenData("EVENT", make([]byte, 640)) // blank
		time.Sleep(10 * time.Millisecond)
		frame := make([]byte, 640)
		frame[0] = 0xFF // non-blank
		_ = client.SendScreenData("EVENT", frame)
	}()

	frame, err := client.WaitForNonBlankFrame(500 * time.Millisecond)
	if err != nil {
		t.Errorf("WaitForNonBlankFrame should succeed: %v", err)
	}
	if frame == nil || frame.Data[0] != 0xFF {
		t.Error("Should return non-blank frame")
	}
}

func TestTimingStats_String(t *testing.T) {
	stats := &TimingStats{
		FrameCount:    10,
		TotalDuration: 1 * time.Second,
		AvgInterval:   100 * time.Millisecond,
		MinInterval:   90 * time.Millisecond,
		MaxInterval:   110 * time.Millisecond,
		StdDev:        5 * time.Millisecond,
		AverageFPS:    10.0,
	}

	str := stats.String()
	if len(str) == 0 {
		t.Error("String() should return non-empty string")
	}
}

func TestSqrt_EdgeCases(t *testing.T) {
	// Test sqrt with negative and zero values
	// sqrt is an internal function, but we can test it through CalculateTimingStats
	// when standard deviation is calculated

	// Create a client and send frames with zero variance (identical intervals)
	// This will exercise the sqrt function with value close to or equal to zero
	client := NewTestClient()

	// We need enough frames to calculate std dev
	for i := 0; i < 10; i++ {
		_ = client.SendScreenData("EVENT", make([]byte, 640))
	}

	// This calculates timing stats including sqrt
	stats := client.CalculateTimingStats()

	// Just verify it doesn't panic and returns valid stats
	if stats.StdDev < 0 {
		t.Error("StdDev should not be negative")
	}

	// Test with exactly 0 frames to ensure sqrt handles zero
	client2 := NewTestClient()
	stats2 := client2.CalculateTimingStats()
	if stats2.StdDev != 0 {
		t.Error("StdDev should be 0 with no frames")
	}
}

func TestSqrt_Direct(t *testing.T) {
	// Direct test of sqrt function to cover all branches

	// Test zero
	result := sqrt(0)
	if result != 0 {
		t.Errorf("sqrt(0) should be 0, got %f", result)
	}

	// Test negative value
	result = sqrt(-1)
	if result != 0 {
		t.Errorf("sqrt(-1) should be 0, got %f", result)
	}

	// Test very negative value
	result = sqrt(-100)
	if result != 0 {
		t.Errorf("sqrt(-100) should be 0, got %f", result)
	}

	// Test positive values for correctness
	result = sqrt(4)
	if result < 1.99 || result > 2.01 {
		t.Errorf("sqrt(4) should be ~2, got %f", result)
	}

	result = sqrt(9)
	if result < 2.99 || result > 3.01 {
		t.Errorf("sqrt(9) should be ~3, got %f", result)
	}

	result = sqrt(2)
	if result < 1.41 || result > 1.42 {
		t.Errorf("sqrt(2) should be ~1.414, got %f", result)
	}
}
