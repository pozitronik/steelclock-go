package webeditor

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// ContextUsage represents token usage within the context window
type ContextUsage struct {
	InputTokens              int `json:"input_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	OutputTokens             int `json:"output_tokens,omitempty"`
}

// ContextWindow represents Claude Code's context window state
type ContextWindow struct {
	ContextWindowSize int          `json:"context_window_size,omitempty"`
	CurrentUsage      ContextUsage `json:"current_usage,omitempty"`
}

// ModelInfo represents the current model information
type ModelInfo struct {
	DisplayName string `json:"display_name,omitempty"`
}

// ClaudeStatus represents the status data from Claude Code
type ClaudeStatus struct {
	State       string    `json:"state"`
	Tool        string    `json:"tool,omitempty"`
	ToolPreview string    `json:"preview,omitempty"`
	Message     string    `json:"message,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Session     struct {
		StartedAt   time.Time `json:"started_at,omitempty"`
		ToolCalls   int       `json:"tool_calls,omitempty"`
		TokensUsed  int       `json:"tokens_used,omitempty"`  // Legacy field
		TokensLimit int       `json:"tokens_limit,omitempty"` // Legacy field
	} `json:"session,omitempty"`
	ContextWindow ContextWindow `json:"context_window,omitempty"`
	Model         ModelInfo     `json:"model,omitempty"`
}

// ClaudeStatusStore manages Claude Code status with thread-safe access
type ClaudeStatusStore struct {
	mu         sync.RWMutex
	status     *ClaudeStatus
	expiration time.Duration
}

// Global status store instance
var claudeStatusStore = &ClaudeStatusStore{
	expiration: 30 * time.Second,
}

// GetClaudeStatus returns the current Claude Code status (for widgets)
func GetClaudeStatus() *ClaudeStatus {
	return claudeStatusStore.Get()
}

// Get returns the current status, or nil if expired/not set
func (s *ClaudeStatusStore) Get() *ClaudeStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.status == nil {
		return nil
	}

	// Check if status is stale
	if time.Since(s.status.Timestamp) > s.expiration {
		return nil
	}

	// Return a copy to prevent race conditions
	statusCopy := *s.status
	return &statusCopy
}

// Set updates the current status
func (s *ClaudeStatusStore) Set(status *ClaudeStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set timestamp if not provided
	if status.Timestamp.IsZero() {
		status.Timestamp = time.Now()
	}

	s.status = status
}

// handleClaudeStatus handles GET (retrieve) and POST (update) for Claude Code status
func (s *Server) handleClaudeStatus(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getClaudeStatus(w)
	case http.MethodPost:
		s.setClaudeStatus(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getClaudeStatus returns current Claude Code status
func (s *Server) getClaudeStatus(w http.ResponseWriter) {
	status := claudeStatusStore.Get()
	if status == nil {
		// Return "not running" status when no data
		respondJSON(w, map[string]interface{}{
			"state":     "not_running",
			"timestamp": time.Now(),
		})
		return
	}

	respondJSON(w, status)
}

// setClaudeStatus updates Claude Code status
func (s *Server) setClaudeStatus(w http.ResponseWriter, r *http.Request) {
	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024) // 64KB max

	var status ClaudeStatus
	if err := json.NewDecoder(r.Body).Decode(&status); err != nil {
		respondError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate state
	validStates := map[string]bool{
		"not_running": true,
		"idle":        true,
		"thinking":    true,
		"tool":        true,
		"success":     true,
		"error":       true,
	}

	if !validStates[status.State] {
		respondError(w, "Invalid state. Valid states: not_running, idle, thinking, tool, success, error", http.StatusBadRequest)
		return
	}

	claudeStatusStore.Set(&status)

	respondJSON(w, map[string]interface{}{
		"success": true,
		"message": "Status updated",
	})
}
