package webeditor

// ConfigProvider abstracts configuration file operations
type ConfigProvider interface {
	// GetConfigPath returns the path to the current configuration file
	GetConfigPath() string
	// Load reads and returns the current configuration as JSON bytes
	Load() ([]byte, error)
	// Save writes the configuration JSON to the file
	Save(data []byte) error
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
}

// ProfileInfo contains profile metadata for the API
type ProfileInfo struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	IsMain   bool   `json:"is_main"`
	IsActive bool   `json:"is_active"`
}
