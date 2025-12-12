package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// MainConfigFile is the primary configuration file name
	MainConfigFile = "steelclock.json"
	// ProfilesDir is the directory containing additional profile configs
	ProfilesDir = "profiles"
	// StateFile stores the last active profile
	StateFile = ".steelclock.state"
)

// Profile represents a configuration profile
type Profile struct {
	Path       string // Full path to config file
	Name       string // Display name (from config_name or filename)
	IsMain     bool   // True if this is the main steelclock.json config
	LoadError  error  // Non-nil if config failed to load (for display purposes)
	loadedName string // Cached name from config file
}

// ProfileManager manages multiple configuration profiles
type ProfileManager struct {
	baseDir       string     // Directory containing steelclock.json
	profiles      []*Profile // All discovered profiles
	activeProfile *Profile   // Currently active profile
}

// appState stores persistent application state
type appState struct {
	ActiveProfilePath string `json:"active_profile_path"`
}

// NewProfileManager creates a new profile manager
// baseDir should be the directory containing steelclock.json
func NewProfileManager(baseDir string) *ProfileManager {
	return &ProfileManager{
		baseDir: baseDir,
	}
}

// LoadProfiles discovers and loads all available profiles
func (pm *ProfileManager) LoadProfiles() error {
	pm.profiles = nil

	// Load main config (steelclock.json)
	mainPath := filepath.Join(pm.baseDir, MainConfigFile)
	if _, err := os.Stat(mainPath); err == nil {
		profile := pm.loadProfile(mainPath, true)
		pm.profiles = append(pm.profiles, profile)
	}

	// Load profiles from profiles/ directory
	profilesPath := filepath.Join(pm.baseDir, ProfilesDir)
	if entries, err := os.ReadDir(profilesPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
				continue
			}

			profilePath := filepath.Join(profilesPath, entry.Name())
			profile := pm.loadProfile(profilePath, false)
			pm.profiles = append(pm.profiles, profile)
		}
	}

	// Sort profiles: main first, then alphabetically by name
	sort.Slice(pm.profiles, func(i, j int) bool {
		if pm.profiles[i].IsMain != pm.profiles[j].IsMain {
			return pm.profiles[i].IsMain // main config first
		}
		return pm.profiles[i].Name < pm.profiles[j].Name
	})

	// Restore last active profile or default to first available
	pm.restoreActiveProfile()

	return nil
}

// loadProfile loads a single profile from path
func (pm *ProfileManager) loadProfile(path string, isMain bool) *Profile {
	profile := &Profile{
		Path:   path,
		IsMain: isMain,
	}

	// Try to load config to get the name
	cfg, err := Load(path)
	if err != nil {
		profile.LoadError = err
		// Use filename as fallback name
		profile.Name = pm.filenameToName(path)
	} else {
		if cfg.ConfigName != "" {
			profile.Name = cfg.ConfigName
			profile.loadedName = cfg.ConfigName
		} else {
			profile.Name = pm.filenameToName(path)
		}
	}

	return profile
}

// filenameToName converts a file path to a display name
func (pm *ProfileManager) filenameToName(path string) string {
	name := filepath.Base(path)
	// Remove .json extension
	if strings.HasSuffix(strings.ToLower(name), ".json") {
		name = name[:len(name)-5]
	}
	return name
}

// GetProfiles returns all discovered profiles
func (pm *ProfileManager) GetProfiles() []*Profile {
	return pm.profiles
}

// GetActiveProfile returns the currently active profile
func (pm *ProfileManager) GetActiveProfile() *Profile {
	return pm.activeProfile
}

// SetActiveProfile switches to the specified profile by path
func (pm *ProfileManager) SetActiveProfile(path string) error {
	for _, p := range pm.profiles {
		if p.Path == path {
			pm.activeProfile = p
			pm.saveState()
			return nil
		}
	}
	return fmt.Errorf("profile not found: %s", path)
}

// SetActiveProfileByIndex switches to the profile at the specified index
func (pm *ProfileManager) SetActiveProfileByIndex(index int) error {
	if index < 0 || index >= len(pm.profiles) {
		return fmt.Errorf("profile index out of range: %d", index)
	}
	pm.activeProfile = pm.profiles[index]
	pm.saveState()
	return nil
}

// GetActiveConfig loads and returns the active profile's configuration
func (pm *ProfileManager) GetActiveConfig() (*Config, error) {
	if pm.activeProfile == nil {
		return nil, fmt.Errorf("no active profile")
	}
	return Load(pm.activeProfile.Path)
}

// restoreActiveProfile restores the last active profile from state file
func (pm *ProfileManager) restoreActiveProfile() {
	if len(pm.profiles) == 0 {
		return
	}

	// Try to load state
	state := pm.loadState()
	if state != nil && state.ActiveProfilePath != "" {
		// Find the profile by path
		for _, p := range pm.profiles {
			if p.Path == state.ActiveProfilePath {
				pm.activeProfile = p
				return
			}
		}
	}

	// Default to first profile (main config if available)
	pm.activeProfile = pm.profiles[0]
}

// loadState loads the application state from file
func (pm *ProfileManager) loadState() *appState {
	statePath := filepath.Join(pm.baseDir, StateFile)
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil
	}

	var state appState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}

	return &state
}

// saveState saves the current application state to file
func (pm *ProfileManager) saveState() {
	if pm.activeProfile == nil {
		return
	}

	state := appState{
		ActiveProfilePath: pm.activeProfile.Path,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}

	statePath := filepath.Join(pm.baseDir, StateFile)
	_ = os.WriteFile(statePath, data, 0644)
}

// RefreshProfile reloads a specific profile's metadata (name from config_name)
func (pm *ProfileManager) RefreshProfile(path string) {
	for _, p := range pm.profiles {
		if p.Path == path {
			cfg, err := Load(path)
			if err != nil {
				p.LoadError = err
			} else {
				p.LoadError = nil
				if cfg.ConfigName != "" {
					p.Name = cfg.ConfigName
					p.loadedName = cfg.ConfigName
				} else {
					p.Name = pm.filenameToName(path)
					p.loadedName = ""
				}
			}
			break
		}
	}
}

// HasMultipleProfiles returns true if there are multiple profiles available
func (pm *ProfileManager) HasMultipleProfiles() bool {
	return len(pm.profiles) > 1
}

// CreateProfile creates a new profile with the given name.
// Returns the path to the created profile file.
func (pm *ProfileManager) CreateProfile(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("profile name cannot be empty")
	}

	// Sanitize name for filename
	filename := pm.sanitizeFilename(name) + ".json"
	profilesPath := filepath.Join(pm.baseDir, ProfilesDir)

	// Ensure profiles directory exists
	if err := os.MkdirAll(profilesPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create profiles directory: %w", err)
	}

	// Build full path
	profilePath := filepath.Join(profilesPath, filename)

	// Check if file already exists
	if _, err := os.Stat(profilePath); err == nil {
		return "", fmt.Errorf("profile '%s' already exists", filename)
	}

	// Create minimal valid config with no widgets
	cfg := &Config{
		ConfigName:      name,
		GameName:        DefaultGameName,
		GameDisplayName: name,
		RefreshRateMs:   DefaultRefreshRateMs,
		Display: DisplayConfig{
			Width:  DefaultDisplayWidth,
			Height: DefaultDisplayHeight,
		},
		Widgets: []WidgetConfig{}, // Empty - user will add widgets
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(profilePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write profile: %w", err)
	}

	// Add to profiles list
	profile := &Profile{
		Path:       profilePath,
		Name:       name,
		IsMain:     false,
		loadedName: name,
	}
	pm.profiles = append(pm.profiles, profile)

	// Re-sort profiles
	sort.Slice(pm.profiles, func(i, j int) bool {
		if pm.profiles[i].IsMain != pm.profiles[j].IsMain {
			return pm.profiles[i].IsMain
		}
		return pm.profiles[i].Name < pm.profiles[j].Name
	})

	return profilePath, nil
}

// RenameProfile updates the config_name of a profile.
// Returns the profile path (unchanged).
func (pm *ProfileManager) RenameProfile(path, newName string) (string, error) {
	if newName == "" {
		return "", fmt.Errorf("profile name cannot be empty")
	}

	// Find the profile
	var profile *Profile
	for _, p := range pm.profiles {
		if p.Path == path {
			profile = p
			break
		}
	}

	if profile == nil {
		return "", fmt.Errorf("profile not found: %s", path)
	}

	// Load and update the config file
	cfg, err := Load(path)
	if err != nil {
		return "", fmt.Errorf("failed to load profile: %w", err)
	}

	cfg.ConfigName = newName

	// Marshal to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	// Save updated config
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save profile: %w", err)
	}

	// Update profile object
	profile.Name = newName
	profile.loadedName = newName

	// Re-sort profiles
	sort.Slice(pm.profiles, func(i, j int) bool {
		if pm.profiles[i].IsMain != pm.profiles[j].IsMain {
			return pm.profiles[i].IsMain
		}
		return pm.profiles[i].Name < pm.profiles[j].Name
	})

	return path, nil
}

// sanitizeFilename converts a profile name to a safe filename
func (pm *ProfileManager) sanitizeFilename(name string) string {
	// Replace spaces with underscores
	result := strings.ReplaceAll(name, " ", "_")

	// Remove or replace invalid characters
	var safe strings.Builder
	for _, r := range result {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' {
			safe.WriteRune(r)
		}
	}

	result = safe.String()
	if result == "" {
		result = "profile"
	}

	// Convert to lowercase for consistency
	return strings.ToLower(result)
}
