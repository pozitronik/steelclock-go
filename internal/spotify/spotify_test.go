package spotify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPlaybackState_String(t *testing.T) {
	tests := []struct {
		state PlaybackState
		want  string
	}{
		{StateStopped, "Stopped"},
		{StatePlaying, "Playing"},
		{StatePaused, "Paused"},
		{PlaybackState(99), "Stopped"}, // Unknown state
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("PlaybackState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenInfo_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		token     *TokenInfo
		wantExpir bool
	}{
		{
			name:      "nil token",
			token:     nil,
			wantExpir: true,
		},
		{
			name: "expired token",
			token: &TokenInfo{
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			wantExpir: true,
		},
		{
			name: "about to expire (within 60s buffer)",
			token: &TokenInfo{
				ExpiresAt: time.Now().Add(30 * time.Second),
			},
			wantExpir: true,
		},
		{
			name: "valid token",
			token: &TokenInfo{
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			wantExpir: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsExpired(); got != tt.wantExpir {
				t.Errorf("TokenInfo.IsExpired() = %v, want %v", got, tt.wantExpir)
			}
		})
	}
}

func TestTokenInfo_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		token *TokenInfo
		want  bool
	}{
		{
			name:  "nil token",
			token: nil,
			want:  false,
		},
		{
			name: "empty access token",
			token: &TokenInfo{
				AccessToken:  "",
				RefreshToken: "refresh",
			},
			want: false,
		},
		{
			name: "empty refresh token",
			token: &TokenInfo{
				AccessToken:  "access",
				RefreshToken: "",
			},
			want: false,
		},
		{
			name: "valid token",
			token: &TokenInfo{
				AccessToken:  "access",
				RefreshToken: "refresh",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsValid(); got != tt.want {
				t.Errorf("TokenInfo.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryTokenStore(t *testing.T) {
	store := NewMemoryTokenStore()

	// Test initial load returns nil
	token, err := store.Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}
	if token != nil {
		t.Errorf("Load() initial = %v, want nil", token)
	}

	// Test save and load
	testToken := &TokenInfo{
		AccessToken:  "test_access",
		RefreshToken: "test_refresh",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		Scope:        "user-read-currently-playing",
	}

	if err := store.Save(testToken); err != nil {
		t.Errorf("Save() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}
	if loaded == nil {
		t.Error("Load() returned nil after save")
	} else if loaded.AccessToken != testToken.AccessToken {
		t.Errorf("Load().AccessToken = %v, want %v", loaded.AccessToken, testToken.AccessToken)
	}

	// Test clear
	if err := store.Clear(); err != nil {
		t.Errorf("Clear() error = %v", err)
	}

	token, err = store.Load()
	if err != nil {
		t.Errorf("Load() after clear error = %v", err)
	}
	if token != nil {
		t.Errorf("Load() after clear = %v, want nil", token)
	}

	// Test Path returns empty for memory store
	if path := store.Path(); path != "" {
		t.Errorf("Path() = %v, want empty string", path)
	}
}

func TestFileTokenStore(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "test_token.json")

	store := NewFileTokenStore(tokenPath)

	// Test initial load (file doesn't exist)
	token, err := store.Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}
	if token != nil {
		t.Errorf("Load() initial = %v, want nil", token)
	}

	// Test save
	testToken := &TokenInfo{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour).Truncate(time.Second),
		Scope:        "user-read-currently-playing",
	}

	if err := store.Save(testToken); err != nil {
		t.Errorf("Save() error = %v", err)
	}

	// Verify file exists and has correct content
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Errorf("ReadFile() error = %v", err)
	}

	var savedToken TokenInfo
	if err := json.Unmarshal(data, &savedToken); err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
	}

	if savedToken.AccessToken != testToken.AccessToken {
		t.Errorf("Saved AccessToken = %v, want %v", savedToken.AccessToken, testToken.AccessToken)
	}

	// Test load
	loaded, err := store.Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}
	if loaded == nil {
		t.Error("Load() returned nil after save")
	} else {
		if loaded.AccessToken != testToken.AccessToken {
			t.Errorf("Load().AccessToken = %v, want %v", loaded.AccessToken, testToken.AccessToken)
		}
		if loaded.RefreshToken != testToken.RefreshToken {
			t.Errorf("Load().RefreshToken = %v, want %v", loaded.RefreshToken, testToken.RefreshToken)
		}
	}

	// Test clear
	if err := store.Clear(); err != nil {
		t.Errorf("Clear() error = %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Error("File should be deleted after Clear()")
	}

	// Test Path
	if path := store.Path(); path != tokenPath {
		t.Errorf("Path() = %v, want %v", path, tokenPath)
	}
}

func TestFileTokenStore_DefaultPath(t *testing.T) {
	store := NewFileTokenStore("")
	path := store.Path()

	if path == "" {
		t.Error("Path() with empty input should not return empty string")
	}

	if filepath.Base(path) != DefaultTokenPath {
		t.Errorf("Path() base = %v, want %v", filepath.Base(path), DefaultTokenPath)
	}
}

func TestAuthMode_Constants(t *testing.T) {
	if AuthModeOAuth != "oauth" {
		t.Errorf("AuthModeOAuth = %v, want oauth", AuthModeOAuth)
	}
	if AuthModeManual != "manual" {
		t.Errorf("AuthModeManual = %v, want manual", AuthModeManual)
	}
}

func TestConstants(t *testing.T) {
	if DefaultCallbackPort != 8888 {
		t.Errorf("DefaultCallbackPort = %v, want 8888", DefaultCallbackPort)
	}
	if DefaultTokenPath != "spotify_token.json" {
		t.Errorf("DefaultTokenPath = %v, want spotify_token.json", DefaultTokenPath)
	}
	if SpotifyAuthURL != "https://accounts.spotify.com/authorize" {
		t.Errorf("SpotifyAuthURL = %v", SpotifyAuthURL)
	}
	if SpotifyTokenURL != "https://accounts.spotify.com/api/token" {
		t.Errorf("SpotifyTokenURL = %v", SpotifyTokenURL)
	}
	if SpotifyAPIURL != "https://api.spotify.com/v1" {
		t.Errorf("SpotifyAPIURL = %v", SpotifyAPIURL)
	}
	if RequiredScope != "user-read-currently-playing" {
		t.Errorf("RequiredScope = %v", RequiredScope)
	}
}

func TestNewClient_MissingClientID(t *testing.T) {
	_, err := NewClient(&ClientConfig{
		ClientID: "",
	})
	if err == nil {
		t.Error("NewClient() with empty ClientID should return error")
	}
}

func TestNewClient_ManualModeWithoutToken(t *testing.T) {
	_, err := NewClient(&ClientConfig{
		ClientID: "test_client_id",
		AuthMode: AuthModeManual,
	})
	if err == nil {
		t.Error("NewClient() in manual mode without access_token should return error")
	}
}

func TestNewClient_ManualModeWithToken(t *testing.T) {
	client, err := NewClient(&ClientConfig{
		ClientID:     "test_client_id",
		AuthMode:     AuthModeManual,
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
	})
	if err != nil {
		t.Errorf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Error("NewClient() returned nil client")
	}

	// Check that client reports as authenticated
	if !client.IsAuthenticated() {
		t.Error("Client should be authenticated with manual tokens")
	}
}

func TestNewClient_OAuthModeDefaults(t *testing.T) {
	// Create temp directory for token storage
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "test_token.json")

	client, err := NewClient(&ClientConfig{
		ClientID:  "test_client_id",
		AuthMode:  AuthModeOAuth,
		TokenPath: tokenPath,
	})
	if err != nil {
		t.Errorf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Error("NewClient() returned nil client")
	}

	// Without saved tokens, should need auth
	if client.IsAuthenticated() {
		t.Error("Client should not be authenticated without tokens")
	}
	if !client.NeedsAuth() {
		t.Error("Client should need auth without tokens")
	}
}
