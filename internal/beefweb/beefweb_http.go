package beefweb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	defaultBaseURL       = "http://localhost:8880"
	defaultTimeout       = 2 * time.Second
	availabilityCacheTTL = 5 * time.Second
)

// httpClient implements the Client interface using HTTP requests.
type httpClient struct {
	baseURL    string
	httpClient *http.Client

	// Availability cache
	mu        sync.RWMutex
	available bool
	lastCheck time.Time
}

// playerResponse represents the JSON response from /api/player endpoint.
type playerResponse struct {
	Player struct {
		PlaybackState string `json:"playbackState"` // "stopped", "playing", "paused"
		ActiveItem    struct {
			Index    int      `json:"index"`
			Position float64  `json:"position"` // seconds
			Duration float64  `json:"duration"` // seconds
			Columns  []string `json:"columns"`  // requested column values
		} `json:"activeItem"`
		Volume struct {
			Value   float64 `json:"value"`
			Min     float64 `json:"min"`
			Max     float64 `json:"max"`
			IsMuted bool    `json:"isMuted"`
		} `json:"volume"`
	} `json:"player"`
}

// newHTTPClient creates a new HTTP-based beefweb client.
func newHTTPClient(baseURL string) *httpClient {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &httpClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// IsAvailable checks if the beefweb server is reachable.
// Results are cached for 5 seconds to avoid excessive requests.
func (c *httpClient) IsAvailable() bool {
	c.mu.RLock()
	if time.Since(c.lastCheck) < availabilityCacheTTL {
		available := c.available
		c.mu.RUnlock()
		return available
	}
	c.mu.RUnlock()

	// Need to check availability
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(c.lastCheck) < availabilityCacheTTL {
		return c.available
	}

	c.lastCheck = time.Now()
	resp, err := c.httpClient.Get(c.baseURL + "/api/player")
	if err != nil {
		c.available = false
		return false
	}
	_ = resp.Body.Close()
	c.available = resp.StatusCode == http.StatusOK
	return c.available
}

// GetState returns the current player state.
func (c *httpClient) GetState() (*PlayerState, error) {
	// Build URL with column queries
	// beefweb expects Foobar2000 title formatting syntax: %artist%, %title%, %album%
	// The % must be URL-encoded as %25 in the query string
	apiURL := c.baseURL + "/api/player?columns=%25artist%25,%25title%25,%25album%25"

	resp, err := c.httpClient.Get(apiURL)
	if err != nil {
		// Update availability cache on connection failure
		c.mu.Lock()
		c.available = false
		c.lastCheck = time.Now()
		c.mu.Unlock()
		return nil, fmt.Errorf("beefweb request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("beefweb returned status %d", resp.StatusCode)
	}

	var pr playerResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("beefweb decode failed: %w", err)
	}

	return c.parsePlayerResponse(&pr), nil
}

// parsePlayerResponse converts API response to PlayerState.
func (c *httpClient) parsePlayerResponse(pr *playerResponse) *PlayerState {
	state := &PlayerState{
		Volume:  normalizeVolume(pr.Player.Volume.Value, pr.Player.Volume.Min, pr.Player.Volume.Max),
		IsMuted: pr.Player.Volume.IsMuted,
	}

	switch pr.Player.PlaybackState {
	case "playing":
		state.State = StatePlaying
	case "paused":
		state.State = StatePaused
	default:
		state.State = StateStopped
	}

	// Parse track info if available
	if len(pr.Player.ActiveItem.Columns) >= 3 {
		state.Track = &TrackInfo{
			Artist:   pr.Player.ActiveItem.Columns[0],
			Title:    pr.Player.ActiveItem.Columns[1],
			Album:    pr.Player.ActiveItem.Columns[2],
			Duration: time.Duration(pr.Player.ActiveItem.Duration * float64(time.Second)),
			Position: time.Duration(pr.Player.ActiveItem.Position * float64(time.Second)),
			Index:    pr.Player.ActiveItem.Index,
		}
	}

	return state
}

// normalizeVolume converts volume from dB range to 0-1 scale.
func normalizeVolume(value, min, max float64) float64 {
	if max <= min {
		return 0
	}
	normalized := (value - min) / (max - min)
	if normalized < 0 {
		return 0
	}
	if normalized > 1 {
		return 1
	}
	return normalized
}
