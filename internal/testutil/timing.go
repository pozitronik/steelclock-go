package testutil

import (
	"fmt"
	"sort"
	"time"
)

// TimingStats provides statistical analysis of frame timing
type TimingStats struct {
	FrameCount     int
	TotalDuration  time.Duration
	MinInterval    time.Duration
	MaxInterval    time.Duration
	AvgInterval    time.Duration
	MedianInterval time.Duration
	StdDev         time.Duration
	Intervals      []time.Duration

	// Frame rate in FPS
	AverageFPS float64
	MinFPS     float64
	MaxFPS     float64
}

// CalculateTimingStats analyzes the timing of captured frames
func (c *TestClient) CalculateTimingStats() *TimingStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.frames) < 2 {
		return &TimingStats{FrameCount: len(c.frames)}
	}

	stats := &TimingStats{
		FrameCount: len(c.frames),
	}

	// Calculate intervals
	intervals := make([]time.Duration, len(c.frames)-1)
	for i := 1; i < len(c.frames); i++ {
		intervals[i-1] = c.frames[i].Timestamp.Sub(c.frames[i-1].Timestamp)
	}

	stats.Intervals = intervals
	stats.TotalDuration = c.frames[len(c.frames)-1].Timestamp.Sub(c.frames[0].Timestamp)

	// Find min/max
	stats.MinInterval = intervals[0]
	stats.MaxInterval = intervals[0]
	var totalNanos int64

	for _, interval := range intervals {
		if interval < stats.MinInterval {
			stats.MinInterval = interval
		}
		if interval > stats.MaxInterval {
			stats.MaxInterval = interval
		}
		totalNanos += interval.Nanoseconds()
	}

	// Calculate average
	stats.AvgInterval = time.Duration(totalNanos / int64(len(intervals)))

	// Calculate median
	sortedIntervals := make([]time.Duration, len(intervals))
	copy(sortedIntervals, intervals)
	sort.Slice(sortedIntervals, func(i, j int) bool {
		return sortedIntervals[i] < sortedIntervals[j]
	})

	mid := len(sortedIntervals) / 2
	if len(sortedIntervals)%2 == 0 {
		stats.MedianInterval = (sortedIntervals[mid-1] + sortedIntervals[mid]) / 2
	} else {
		stats.MedianInterval = sortedIntervals[mid]
	}

	// Calculate standard deviation
	var varianceSum float64
	avgNanos := float64(stats.AvgInterval.Nanoseconds())
	for _, interval := range intervals {
		diff := float64(interval.Nanoseconds()) - avgNanos
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(len(intervals))
	stats.StdDev = time.Duration(sqrt(variance))

	// Calculate FPS
	if stats.AvgInterval > 0 {
		stats.AverageFPS = float64(time.Second) / float64(stats.AvgInterval)
	}
	if stats.MaxInterval > 0 {
		stats.MinFPS = float64(time.Second) / float64(stats.MaxInterval)
	}
	if stats.MinInterval > 0 {
		stats.MaxFPS = float64(time.Second) / float64(stats.MinInterval)
	}

	return stats
}

// VerifyFrameRate checks if the average frame rate is within expected bounds
func (c *TestClient) VerifyFrameRate(expectedIntervalMs int, tolerancePercent float64) error {
	stats := c.CalculateTimingStats()

	if stats.FrameCount < 2 {
		return fmt.Errorf("not enough frames to calculate frame rate (have %d)", stats.FrameCount)
	}

	expectedInterval := time.Duration(expectedIntervalMs) * time.Millisecond
	tolerance := time.Duration(float64(expectedInterval) * tolerancePercent / 100)

	minAcceptable := expectedInterval - tolerance
	maxAcceptable := expectedInterval + tolerance

	if stats.AvgInterval < minAcceptable || stats.AvgInterval > maxAcceptable {
		return fmt.Errorf(
			"frame rate out of bounds: expected %v (±%.0f%%), got avg=%v (min=%v, max=%v)",
			expectedInterval, tolerancePercent, stats.AvgInterval, stats.MinInterval, stats.MaxInterval,
		)
	}

	return nil
}

// VerifyMinimumFrames checks that at least N frames were captured
func (c *TestClient) VerifyMinimumFrames(minFrames int) error {
	count := c.FrameCount()
	if count < minFrames {
		return fmt.Errorf("expected at least %d frames, got %d", minFrames, count)
	}
	return nil
}

// VerifyFrameCountInRange checks frame count is within expected range
func (c *TestClient) VerifyFrameCountInRange(min, max int) error {
	count := c.FrameCount()
	if count < min || count > max {
		return fmt.Errorf("expected %d-%d frames, got %d", min, max, count)
	}
	return nil
}

// WaitForFrames blocks until the specified number of frames are captured or timeout
func (c *TestClient) WaitForFrames(count int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c.FrameCount() >= count {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %d frames (got %d)", count, c.FrameCount())
}

// WaitForFrameMatching blocks until a frame matching the predicate is captured
func (c *TestClient) WaitForFrameMatching(predicate func(Frame) bool, timeout time.Duration) (*Frame, error) {
	deadline := time.Now().Add(timeout)
	lastChecked := 0

	for time.Now().Before(deadline) {
		frames := c.Frames()
		for i := lastChecked; i < len(frames); i++ {
			if predicate(frames[i]) {
				return &frames[i], nil
			}
		}
		lastChecked = len(frames)
		time.Sleep(10 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout waiting for matching frame")
}

// WaitForNonBlankFrame waits for a frame with at least one set pixel
func (c *TestClient) WaitForNonBlankFrame(timeout time.Duration) (*Frame, error) {
	return c.WaitForFrameMatching(func(f Frame) bool {
		return !IsBlankFrame(f.Data)
	}, timeout)
}

// String returns a human-readable summary of timing stats
func (s *TimingStats) String() string {
	return fmt.Sprintf(
		"Frames: %d, Duration: %v, Avg: %v (%.1f FPS), Min: %v, Max: %v, StdDev: %v",
		s.FrameCount, s.TotalDuration, s.AvgInterval, s.AverageFPS,
		s.MinInterval, s.MaxInterval, s.StdDev,
	)
}

// Simple sqrt implementation to avoid math import
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x / 2
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}
