package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestProfileManager_SanitizeFilename(t *testing.T) {
	pm := NewProfileManager("/tmp")

	tests := []struct {
		name string
		want string
	}{
		{"My Profile", "my_profile"},
		{"Gaming Setup", "gaming_setup"},
		{"test-profile", "test-profile"},
		{"TEST_123", "test_123"},
		{"Profile@#$%!", "profile"},
		{"", "profile"},
		{"   ", "___"}, // spaces become underscores
		{"simple", "simple"},
		{"@#$%", "profile"}, // all special chars result in fallback
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pm.sanitizeFilename(tt.name)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestProfileManager_CreateProfile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create main config first
	writeConfig(t, tmpDir, MainConfigFile, "Main")

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	// Test creating a new profile
	path, err := pm.CreateProfile("My New Profile")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	// Verify the path is correct
	expectedPath := filepath.Join(tmpDir, ProfilesDir, "my_new_profile.json")
	if path != expectedPath {
		t.Errorf("Created profile path = %q, want %q", path, expectedPath)
	}

	// Verify the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("Profile file was not created")
	}

	// Read and parse file directly (not using Load which validates widgets)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read created profile: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Failed to parse created profile: %v", err)
	}

	if cfg.ConfigName != "My New Profile" {
		t.Errorf("ConfigName = %q, want %q", cfg.ConfigName, "My New Profile")
	}
	if cfg.GameName != DefaultGameName {
		t.Errorf("GameName = %q, want %q", cfg.GameName, DefaultGameName)
	}
	if len(cfg.Widgets) != 0 {
		t.Errorf("Widgets count = %d, want 0", len(cfg.Widgets))
	}

	// Verify the profile was added to the list
	profiles := pm.GetProfiles()
	var found bool
	for _, p := range profiles {
		if p.Name == "My New Profile" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Created profile not found in profiles list")
	}
}

func TestProfileManager_CreateProfile_EmptyName(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	pm := NewProfileManager(tmpDir)

	_, err := pm.CreateProfile("")
	if err == nil {
		t.Error("Expected error for empty profile name")
	}
}

func TestProfileManager_CreateProfile_Duplicate(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	// Create first profile
	_, err := pm.CreateProfile("Duplicate")
	if err != nil {
		t.Fatalf("First CreateProfile failed: %v", err)
	}

	// Try to create duplicate
	_, err = pm.CreateProfile("Duplicate")
	if err == nil {
		t.Error("Expected error for duplicate profile name")
	}
}

func TestSetTopLevelJSONField(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		value   string
		want    string
		wantErr bool
	}{
		{
			name:  "replaces existing value only",
			input: "{\n  \"$schema\": \"s.json\",\n  \"config_name\": \"Old\",\n  \"x\": {\"config_name\": \"nested\"}\n}\n",
			value: `"New"`,
			want:  "{\n  \"$schema\": \"s.json\",\n  \"config_name\": \"New\",\n  \"x\": {\"config_name\": \"nested\"}\n}\n",
		},
		{
			name:  "keeps CRLF and tabs",
			input: "{\r\n\t\"config_name\" :  \"Old\" ,\r\n\t\"a\": 1\r\n}",
			value: `"New"`,
			want:  "{\r\n\t\"config_name\" :  \"New\" ,\r\n\t\"a\": 1\r\n}",
		},
		{
			name:  "replaces non-string value",
			input: `{"config_name": null, "a": 1}`,
			value: `"New"`,
			want:  `{"config_name": "New", "a": 1}`,
		},
		{
			name:  "replaces every duplicate",
			input: `{"config_name": "First", "a": 1, "config_name": "Effective"}`,
			value: `"New"`,
			want:  `{"config_name": "New", "a": 1, "config_name": "New"}`,
		},
		{
			name:  "replaces case variants the loader also reads",
			input: `{"Config_Name": "Old", "a": 1}`,
			value: `"New"`,
			want:  `{"Config_Name": "New", "a": 1}`,
		},
		{
			name:  "ignores nested field with same name",
			input: "{\n  \"x\": {\"config_name\": \"nested\"}\n}",
			value: `"New"`,
			want:  "{\n  \"config_name\": \"New\",\n  \"x\": {\"config_name\": \"nested\"}\n}",
		},
		{
			name:  "inserts into compact object",
			input: `{"a":1}`,
			value: `"New"`,
			want:  `{"config_name": "New","a":1}`,
		},
		{
			name:  "inserts into empty object",
			input: `{ }`,
			value: `"New"`,
			want:  `{"config_name": "New" }`,
		},
		{name: "array document", input: `[1, 2]`, value: `"New"`, wantErr: true},
		{name: "null document", input: `null`, value: `"New"`, wantErr: true},
		{name: "invalid JSON", input: `{"a": }`, value: `"New"`, wantErr: true},
		{name: "unterminated object", input: `{"a": 1`, value: `"New"`, wantErr: true},
		{name: "empty input", input: ``, value: `"New"`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := setTopLevelJSONField([]byte(tt.input), "config_name", []byte(tt.value))
			if (err != nil) != tt.wantErr {
				t.Fatalf("setTopLevelJSONField() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if string(got) != tt.want {
				t.Errorf("setTopLevelJSONField() =\n%q\nwant\n%q", got, tt.want)
			}
			if !json.Valid(got) {
				t.Errorf("result is not valid JSON: %q", got)
			}
		})
	}
}

func TestRenameProfile_PreservesDocument(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := createProfilesDir(t, tmpDir)
	path := filepath.Join(profilesDir, "clock.json")
	original := "{\n" +
		"  \"$schema\": \"../profiles/.schema/config.schema.json\",\n" +
		"  \"config_name\": \"Old Name\",\n" +
		"  \"future_extension\": {\"keep\": true},\n" +
		"  \"widgets\": [{\"type\": \"clock\", \"position\": {\"x\": 0, \"y\": 0, \"w\": 128, \"h\": 40}}]\n" +
		"}\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles() error = %v", err)
	}

	newPath, err := pm.RenameProfile(path, `New "quoted" Name`)
	if err != nil {
		t.Fatalf("RenameProfile() error = %v", err)
	}
	if newPath != path {
		t.Errorf("RenameProfile() path = %q, want %q", newPath, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(original, `"Old Name"`, `"New \"quoted\" Name"`, 1)
	if string(data) != want {
		t.Errorf("file after rename =\n%s\nwant\n%s", data, want)
	}

	for _, p := range pm.GetAllProfiles() {
		if p.Path == path && p.Name != `New "quoted" Name` {
			t.Errorf("profile name = %q, want the new name", p.Name)
		}
	}
}

// TestRenameProfile_DuplicateName checks a file with config_name twice: the
// loader uses the last one, so the rename must change the name it reads.
func TestRenameProfile_DuplicateName(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := createProfilesDir(t, tmpDir)
	path := filepath.Join(profilesDir, "clock.json")
	content := `{"config_name": "First", ` +
		`"widgets": [{"type": "clock", "position": {"x": 0, "y": 0, "w": 128, "h": 40}}], ` +
		`"config_name": "Effective"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles() error = %v", err)
	}
	if _, err := pm.RenameProfile(path, "Renamed"); err != nil {
		t.Fatalf("RenameProfile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ConfigName != "Renamed" {
		t.Errorf("loaded name = %q, want Renamed", cfg.ConfigName)
	}
}

func TestRenameProfile_Errors(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := createProfilesDir(t, tmpDir)
	path := writeConfig(t, profilesDir, "clock.json", "Clock")

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles() error = %v", err)
	}

	tests := []struct {
		name    string
		path    string
		newName string
	}{
		{"empty name", path, ""},
		{"unknown profile", filepath.Join(profilesDir, "missing.json"), "Name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := pm.RenameProfile(tt.path, tt.newName); err == nil {
				t.Error("RenameProfile() error = nil, want error")
			}
		})
	}
}
