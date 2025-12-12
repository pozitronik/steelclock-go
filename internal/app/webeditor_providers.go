package app

import (
	"os"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/webeditor"
)

// ConfigProviderAdapter adapts ConfigManager to webeditor.ConfigProvider interface
type ConfigProviderAdapter struct {
	configMgr *ConfigManager
}

// NewConfigProviderAdapter creates a new ConfigProviderAdapter
func NewConfigProviderAdapter(configMgr *ConfigManager) *ConfigProviderAdapter {
	return &ConfigProviderAdapter{configMgr: configMgr}
}

// GetConfigPath returns the path to the current configuration file
func (a *ConfigProviderAdapter) GetConfigPath() string {
	return a.configMgr.GetConfigPath()
}

// Load reads and returns the current configuration as JSON bytes
func (a *ConfigProviderAdapter) Load() ([]byte, error) {
	path := a.configMgr.GetConfigPath()
	return os.ReadFile(path)
}

// Save writes the configuration JSON to the file
func (a *ConfigProviderAdapter) Save(data []byte) error {
	path := a.configMgr.GetConfigPath()
	return os.WriteFile(path, data, 0644)
}

// ProfileProviderAdapter adapts ProfileManager to webeditor.ProfileProvider interface
type ProfileProviderAdapter struct {
	profileMgr *config.ProfileManager
}

// NewProfileProviderAdapter creates a new ProfileProviderAdapter
func NewProfileProviderAdapter(profileMgr *config.ProfileManager) *ProfileProviderAdapter {
	if profileMgr == nil {
		return nil
	}
	return &ProfileProviderAdapter{profileMgr: profileMgr}
}

// GetProfiles returns all available profiles
func (a *ProfileProviderAdapter) GetProfiles() []webeditor.ProfileInfo {
	if a.profileMgr == nil {
		return nil
	}

	profiles := a.profileMgr.GetProfiles()
	active := a.profileMgr.GetActiveProfile()

	result := make([]webeditor.ProfileInfo, len(profiles))
	for i, p := range profiles {
		result[i] = webeditor.ProfileInfo{
			Path:     p.Path,
			Name:     p.Name,
			IsMain:   p.IsMain,
			IsActive: active != nil && p.Path == active.Path,
		}
	}
	return result
}

// GetActiveProfile returns the currently active profile, or nil if none
func (a *ProfileProviderAdapter) GetActiveProfile() *webeditor.ProfileInfo {
	if a.profileMgr == nil {
		return nil
	}

	active := a.profileMgr.GetActiveProfile()
	if active == nil {
		return nil
	}

	return &webeditor.ProfileInfo{
		Path:     active.Path,
		Name:     active.Name,
		IsMain:   active.IsMain,
		IsActive: true,
	}
}

// SetActiveProfile switches to a different profile by path
func (a *ProfileProviderAdapter) SetActiveProfile(path string) error {
	if a.profileMgr == nil {
		return nil
	}
	return a.profileMgr.SetActiveProfile(path)
}

// CreateProfile creates a new profile with the given name and returns its path
func (a *ProfileProviderAdapter) CreateProfile(name string) (string, error) {
	if a.profileMgr == nil {
		return "", nil
	}
	return a.profileMgr.CreateProfile(name)
}

// RenameProfile renames a profile and returns its new path
func (a *ProfileProviderAdapter) RenameProfile(oldPath, newName string) (string, error) {
	if a.profileMgr == nil {
		return "", nil
	}
	return a.profileMgr.RenameProfile(oldPath, newName)
}
