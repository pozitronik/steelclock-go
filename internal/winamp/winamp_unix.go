//go:build !windows

package winamp

// stubClient is a no-op implementation for non-Windows platforms
type stubClient struct{}

func newPlatformClient() Client {
	return &stubClient{}
}

// IsRunning always returns false on non-Windows platforms
func (c *stubClient) IsRunning() bool {
	return false
}

// GetStatus always returns StatusStopped on non-Windows platforms
func (c *stubClient) GetStatus() PlaybackStatus {
	return StatusStopped
}

// GetCurrentTitle always returns empty string on non-Windows platforms
func (c *stubClient) GetCurrentTitle() string {
	return ""
}

// GetCurrentPosition always returns -1 on non-Windows platforms
func (c *stubClient) GetCurrentPosition() int {
	return -1
}

// GetTrackDuration always returns -1 on non-Windows platforms
func (c *stubClient) GetTrackDuration() int {
	return -1
}

// GetTrackInfo always returns nil on non-Windows platforms
func (c *stubClient) GetTrackInfo() *TrackInfo {
	return nil
}
