package spotify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TokenInfo represents stored OAuth tokens.
type TokenInfo struct {
	// AccessToken is the OAuth access token.
	AccessToken string `json:"access_token"`
	// RefreshToken is the OAuth refresh token.
	RefreshToken string `json:"refresh_token"`
	// TokenType is the token type (usually "Bearer").
	TokenType string `json:"token_type"`
	// ExpiresAt is when the access token expires.
	ExpiresAt time.Time `json:"expires_at"`
	// Scope is the OAuth scope granted.
	Scope string `json:"scope"`
}

// IsExpired returns true if the access token has expired.
// Returns true if expires within 60 seconds (buffer for API calls).
func (t *TokenInfo) IsExpired() bool {
	if t == nil {
		return true
	}
	return time.Now().Add(60 * time.Second).After(t.ExpiresAt)
}

// IsValid returns true if the token info is valid and has a refresh token.
func (t *TokenInfo) IsValid() bool {
	if t == nil {
		return false
	}
	return t.AccessToken != "" && t.RefreshToken != ""
}

// TokenStore defines the interface for token persistence.
type TokenStore interface {
	// Load retrieves stored token info.
	// Returns nil, nil if no token is stored.
	Load() (*TokenInfo, error)
	// Save persists token info.
	Save(token *TokenInfo) error
	// Clear removes stored token info.
	Clear() error
	// Path returns the storage path.
	Path() string
}

// fileTokenStore implements TokenStore using a JSON file.
type fileTokenStore struct {
	path string
	mu   sync.RWMutex
}

// NewFileTokenStore creates a file-based token storage.
// If path is empty, uses default path relative to executable.
func NewFileTokenStore(path string) TokenStore {
	if path == "" {
		exePath, err := os.Executable()
		if err != nil {
			// Fallback to current directory
			path = DefaultTokenPath
		} else {
			path = filepath.Join(filepath.Dir(exePath), DefaultTokenPath)
		}
	}
	return &fileTokenStore{path: path}
}

// Load retrieves stored token info from the file.
func (s *fileTokenStore) Load() (*TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var token TokenInfo
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	return &token, nil
}

// Save persists token info to the file.
func (s *fileTokenStore) Save(token *TokenInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Write with restrictive permissions (owner read/write only)
	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// Clear removes the token file.
func (s *fileTokenStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.Remove(s.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token file: %w", err)
	}
	return nil
}

// Path returns the token storage path.
func (s *fileTokenStore) Path() string {
	return s.path
}

// memoryTokenStore implements TokenStore using in-memory storage (for testing).
type memoryTokenStore struct {
	token *TokenInfo
	mu    sync.RWMutex
}

// NewMemoryTokenStore creates an in-memory token storage (for testing).
func NewMemoryTokenStore() TokenStore {
	return &memoryTokenStore{}
}

// Load retrieves the stored token from memory.
func (s *memoryTokenStore) Load() (*TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.token == nil {
		return nil, nil
	}
	// Return a copy
	tokenCopy := *s.token
	return &tokenCopy, nil
}

// Save stores the token in memory.
func (s *memoryTokenStore) Save(token *TokenInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if token != nil {
		tokenCopy := *token
		s.token = &tokenCopy
	} else {
		s.token = nil
	}
	return nil
}

// Clear removes the stored token from memory.
func (s *memoryTokenStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.token = nil
	return nil
}

// Path returns empty string for memory store.
func (s *memoryTokenStore) Path() string {
	return ""
}
