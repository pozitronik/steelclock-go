//go:build !windows

package winamp

import "testing"

// TestStubClient_IsRunning verifies stub returns false for IsRunning
func TestStubClient_IsRunning(t *testing.T) {
	client := NewClient()
	if client.IsRunning() {
		t.Error("stubClient.IsRunning() = true on non-Windows, want false")
	}
}

// TestStubClient_GetStatus verifies stub returns StatusStopped
func TestStubClient_GetStatus(t *testing.T) {
	client := NewClient()
	status := client.GetStatus()
	if status != StatusStopped {
		t.Errorf("stubClient.GetStatus() = %v, want StatusStopped", status)
	}
}

// TestStubClient_GetCurrentTitle verifies stub returns empty string
func TestStubClient_GetCurrentTitle(t *testing.T) {
	client := NewClient()
	title := client.GetCurrentTitle()
	if title != "" {
		t.Errorf("stubClient.GetCurrentTitle() = %q, want empty string", title)
	}
}

// TestStubClient_GetCurrentPosition verifies stub returns -1
func TestStubClient_GetCurrentPosition(t *testing.T) {
	client := NewClient()
	pos := client.GetCurrentPosition()
	if pos != -1 {
		t.Errorf("stubClient.GetCurrentPosition() = %d, want -1", pos)
	}
}

// TestStubClient_GetTrackDuration verifies stub returns -1
func TestStubClient_GetTrackDuration(t *testing.T) {
	client := NewClient()
	duration := client.GetTrackDuration()
	if duration != -1 {
		t.Errorf("stubClient.GetTrackDuration() = %d, want -1", duration)
	}
}

// TestStubClient_GetTrackInfo verifies stub returns nil
func TestStubClient_GetTrackInfo(t *testing.T) {
	client := NewClient()
	info := client.GetTrackInfo()
	if info != nil {
		t.Error("stubClient.GetTrackInfo() != nil, want nil")
	}
}
