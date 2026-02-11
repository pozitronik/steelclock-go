package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Error definitions.
var (
	// ErrNotAuthenticated indicates the client is not authenticated.
	ErrNotAuthenticated = errors.New("not authenticated")
	// ErrAuthRequired indicates authentication is required.
	ErrAuthRequired = errors.New("authentication required")
	// ErrTokenExpired indicates the access token has expired.
	ErrTokenExpired = errors.New("token expired")
	// ErrRefreshFailed indicates token refresh failed.
	ErrRefreshFailed = errors.New("token refresh failed")
	// ErrNoActiveDevice indicates no Spotify device is active.
	ErrNoActiveDevice = errors.New("no active device")
	// ErrRateLimited indicates the API rate limit was exceeded.
	ErrRateLimited = errors.New("rate limited")
)

// httpClient implements the Client interface using HTTP.
type httpClient struct {
	clientID     string
	tokenStore   TokenStore
	httpClient   *http.Client
	authMode     AuthMode
	callbackPort int

	token       *TokenInfo
	pkceAuth    *PKCEAuth
	mu          sync.RWMutex
	available   bool
	lastCheck   time.Time
	connected   bool
	authPending bool
}

// spotifyAPIResponse represents the currently playing response.
type spotifyAPIResponse struct {
	IsPlaying    bool   `json:"is_playing"`
	ProgressMS   int    `json:"progress_ms"`
	ShuffleState bool   `json:"shuffle_state"`
	RepeatState  string `json:"repeat_state"`
	Item         *struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		DurationMS int    `json:"duration_ms"`
		Explicit   bool   `json:"explicit"`
		Artists    []struct {
			Name string `json:"name"`
		} `json:"artists"`
		Album struct {
			Name string `json:"name"`
		} `json:"album"`
	} `json:"item"`
	Device *struct {
		Name          string `json:"name"`
		VolumePercent int    `json:"volume_percent"`
	} `json:"device"`
}

// NewClient creates a new Spotify client.
func NewClient(cfg *ClientConfig) (Client, error) {
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("client_id is required")
	}

	authMode := cfg.AuthMode
	if authMode == "" {
		authMode = AuthModeOAuth
	}

	callbackPort := cfg.CallbackPort
	if callbackPort == 0 {
		callbackPort = DefaultCallbackPort
	}

	tokenPath := cfg.TokenPath
	if tokenPath == "" {
		tokenPath = DefaultTokenPath
	}

	c := &httpClient{
		clientID:     cfg.ClientID,
		tokenStore:   NewFileTokenStore(tokenPath),
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		authMode:     authMode,
		callbackPort: callbackPort,
	}

	// For manual mode, create token from config
	if authMode == AuthModeManual {
		if cfg.AccessToken == "" {
			return nil, fmt.Errorf("access_token is required for manual mode")
		}
		c.token = &TokenInfo{
			AccessToken:  cfg.AccessToken,
			RefreshToken: cfg.RefreshToken,
			TokenType:    "Bearer",
			ExpiresAt:    time.Now().Add(1 * time.Hour), // Assume 1 hour validity
			Scope:        RequiredScope,
		}
	} else {
		// Try to load existing token
		token, err := c.tokenStore.Load()
		if err != nil {
			log.Printf("spotify: failed to load token: %v", err)
		} else {
			c.token = token
		}
	}

	return c, nil
}

// IsAuthenticated returns true if the client has valid tokens.
func (c *httpClient) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.token != nil && c.token.IsValid()
}

// IsAvailable checks if Spotify API is reachable.
func (c *httpClient) IsAvailable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Cache availability check for 5 seconds
	if time.Since(c.lastCheck) < 5*time.Second {
		return c.available
	}

	return c.checkAvailability()
}

// checkAvailability performs the actual availability check (must hold lock).
func (c *httpClient) checkAvailability() bool {
	c.mu.RUnlock()
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		c.mu.RLock()
	}()

	c.lastCheck = time.Now()

	if c.token == nil || !c.token.IsValid() {
		c.available = false
		return false
	}

	// Try to refresh if expired
	if c.token.IsExpired() {
		if err := c.refreshTokenInternal(); err != nil {
			c.available = false
			return false
		}
	}

	c.available = true
	return true
}

// GetState returns the current player state.
func (c *httpClient) GetState() (*PlayerState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token == nil || !c.token.IsValid() {
		return nil, ErrNotAuthenticated
	}

	// Refresh token if expired
	if c.token.IsExpired() {
		if err := c.refreshTokenInternal(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrRefreshFailed, err)
		}
	}

	// Make API request
	req, err := http.NewRequest("GET", SpotifyAPIURL+"/me/player/currently-playing", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle response codes
	switch resp.StatusCode {
	case http.StatusNoContent:
		// No track currently playing
		return &PlayerState{State: StateStopped}, nil

	case http.StatusOK:
		// Parse response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		var apiResp spotifyAPIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		return c.parseAPIResponse(&apiResp), nil

	case http.StatusUnauthorized:
		// Token expired or invalid
		c.token = nil
		return nil, ErrAuthRequired

	case http.StatusTooManyRequests:
		return nil, ErrRateLimited

	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
}

// parseAPIResponse converts API response to PlayerState.
func (c *httpClient) parseAPIResponse(resp *spotifyAPIResponse) *PlayerState {
	state := &PlayerState{
		ShuffleOn:  resp.ShuffleState,
		RepeatMode: resp.RepeatState,
	}

	if resp.IsPlaying {
		state.State = StatePlaying
	} else {
		state.State = StatePaused
	}

	if resp.Device != nil {
		state.DeviceName = resp.Device.Name
		state.Volume = resp.Device.VolumePercent
	}

	if resp.Item != nil {
		artists := make([]string, len(resp.Item.Artists))
		for i, a := range resp.Item.Artists {
			artists[i] = a.Name
		}

		state.Track = &TrackInfo{
			ID:         resp.Item.ID,
			Name:       resp.Item.Name,
			Artists:    artists,
			Album:      resp.Item.Album.Name,
			Duration:   time.Duration(resp.Item.DurationMS) * time.Millisecond,
			Position:   time.Duration(resp.ProgressMS) * time.Millisecond,
			IsExplicit: resp.Item.Explicit,
		}
	}

	return state
}

// Connect initiates authentication.
func (c *httpClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If already authenticated, just verify
	if c.token != nil && c.token.IsValid() {
		if c.token.IsExpired() {
			return c.refreshTokenInternal()
		}
		c.connected = true
		return nil
	}

	// For oauth mode, start auth flow
	if c.authMode == AuthModeOAuth {
		c.authPending = true
		return ErrAuthRequired
	}

	return ErrNotAuthenticated
}

// IsConnected returns true if authenticated and connected.
func (c *httpClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.connected && c.token != nil && c.token.IsValid()
}

// NeedsAuth returns true if authentication is required.
func (c *httpClient) NeedsAuth() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.token == nil || !c.token.IsValid() || c.authPending
}

// StartAuth initiates the OAuth PKCE flow.
func (c *httpClient) StartAuth(ctx context.Context) error {
	// Check mode and set pending flag with lock
	c.mu.Lock()
	if c.authMode != AuthModeOAuth {
		c.mu.Unlock()
		return fmt.Errorf("StartAuth only available in oauth mode")
	}

	// Mark auth as pending and create PKCE handler
	c.authPending = true
	c.pkceAuth = NewPKCEAuth(c.clientID, c.callbackPort)
	c.mu.Unlock()

	// Start auth flow WITHOUT holding the lock (this may take up to 5 minutes)
	token, err := c.pkceAuth.StartAuth(ctx)

	// Reacquire lock to update state
	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		c.authPending = false
		return fmt.Errorf("OAuth flow failed: %w", err)
	}

	// Save token
	c.token = token
	if err := c.tokenStore.Save(token); err != nil {
		log.Printf("spotify: failed to save token: %v", err)
	}

	c.connected = true
	c.authPending = false
	return nil
}

// RefreshToken attempts to refresh the access token.
func (c *httpClient) RefreshToken() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.refreshTokenInternal()
}

// refreshTokenInternal refreshes the token (must hold lock).
func (c *httpClient) refreshTokenInternal() error {
	if c.token == nil || c.token.RefreshToken == "" {
		return ErrNotAuthenticated
	}

	// Prepare request
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", c.token.RefreshToken)
	data.Set("client_id", c.clientID)

	req, err := http.NewRequest("POST", SpotifyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Clear invalid token
		c.token = nil
		_ = c.tokenStore.Clear()
		return fmt.Errorf("refresh failed (%d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	// Update token
	c.token.AccessToken = tokenResp.AccessToken
	c.token.TokenType = tokenResp.TokenType
	c.token.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	if tokenResp.RefreshToken != "" {
		c.token.RefreshToken = tokenResp.RefreshToken
	}
	if tokenResp.Scope != "" {
		c.token.Scope = tokenResp.Scope
	}

	// Save updated token
	if err := c.tokenStore.Save(c.token); err != nil {
		log.Printf("spotify: failed to save refreshed token: %v", err)
	}

	return nil
}
