// Package beefweb provides a client for the beefweb REST API
// used by Foobar2000 and DeaDBeeF music players.
package beefweb

import "time"

// PlaybackState represents the current player state.
type PlaybackState int

const (
	// StateStopped indicates playback is stopped.
	StateStopped PlaybackState = iota
	// StatePlaying indicates playback is active.
	StatePlaying
	// StatePaused indicates playback is paused.
	StatePaused
)

// String returns a human-readable state name.
func (s PlaybackState) String() string {
	switch s {
	case StatePlaying:
		return "Playing"
	case StatePaused:
		return "Paused"
	default:
		return "Stopped"
	}
}

// TrackInfo contains metadata for the currently playing track.
type TrackInfo struct {
	Artist   string
	Title    string
	Album    string
	Duration time.Duration
	Position time.Duration
	Index    int // Track index in playlist
}

// PlayerState contains the complete player state.
type PlayerState struct {
	State   PlaybackState
	Track   *TrackInfo // nil when stopped or no track loaded
	Volume  float64    // Normalized volume 0.0 to 1.0
	IsMuted bool
}

// Client defines the interface for beefweb API communication.
type Client interface {
	// IsAvailable checks if the beefweb server is reachable.
	IsAvailable() bool

	// GetState returns the current player state.
	// Returns nil state and error if server is unavailable.
	GetState() (*PlayerState, error)
}

// New creates a new beefweb HTTP client with the specified base URL.
// If baseURL is empty, defaults to http://localhost:8880.
func New(baseURL string) Client {
	return newHTTPClient(baseURL)
}
