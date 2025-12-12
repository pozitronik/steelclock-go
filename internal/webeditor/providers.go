package webeditor

import (
	"net/http"
	"time"
)

// ConfigProvider abstracts configuration file operations
type ConfigProvider interface {
	// GetConfigPath returns the path to the current configuration file
	GetConfigPath() string
	// Load reads and returns the current configuration as JSON bytes
	Load() ([]byte, error)
	// Save writes the configuration JSON to the file
	Save(data []byte) error
}

// PreviewProvider abstracts preview frame access
type PreviewProvider interface {
	// GetCurrentFrame returns the current frame data, frame number, and timestamp
	GetCurrentFrame() (data []byte, frameNum uint64, timestamp time.Time)
	// GetPreviewConfig returns the preview configuration (width, height, fps)
	GetPreviewConfig() PreviewDisplayConfig
	// HandleWebSocket handles a WebSocket connection for live preview
	HandleWebSocket(w http.ResponseWriter, r *http.Request)
}

// PreviewDisplayConfig contains display configuration for preview
type PreviewDisplayConfig struct {
	Width     int `json:"width"`
	Height    int `json:"height"`
	TargetFPS int `json:"target_fps"`
}

// ProfileProvider abstracts profile management operations
type ProfileProvider interface {
	// GetProfiles returns all available profiles
	GetProfiles() []ProfileInfo
	// GetActiveProfile returns the currently active profile, or nil if none
	GetActiveProfile() *ProfileInfo
	// SetActiveProfile switches to a different profile by path
	SetActiveProfile(path string) error
	// CreateProfile creates a new profile with the given name and returns its path
	CreateProfile(name string) (string, error)
	// RenameProfile renames a profile and returns its new path
	RenameProfile(oldPath, newName string) (string, error)
}

// ProfileInfo contains profile metadata for the API
type ProfileInfo struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	IsMain   bool   `json:"is_main"`
	IsActive bool   `json:"is_active"`
}
