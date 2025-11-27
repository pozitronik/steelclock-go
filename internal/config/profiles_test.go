package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// testConfig is a minimal valid config JSON
const testConfig = `{
	"config_name": "%s",
	"game_name": "STEELCLOCK",
	"game_display_name": "SteelClock",
	"refresh_rate_ms": 100,
	"display": {"width": 128, "height": 40},
	"widgets": [{"type": "clock", "position": {"x": 0, "y": 0, "w": 128, "h": 40}}]
}`

// testConfigNoName is a config without config_name (uses filename as fallback)
const testConfigNoName = `{
	"game_name": "STEELCLOCK",
	"game_display_name": "SteelClock",
	"refresh_rate_ms": 100,
	"display": {"width": 128, "height": 40},
	"widgets": [{"type": "clock", "position": {"x": 0, "y": 0, "w": 128, "h": 40}}]
}`

// setupTestDir creates a temp directory and returns cleanup function
func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "steelclock-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tmpDir, func() { _ = os.RemoveAll(tmpDir) }
}

// writeConfig writes a config file with the given name
func writeConfig(t *testing.T, dir, filename, configName string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	var content string
	if configName == "" {
		content = testConfigNoName
	} else {
		content = fmt.Sprintf(testConfig, configName)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config %s: %v", filename, err)
	}
	return path
}

// createProfilesDir creates the profiles subdirectory
func createProfilesDir(t *testing.T, baseDir string) string {
	t.Helper()
	dir := filepath.Join(baseDir, ProfilesDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create profiles dir: %v", err)
	}
	return dir
}

func TestNewProfileManager(t *testing.T) {
	pm := NewProfileManager("/tmp/test")
	if pm == nil {
		t.Fatal("NewProfileManager returned nil")
	}
	if pm.baseDir != "/tmp/test" {
		t.Errorf("baseDir = %q, want %q", pm.baseDir, "/tmp/test")
	}
}

func TestProfileManager_LoadProfiles_NoConfigs(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	if len(pm.GetProfiles()) != 0 {
		t.Errorf("Expected 0 profiles, got %d", len(pm.GetProfiles()))
	}
}

func TestProfileManager_LoadProfiles_MainConfigOnly(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	writeConfig(t, tmpDir, MainConfigFile, "Main Profile")

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	profiles := pm.GetProfiles()
	if len(profiles) != 1 {
		t.Fatalf("Expected 1 profile, got %d", len(profiles))
	}

	if profiles[0].Name != "Main Profile" {
		t.Errorf("Profile name = %q, want %q", profiles[0].Name, "Main Profile")
	}
	if !profiles[0].IsMain {
		t.Error("Expected profile to be marked as main")
	}
}

func TestProfileManager_LoadProfiles_WithSubProfiles(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	writeConfig(t, tmpDir, MainConfigFile, "Default")

	profilesDir := createProfilesDir(t, tmpDir)
	writeConfig(t, profilesDir, "gaming.json", "Gaming")
	writeConfig(t, profilesDir, "work.json", "") // No config_name, uses filename

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	profiles := pm.GetProfiles()
	if len(profiles) != 3 {
		t.Fatalf("Expected 3 profiles, got %d", len(profiles))
	}

	// Main should be first
	if !profiles[0].IsMain {
		t.Error("First profile should be main")
	}
	if profiles[0].Name != "Default" {
		t.Errorf("First profile name = %q, want %q", profiles[0].Name, "Default")
	}

	// Check Gaming profile has config_name
	var found bool
	for _, p := range profiles {
		if p.Name == "Gaming" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Gaming profile not found")
	}

	// Check work profile uses filename (no config_name)
	found = false
	for _, p := range profiles {
		if p.Name == "work" {
			found = true
			break
		}
	}
	if !found {
		t.Error("work profile not found")
	}
}

func TestProfileManager_SetActiveProfile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	mainPath := writeConfig(t, tmpDir, MainConfigFile, "Main")

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	// Test setting active profile
	if err := pm.SetActiveProfile(mainPath); err != nil {
		t.Errorf("SetActiveProfile failed: %v", err)
	}

	active := pm.GetActiveProfile()
	if active == nil {
		t.Fatal("GetActiveProfile returned nil")
	}
	if active.Path != mainPath {
		t.Errorf("Active profile path = %q, want %q", active.Path, mainPath)
	}

	// Test setting non-existent profile
	if err := pm.SetActiveProfile("/nonexistent/path.json"); err == nil {
		t.Error("Expected error for non-existent profile")
	}
}

func TestProfileManager_GetActiveConfig(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	writeConfig(t, tmpDir, MainConfigFile, "Test Config")

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	cfg, err := pm.GetActiveConfig()
	if err != nil {
		t.Fatalf("GetActiveConfig failed: %v", err)
	}

	if cfg.GameName != "STEELCLOCK" {
		t.Errorf("GameName = %q, want %q", cfg.GameName, "STEELCLOCK")
	}
	if cfg.ConfigName != "Test Config" {
		t.Errorf("ConfigName = %q, want %q", cfg.ConfigName, "Test Config")
	}
}

func TestProfileManager_StateRestore(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	writeConfig(t, tmpDir, MainConfigFile, "Main")

	profilesDir := createProfilesDir(t, tmpDir)
	otherPath := writeConfig(t, profilesDir, "other.json", "Other")

	// First profile manager - set active to "Other"
	pm1 := NewProfileManager(tmpDir)
	if err := pm1.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	if err := pm1.SetActiveProfile(otherPath); err != nil {
		t.Fatalf("SetActiveProfile failed: %v", err)
	}

	// Verify state file was created
	statePath := filepath.Join(tmpDir, StateFile)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatal("State file was not created")
	}

	// Second profile manager - should restore "Other" as active
	pm2 := NewProfileManager(tmpDir)
	if err := pm2.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	active := pm2.GetActiveProfile()
	if active == nil {
		t.Fatal("No active profile after restore")
	}
	if active.Name != "Other" {
		t.Errorf("Restored profile = %q, want %q", active.Name, "Other")
	}
}

func TestProfileManager_HasMultipleProfiles(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	writeConfig(t, tmpDir, MainConfigFile, "")

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	if pm.HasMultipleProfiles() {
		t.Error("HasMultipleProfiles should be false with only main config")
	}

	// Add another profile
	profilesDir := createProfilesDir(t, tmpDir)
	writeConfig(t, profilesDir, "second.json", "")

	// Reload profiles
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	if !pm.HasMultipleProfiles() {
		t.Error("HasMultipleProfiles should be true with multiple configs")
	}
}

func TestProfile_FilenameToName(t *testing.T) {
	pm := NewProfileManager("/tmp")

	tests := []struct {
		path string
		want string
	}{
		{"/path/to/config.json", "config"},
		{"/path/to/my-profile.json", "my-profile"},
		{"/path/to/TEST.JSON", "TEST"},
		{"/path/to/no-extension", "no-extension"},
		{"simple.json", "simple"},
	}

	for _, tt := range tests {
		got := pm.filenameToName(tt.path)
		if got != tt.want {
			t.Errorf("filenameToName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
