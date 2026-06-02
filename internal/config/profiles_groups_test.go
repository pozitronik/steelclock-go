package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeGroupConfig creates profiles/<group>/<filename> and returns its path.
func writeGroupConfig(t *testing.T, baseDir, group, filename, configName string) string {
	t.Helper()
	dir := filepath.Join(baseDir, ProfilesDir, group)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create group dir %s: %v", group, err)
	}
	return writeConfig(t, dir, filename, configName)
}

func groupNames(profiles []*Profile) []string {
	names := make([]string, len(profiles))
	for i, p := range profiles {
		names[i] = p.Name
	}
	return names
}

func TestProfileManager_Groups_Discovered(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	createProfilesDir(t, tmpDir)
	writeConfig(t, tmpDir, MainConfigFile, "Main") // -> General
	writeGroupConfig(t, tmpDir, "128x40", "clock.json", "Clock40")
	writeGroupConfig(t, tmpDir, "128x40", "cpu.json", "Cpu40")
	writeGroupConfig(t, tmpDir, "128x64", "clock.json", "Clock64")

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	// General (main config) first, then subdirectory groups alphabetically.
	gotGroups := pm.GetGroups()
	wantGroups := []string{DefaultGroupName, "128x40", "128x64"}
	if len(gotGroups) != len(wantGroups) {
		t.Fatalf("groups = %v, want %v", gotGroups, wantGroups)
	}
	for i := range wantGroups {
		if gotGroups[i] != wantGroups[i] {
			t.Errorf("groups[%d] = %q, want %q", i, gotGroups[i], wantGroups[i])
		}
	}

	if !pm.HasMultipleGroups() {
		t.Error("HasMultipleGroups() = false, want true")
	}

	// Default current group is the first one (General); only its profiles show.
	if pm.GetCurrentGroup() != DefaultGroupName {
		t.Errorf("current group = %q, want %q", pm.GetCurrentGroup(), DefaultGroupName)
	}
	if got := groupNames(pm.GetProfiles()); len(got) != 1 || got[0] != "Main" {
		t.Errorf("General profiles = %v, want [Main]", got)
	}
}

func TestProfileManager_SetCurrentGroup_Filters(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	createProfilesDir(t, tmpDir)
	writeGroupConfig(t, tmpDir, "128x40", "clock.json", "Clock40")
	writeGroupConfig(t, tmpDir, "128x64", "clock.json", "Clock64")
	writeGroupConfig(t, tmpDir, "128x64", "weather.json", "Weather64")

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	if err := pm.SetCurrentGroup("128x64"); err != nil {
		t.Fatalf("SetCurrentGroup failed: %v", err)
	}
	if pm.GetCurrentGroup() != "128x64" {
		t.Errorf("current group = %q, want 128x64", pm.GetCurrentGroup())
	}
	if got := len(pm.GetProfiles()); got != 2 {
		t.Errorf("128x64 profile count = %d, want 2", got)
	}
	if active := pm.GetActiveProfile(); active == nil || active.Group != "128x64" {
		t.Errorf("active profile group = %v, want 128x64", active)
	}

	if err := pm.SetCurrentGroup("does-not-exist"); err == nil {
		t.Error("SetCurrentGroup(nonexistent) should error")
	}
}

func TestProfileManager_GroupRemembersProfile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	createProfilesDir(t, tmpDir)
	c40 := writeGroupConfig(t, tmpDir, "128x40", "clock.json", "Clock40")
	writeGroupConfig(t, tmpDir, "128x40", "cpu.json", "Cpu40")
	writeGroupConfig(t, tmpDir, "128x64", "clock.json", "Clock64")

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	// In 128x40, pick a specific (non-first) profile.
	if err := pm.SetCurrentGroup("128x40"); err != nil {
		t.Fatalf("SetCurrentGroup failed: %v", err)
	}
	if err := pm.SetActiveProfile(c40); err != nil {
		t.Fatalf("SetActiveProfile failed: %v", err)
	}

	// Switch away, then back: the 128x40 choice should be remembered.
	if err := pm.SetCurrentGroup("128x64"); err != nil {
		t.Fatalf("SetCurrentGroup failed: %v", err)
	}
	if err := pm.SetCurrentGroup("128x40"); err != nil {
		t.Fatalf("SetCurrentGroup failed: %v", err)
	}
	if active := pm.GetActiveProfile(); active == nil || active.Path != c40 {
		t.Errorf("remembered profile = %v, want %s", active, c40)
	}
}

func TestProfileManager_StatePersistence_AcrossReload(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	createProfilesDir(t, tmpDir)
	writeGroupConfig(t, tmpDir, "128x40", "clock.json", "Clock40")
	c64 := writeGroupConfig(t, tmpDir, "128x64", "clock.json", "Clock64")

	pm1 := NewProfileManager(tmpDir)
	if err := pm1.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}
	if err := pm1.SetCurrentGroup("128x64"); err != nil {
		t.Fatalf("SetCurrentGroup failed: %v", err)
	}
	if err := pm1.SetActiveProfile(c64); err != nil {
		t.Fatalf("SetActiveProfile failed: %v", err)
	}

	// A fresh manager should restore the saved group + profile.
	pm2 := NewProfileManager(tmpDir)
	if err := pm2.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}
	if pm2.GetCurrentGroup() != "128x64" {
		t.Errorf("restored group = %q, want 128x64", pm2.GetCurrentGroup())
	}
	if active := pm2.GetActiveProfile(); active == nil || active.Path != c64 {
		t.Errorf("restored profile = %v, want %s", active, c64)
	}
}

func TestProfileManager_BackwardCompatState(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	createProfilesDir(t, tmpDir)
	writeGroupConfig(t, tmpDir, "128x40", "clock.json", "Clock40")
	c64 := writeGroupConfig(t, tmpDir, "128x64", "clock.json", "Clock64")

	// Old-format state: only active_profile_path, no group fields.
	oldState := `{"active_profile_path": ` + jsonString(c64) + `}`
	if err := os.WriteFile(filepath.Join(tmpDir, StateFile), []byte(oldState), 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}
	// The group should be derived from the remembered profile's directory.
	if pm.GetCurrentGroup() != "128x64" {
		t.Errorf("derived group = %q, want 128x64", pm.GetCurrentGroup())
	}
	if active := pm.GetActiveProfile(); active == nil || active.Path != c64 {
		t.Errorf("active profile = %v, want %s", active, c64)
	}
}

func TestProfileManager_NestedDirsIgnored(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	createProfilesDir(t, tmpDir)
	writeGroupConfig(t, tmpDir, "128x64", "clock.json", "Clock64")
	// A nested directory two levels deep must not become a group or a profile.
	nested := filepath.Join(tmpDir, ProfilesDir, "128x64", "extra")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	writeConfig(t, nested, "deep.json", "DeepProfile")

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}
	for _, g := range pm.GetGroups() {
		if g == "extra" {
			t.Error("nested directory 'extra' should not be a group")
		}
	}
	_ = pm.SetCurrentGroup("128x64")
	if got := len(pm.GetProfiles()); got != 1 {
		t.Errorf("128x64 profile count = %d, want 1 (nested ignored)", got)
	}
}

func TestProfileManager_CreateProfile_InCurrentGroup(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	createProfilesDir(t, tmpDir)
	writeGroupConfig(t, tmpDir, "128x64", "clock.json", "Clock64")

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}
	if err := pm.SetCurrentGroup("128x64"); err != nil {
		t.Fatalf("SetCurrentGroup failed: %v", err)
	}

	path, err := pm.CreateProfile("New One")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}
	wantDir := filepath.Join(tmpDir, ProfilesDir, "128x64")
	if filepath.Dir(path) != wantDir {
		t.Errorf("created profile dir = %q, want %q", filepath.Dir(path), wantDir)
	}
	// The new profile must be in the current group and visible there.
	found := false
	for _, p := range pm.GetProfiles() {
		if p.Path == path {
			found = true
			if p.Group != "128x64" {
				t.Errorf("new profile group = %q, want 128x64", p.Group)
			}
		}
	}
	if !found {
		t.Error("created profile not visible in current group")
	}
}

func TestProfileManager_DotDirsExcluded(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	createProfilesDir(t, tmpDir)
	writeGroupConfig(t, tmpDir, "128x40", "clock.json", "Clock40")
	// A dot-prefixed subdirectory (e.g. .schema) must not become a group, and
	// its files must not appear as profiles.
	writeGroupConfig(t, tmpDir, ".schema", "config.json", "Hidden")

	pm := NewProfileManager(tmpDir)
	if err := pm.LoadProfiles(); err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}

	for _, g := range pm.GetGroups() {
		if g == ".schema" {
			t.Errorf("dot-prefixed directory %q must not be a group", g)
		}
	}
	for _, p := range pm.GetAllProfiles() {
		if p.Group == ".schema" {
			t.Errorf("profile from a dot-prefixed directory leaked into groups: %s", p.Path)
		}
	}
}

// jsonString quotes a string as a JSON literal for embedding in test state.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
