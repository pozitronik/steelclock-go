// Package winamp provides communication with Winamp media player via Windows IPC messages.
package winamp

// PlaybackStatus represents Winamp playback state
type PlaybackStatus int

const (
	StatusStopped PlaybackStatus = 0
	StatusPlaying PlaybackStatus = 1
	StatusPaused  PlaybackStatus = 3
)

// String returns a human-readable status string
func (s PlaybackStatus) String() string {
	switch s {
	case StatusPlaying:
		return "Playing"
	case StatusPaused:
		return "Paused"
	default:
		return "Stopped"
	}
}

// TrackInfo contains information about the currently playing track
type TrackInfo struct {
	Title          string         // Track title from playlist
	FilePath       string         // Full file path
	FileName       string         // Filename without path
	PositionMs     int            // Current position in milliseconds
	DurationS      int            // Track duration in seconds
	Bitrate        int            // Audio bitrate in kbps
	SampleRate     int            // Sample rate in Hz
	Channels       int            // Number of audio channels
	Status         PlaybackStatus // Playback status
	TrackNumber    int            // Current track number in playlist (1-based)
	PlaylistLength int            // Total number of tracks in playlist
	Shuffle        bool           // Shuffle mode enabled
	Repeat         bool           // Repeat mode enabled
	Version        string         // Winamp version string
}

// Client provides an interface to communicate with Winamp
type Client interface {
	// IsRunning returns true if Winamp is running
	IsRunning() bool

	// GetStatus returns the current playback status
	GetStatus() PlaybackStatus

	// GetTrackInfo returns information about the current track
	// Returns nil if Winamp is not running or no track is loaded
	GetTrackInfo() *TrackInfo

	// GetCurrentTitle returns the title of the currently playing track
	// Returns empty string if not available
	GetCurrentTitle() string

	// GetCurrentPosition returns the current playback position in milliseconds
	// Returns -1 if not playing
	GetCurrentPosition() int

	// GetTrackDuration returns the track duration in seconds
	// Returns -1 if not available
	GetTrackDuration() int
}

// NewClient creates a new Winamp client
// Returns a platform-specific implementation
func NewClient() Client {
	return newPlatformClient()
}
