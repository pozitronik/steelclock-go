//go:build !darwin

package coreaudio

import "fmt"

// AudioTap is a stub for non-darwin platforms
type AudioTap struct{}

// NewAudioTap returns an error on non-darwin platforms
func NewAudioTap() (*AudioTap, error) {
	return nil, fmt.Errorf("CoreAudio tap is only supported on macOS")
}

// GetPeakLevels returns zero levels on unsupported platforms
func (t *AudioTap) GetPeakLevels() (left, right float64) { return 0, 0 }

// ReadSamples returns an error on unsupported platforms
func (t *AudioTap) ReadSamples() (left, right []float32, err error) {
	return nil, nil, fmt.Errorf("CoreAudio tap is only supported on macOS")
}

// GetRecentSamples returns nil on unsupported platforms
func (t *AudioTap) GetRecentSamples(count int) (left, right []float32) { return nil, nil }

// SampleRate returns 0 on unsupported platforms
func (t *AudioTap) SampleRate() int { return 0 }

// Channels returns 0 on unsupported platforms
func (t *AudioTap) Channels() int { return 0 }

// IsRunning returns false on unsupported platforms
func (t *AudioTap) IsRunning() bool { return false }

// Close is a no-op on unsupported platforms
func (t *AudioTap) Close() {}
