// Package spotify provides a client for the Spotify Web API.
package spotify

import (
	"context"
	"time"
)

// PlaybackState represents Spotify playback state.
type PlaybackState int

const (
	// StateStopped indicates no track is playing or player is inactive.
	StateStopped PlaybackState = iota
	// StatePlaying indicates a track is currently playing.
	StatePlaying
	// StatePaused indicates playback is paused.
	StatePaused
)

// String returns string representation of PlaybackState.
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
	// ID is the Spotify track ID.
	ID string
	// Name is the track name.
	Name string
	// Artists contains artist names.
	Artists []string
	// Album is the album name.
	Album string
	// Duration is the total track duration.
	Duration time.Duration
	// Position is the current playback position.
	Position time.Duration
	// IsExplicit indicates explicit content.
	IsExplicit bool
}

// PlayerState contains the complete Spotify player state.
type PlayerState struct {
	// State is the current playback state.
	State PlaybackState
	// Track contains track metadata (nil when stopped or no track).
	Track *TrackInfo
	// DeviceName is the name of the active playback device.
	DeviceName string
	// Volume is the volume percentage (0-100).
	Volume int
	// ShuffleOn indicates if shuffle is enabled.
	ShuffleOn bool
	// RepeatMode is the repeat mode: "off", "context", or "track".
	RepeatMode string
}

// Client defines the interface for Spotify API communication.
type Client interface {
	// IsAuthenticated returns true if the client has valid tokens.
	IsAuthenticated() bool

	// IsAvailable checks if Spotify API is reachable and tokens are valid.
	IsAvailable() bool

	// GetState returns the current player state.
	// Returns nil state when no track is playing.
	GetState() (*PlayerState, error)

	// Connect initiates authentication (implements util.Connectable).
	Connect(ctx context.Context) error

	// IsConnected returns true if authenticated and API is accessible.
	IsConnected() bool

	// NeedsAuth returns true if authentication is required.
	NeedsAuth() bool

	// StartAuth initiates the OAuth flow (for oauth mode).
	// Returns error if auth mode is manual.
	StartAuth(ctx context.Context) error

	// RefreshToken attempts to refresh the access token.
	RefreshToken() error
}

// AuthMode determines the authentication method.
type AuthMode string

const (
	// AuthModeOAuth uses interactive OAuth PKCE flow.
	AuthModeOAuth AuthMode = "oauth"
	// AuthModeManual uses tokens provided in configuration.
	AuthModeManual AuthMode = "manual"
)

// ClientConfig contains configuration for creating a Spotify client.
type ClientConfig struct {
	// ClientID is the Spotify application Client ID (required).
	ClientID string
	// AuthMode is the authentication mode (oauth or manual).
	AuthMode AuthMode
	// AccessToken is the pre-obtained access token (for manual mode).
	AccessToken string
	// RefreshToken is the pre-obtained refresh token (for manual mode).
	RefreshToken string
	// TokenPath is the path to token storage file.
	TokenPath string
	// CallbackPort is the local port for OAuth callback server.
	CallbackPort int
}

// DefaultCallbackPort is the default port for OAuth callback server.
const DefaultCallbackPort = 8888

// DefaultTokenPath is the default token storage filename.
const DefaultTokenPath = "spotify_token.json"

// AuthURL is the Spotify authorization endpoint.
const AuthURL = "https://accounts.spotify.com/authorize"

// TokenURL is the Spotify token endpoint.
const TokenURL = "https://accounts.spotify.com/api/token"

// APIURL is the base URL for Spotify Web API.
const APIURL = "https://api.spotify.com/v1"

// RequiredScope is the OAuth scope required for current playback.
const RequiredScope = "user-read-currently-playing"
