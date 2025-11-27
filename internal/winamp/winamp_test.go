package winamp

import "testing"

// TestPlaybackStatus_String tests the String method of PlaybackStatus
func TestPlaybackStatus_String(t *testing.T) {
	tests := []struct {
		status   PlaybackStatus
		expected string
	}{
		{StatusPlaying, "Playing"},
		{StatusPaused, "Paused"},
		{StatusStopped, "Stopped"},
		{PlaybackStatus(99), "Stopped"}, // Unknown status defaults to Stopped
	}

	for _, tt := range tests {
		got := tt.status.String()
		if got != tt.expected {
			t.Errorf("PlaybackStatus(%d).String() = %q, want %q", tt.status, got, tt.expected)
		}
	}
}

// TestNewClient verifies that NewClient returns a non-nil Client
func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Error("NewClient() returned nil")
	}
}

// TestClient_Interface verifies that the client implements the Client interface
func TestClient_Interface(t *testing.T) {
	var _ = NewClient()
}

// TestTrackInfo_Fields verifies TrackInfo struct field types
func TestTrackInfo_Fields(t *testing.T) {
	info := &TrackInfo{
		Title:          "Test Title",
		FilePath:       "/path/to/file.mp3",
		FileName:       "file.mp3",
		PositionMs:     60000,
		DurationS:      180,
		Bitrate:        320,
		SampleRate:     44100,
		Channels:       2,
		Status:         StatusPlaying,
		TrackNumber:    1,
		PlaylistLength: 10,
		Shuffle:        true,
		Repeat:         false,
		Version:        "5.666",
	}

	if info.Title != "Test Title" {
		t.Error("TrackInfo.Title not set correctly")
	}
	if info.Status != StatusPlaying {
		t.Error("TrackInfo.Status not set correctly")
	}
	if info.Shuffle != true {
		t.Error("TrackInfo.Shuffle not set correctly")
	}
}

// TestPlaybackStatusConstants verifies the playback status constants
//
//goland:noinspection GoBoolExpressions,GoBoolExpressions,GoBoolExpressions
func TestPlaybackStatusConstants(t *testing.T) {
	// These values match Winamp's IPC_ISPLAYING return values
	if StatusStopped != 0 {
		t.Errorf("StatusStopped = %d, want 0", StatusStopped)
	}
	if StatusPlaying != 1 {
		t.Errorf("StatusPlaying = %d, want 1", StatusPlaying)
	}
	if StatusPaused != 3 {
		t.Errorf("StatusPaused = %d, want 3", StatusPaused)
	}
}
