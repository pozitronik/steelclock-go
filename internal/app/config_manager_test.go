package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewConfigManager(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
	}{
		{"simple path", "config.json"},
		{"absolute path", "/path/to/config.json"},
		{"empty path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewConfigManager(tt.configPath)
			if mgr == nil {
				t.Fatal("NewConfigManager returned nil")
			}
			if mgr.configPath != tt.configPath {
				t.Errorf("configPath = %q, want %q", mgr.configPath, tt.configPath)
			}
			if mgr.profileMgr != nil {
				t.Error("profileMgr should be nil for direct config mode")
			}
		})
	}
}

func TestNewConfigManagerWithProfiles(t *testing.T) {
	// Test with nil profile manager
	mgr := NewConfigManagerWithProfiles(nil)
	if mgr == nil {
		t.Fatal("NewConfigManagerWithProfiles returned nil")
	}
	if mgr.profileMgr != nil {
		t.Error("profileMgr should be nil when passed nil")
	}

	// Test with real profile manager (using temp directory)
	tmpDir := t.TempDir()
	pm := config.NewProfileManager(tmpDir)
	mgr = NewConfigManagerWithProfiles(pm)
	if mgr == nil {
		t.Fatal("NewConfigManagerWithProfiles returned nil")
	}
	if mgr.profileMgr != pm {
		t.Error("profileMgr should match the provided ProfileManager")
	}
}

func TestConfigManager_HasProfiles(t *testing.T) {
	// Without profiles
	mgrDirect := NewConfigManager("config.json")
	if mgrDirect.HasProfiles() {
		t.Error("HasProfiles should return false for direct config mode")
	}

	// With profiles
	tmpDir := t.TempDir()
	pm := config.NewProfileManager(tmpDir)
	mgrProfiles := NewConfigManagerWithProfiles(pm)
	if !mgrProfiles.HasProfiles() {
		t.Error("HasProfiles should return true for profile mode")
	}

	// With nil profile manager
	mgrNil := NewConfigManagerWithProfiles(nil)
	if mgrNil.HasProfiles() {
		t.Error("HasProfiles should return false when profileMgr is nil")
	}
}

func TestConfigManager_GetProfileManager(t *testing.T) {
	// Direct mode - should return nil
	mgrDirect := NewConfigManager("config.json")
	if mgrDirect.GetProfileManager() != nil {
		t.Error("GetProfileManager should return nil for direct config mode")
	}

	// Profile mode
	tmpDir := t.TempDir()
	pm := config.NewProfileManager(tmpDir)
	mgrProfiles := NewConfigManagerWithProfiles(pm)
	if mgrProfiles.GetProfileManager() != pm {
		t.Error("GetProfileManager should return the profile manager")
	}
}

func TestConfigManager_GetConfigPath(t *testing.T) {
	// Direct mode
	expectedPath := "test/config.json"
	mgrDirect := NewConfigManager(expectedPath)
	if mgrDirect.GetConfigPath() != expectedPath {
		t.Errorf("GetConfigPath = %q, want %q", mgrDirect.GetConfigPath(), expectedPath)
	}

	// Profile mode without active profile
	tmpDir := t.TempDir()
	pm := config.NewProfileManager(tmpDir)
	mgrProfiles := NewConfigManagerWithProfiles(pm)
	// No profiles loaded, no active profile
	if mgrProfiles.GetConfigPath() != "" {
		t.Errorf("GetConfigPath should return empty string when no active profile, got %q", mgrProfiles.GetConfigPath())
	}
}

func TestConfigManager_GetActiveProfileName(t *testing.T) {
	// Direct mode - should return empty string
	mgrDirect := NewConfigManager("config.json")
	if mgrDirect.GetActiveProfileName() != "" {
		t.Errorf("GetActiveProfileName should return empty for direct mode, got %q", mgrDirect.GetActiveProfileName())
	}

	// Profile mode without active profile
	tmpDir := t.TempDir()
	pm := config.NewProfileManager(tmpDir)
	mgrProfiles := NewConfigManagerWithProfiles(pm)
	if mgrProfiles.GetActiveProfileName() != "" {
		t.Errorf("GetActiveProfileName should return empty when no active profile, got %q", mgrProfiles.GetActiveProfileName())
	}
}

func TestConfigManager_Load_DirectMode(t *testing.T) {
	// Test with non-existent file - may or may not error depending on platform
	mgr := NewConfigManager("non_existent_config_that_definitely_does_not_exist_12345.json")
	_, err := mgr.Load()
	// Just verify it doesn't panic - error behavior may vary
	_ = err
}

func TestConfigManager_Load_WithValidConfig(t *testing.T) {
	// Create a temporary valid config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{
		"game_name": "TEST",
		"game_display_name": "Test",
		"refresh_rate_ms": 100,
		"display": {
			"width": 128,
			"height": 40,
			"background": 0
		},
		"widgets": [
			{
				"type": "clock",
				"position": {"x": 0, "y": 0, "w": 128, "h": 40}
			}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	mgr := NewConfigManager(configPath)
	cfg, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.GameName != "TEST" {
		t.Errorf("GameName = %q, want %q", cfg.GameName, "TEST")
	}
}

func TestConfigManager_Reload_NoFile(t *testing.T) {
	mgr := NewConfigManager("non_existent_config.json")
	cfg, info, err := mgr.Reload()
	if err == nil {
		t.Error("Reload should return error for non-existent file")
	}
	if cfg != nil {
		t.Error("Config should be nil on error")
	}
	if info != nil {
		t.Error("FileInfo should be nil when file cannot be accessed")
	}
}

func TestConfigManager_Reload_ValidFile(t *testing.T) {
	// Create a temporary valid config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{
		"game_name": "RELOAD_TEST",
		"game_display_name": "Reload Test",
		"refresh_rate_ms": 100,
		"display": {
			"width": 128,
			"height": 40,
			"background": 0
		},
		"widgets": [
			{
				"type": "clock",
				"position": {"x": 0, "y": 0, "w": 128, "h": 40}
			}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	mgr := NewConfigManager(configPath)
	cfg, info, err := mgr.Reload()
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("Config should not be nil")
	}
	if cfg.GameName != "RELOAD_TEST" {
		t.Errorf("GameName = %q, want %q", cfg.GameName, "RELOAD_TEST")
	}
	if info == nil {
		t.Fatal("FileInfo should not be nil")
	}
	if info.Path != configPath {
		t.Errorf("FileInfo.Path = %q, want %q", info.Path, configPath)
	}
	if info.AbsolutePath == "" {
		t.Error("FileInfo.AbsolutePath should not be empty")
	}
	if info.ModTime == "" {
		t.Error("FileInfo.ModTime should not be empty")
	}
}

func TestConfigManager_Reload_InvalidJSON(t *testing.T) {
	// Create a temporary invalid config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	invalidContent := `{ invalid json }`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	mgr := NewConfigManager(configPath)
	cfg, info, err := mgr.Reload()
	if err == nil {
		t.Error("Reload should return error for invalid JSON")
	}
	if cfg != nil {
		t.Error("Config should be nil on error")
	}
	// File exists, so info should be returned
	if info == nil {
		t.Error("FileInfo should be returned even when config is invalid")
	}
}

func TestConfigManager_Reload_NoActiveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	pm := config.NewProfileManager(tmpDir)
	mgr := NewConfigManagerWithProfiles(pm)

	// No active profile - GetConfigPath returns ""
	_, _, err := mgr.Reload()
	if err == nil {
		t.Error("Reload should return error when no active profile")
	}
}

func TestConfigManager_SwitchProfile_NoProfileManager(t *testing.T) {
	mgr := NewConfigManager("config.json")
	_, err := mgr.SwitchProfile("some/path")
	if err == nil {
		t.Error("SwitchProfile should return error when not in profile mode")
	}
}

func TestConfigManager_SwitchProfile_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	pm := config.NewProfileManager(tmpDir)
	mgr := NewConfigManagerWithProfiles(pm)

	_, err := mgr.SwitchProfile("non_existent_profile.json")
	if err == nil {
		t.Error("SwitchProfile should return error for invalid profile path")
	}
}

func TestConfigManager_LogStartupInfo_DirectMode(t *testing.T) {
	mgr := NewConfigManager("test_config.json")
	// Should not panic
	mgr.LogStartupInfo()
}

func TestConfigManager_LogStartupInfo_ProfileMode(t *testing.T) {
	tmpDir := t.TempDir()
	pm := config.NewProfileManager(tmpDir)
	mgr := NewConfigManagerWithProfiles(pm)
	// Should not panic even without active profile
	mgr.LogStartupInfo()
}

func TestConfigManager_LogStartupInfo_ProfileModeWithActiveProfile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid config file
	configPath := filepath.Join(tmpDir, "steelclock.json")
	configContent := `{
		"game_name": "TEST",
		"game_display_name": "Test",
		"refresh_rate_ms": 100,
		"display": {"width": 128, "height": 40, "background": 0},
		"widgets": [{"type": "clock", "position": {"x": 0, "y": 0, "w": 128, "h": 40}}]
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	pm := config.NewProfileManager(tmpDir)
	_ = pm.LoadProfiles()
	mgr := NewConfigManagerWithProfiles(pm)

	// Should not panic with active profile
	mgr.LogStartupInfo()
}

func TestConfigFileInfo(t *testing.T) {
	info := &ConfigFileInfo{
		Path:         "config.json",
		AbsolutePath: "/full/path/config.json",
		ModTime:      "2025-01-01 12:00:00",
	}

	if info.Path != "config.json" {
		t.Errorf("Path = %q, want %q", info.Path, "config.json")
	}
	if info.AbsolutePath != "/full/path/config.json" {
		t.Errorf("AbsolutePath = %q, want %q", info.AbsolutePath, "/full/path/config.json")
	}
	if info.ModTime != "2025-01-01 12:00:00" {
		t.Errorf("ModTime = %q, want %q", info.ModTime, "2025-01-01 12:00:00")
	}
}
