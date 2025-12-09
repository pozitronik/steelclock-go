package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// ConfigManager handles configuration loading and profile management.
// It abstracts the source of configuration (direct file or profile-based).
type ConfigManager struct {
	configPath string
	profileMgr *config.ProfileManager
}

// NewConfigManager creates a ConfigManager for direct config file mode.
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// NewConfigManagerWithProfiles creates a ConfigManager with profile support.
func NewConfigManagerWithProfiles(profileMgr *config.ProfileManager) *ConfigManager {
	return &ConfigManager{
		profileMgr: profileMgr,
	}
}

// HasProfiles returns true if profile management is enabled.
func (m *ConfigManager) HasProfiles() bool {
	return m.profileMgr != nil
}

// GetProfileManager returns the profile manager, or nil if not in profile mode.
func (m *ConfigManager) GetProfileManager() *config.ProfileManager {
	return m.profileMgr
}

// GetConfigPath returns the path to the current configuration file.
// Returns empty string if no active profile in profile mode.
func (m *ConfigManager) GetConfigPath() string {
	if m.profileMgr != nil {
		activeProfile := m.profileMgr.GetActiveProfile()
		if activeProfile != nil {
			return activeProfile.Path
		}
		return ""
	}
	return m.configPath
}

// GetActiveProfileName returns the name of the active profile.
// Returns empty string if not in profile mode or no active profile.
func (m *ConfigManager) GetActiveProfileName() string {
	if m.profileMgr == nil {
		return ""
	}
	activeProfile := m.profileMgr.GetActiveProfile()
	if activeProfile == nil {
		return ""
	}
	return activeProfile.Name
}

// Load loads the configuration from the current source.
func (m *ConfigManager) Load() (*config.Config, error) {
	if m.profileMgr != nil {
		return m.profileMgr.GetActiveConfig()
	}
	return config.Load(m.configPath)
}

// Reload validates and loads a fresh configuration.
// Returns detailed file info for logging.
func (m *ConfigManager) Reload() (*config.Config, *ConfigFileInfo, error) {
	configPath := m.GetConfigPath()
	if configPath == "" {
		return nil, nil, fmt.Errorf("no active profile")
	}

	absPath, _ := filepath.Abs(configPath)

	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot access config file: %w", err)
	}

	info := &ConfigFileInfo{
		Path:         configPath,
		AbsolutePath: absPath,
		ModTime:      fileInfo.ModTime().Format("2006-01-02 15:04:05"),
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, info, err
	}

	return cfg, info, nil
}

// SwitchProfile switches to a different profile.
// Only valid in profile mode.
func (m *ConfigManager) SwitchProfile(path string) (*config.Config, error) {
	if m.profileMgr == nil {
		return nil, fmt.Errorf("profile manager not available")
	}

	if err := m.profileMgr.SetActiveProfile(path); err != nil {
		return nil, fmt.Errorf("failed to set active profile: %w", err)
	}

	cfg, err := m.profileMgr.GetActiveConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load profile config: %w", err)
	}

	return cfg, nil
}

// LogStartupInfo logs configuration information at startup.
func (m *ConfigManager) LogStartupInfo() {
	if m.profileMgr != nil {
		activeProfile := m.profileMgr.GetActiveProfile()
		if activeProfile != nil {
			log.Printf("Active profile: %s (%s)", activeProfile.Name, activeProfile.Path)
		}
		log.Printf("Available profiles: %d", len(m.profileMgr.GetProfiles()))
	} else {
		log.Printf("Config: %s", m.configPath)
	}
}

// ConfigFileInfo contains information about a config file.
type ConfigFileInfo struct {
	Path         string
	AbsolutePath string
	ModTime      string
}
