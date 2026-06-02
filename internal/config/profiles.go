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
	// DefaultGroupName is the group name for the main config and any profiles
	// directly in profiles/ (i.e. not inside a subdirectory). Subdirectories of
	// profiles/ form their own groups named after the directory.
	DefaultGroupName = "General"
)

// Profile represents a configuration profile
type Profile struct {
	Path       string // Full path to config file
	Name       string // Display name (from config_name or filename)
	Group      string // Group this profile belongs to (DefaultGroupName or a profiles/ subdirectory)
	IsMain     bool   // True if this is the main steelclock.json config
	LoadError  error  // Non-nil if config failed to load (for display purposes)
	loadedName string // Cached name from config file
}

// ProfileManager manages multiple configuration profiles, organized into groups.
// A group is either the implicit DefaultGroupName (the main config plus any
// profiles directly in profiles/) or a subdirectory of profiles/. Only the
// current group's profiles are surfaced via GetProfiles.
type ProfileManager struct {
	baseDir       string     // Directory containing steelclock.json
	profiles      []*Profile // All discovered profiles, across every group
	activeProfile *Profile   // Currently active profile
	groups        []string   // Ordered list of group names that contain profiles
	currentGroup  string     // Group whose profiles are currently surfaced
}

// appState stores persistent application state
type appState struct {
	ActiveProfilePath string `json:"active_profile_path"`
	// ActiveGroup is the group selected when the app last ran.
	ActiveGroup string `json:"active_group,omitempty"`
	// GroupProfiles remembers the last active profile path per group, so
	// switching back to a group restores its previous selection.
	GroupProfiles map[string]string `json:"group_profiles,omitempty"`
}

// NewProfileManager creates a new profile manager
// baseDir should be the directory containing steelclock.json
func NewProfileManager(baseDir string) *ProfileManager {
	return &ProfileManager{
		baseDir: baseDir,
	}
}

// LoadProfiles discovers and loads all available profiles, grouping the main
// config and any profiles directly in profiles/ under DefaultGroupName and each
// profiles/ subdirectory under a group named after the directory.
func (pm *ProfileManager) LoadProfiles() error {
	pm.profiles = nil
	pm.groups = nil

	// Load main config (steelclock.json) into the default group.
	mainPath := filepath.Join(pm.baseDir, MainConfigFile)
	if _, err := os.Stat(mainPath); err == nil {
		profile := pm.loadProfile(mainPath, true)
		profile.Group = DefaultGroupName
		pm.profiles = append(pm.profiles, profile)
	}

	// Load profiles directly in profiles/ into the default group, and each
	// subdirectory of profiles/ into a group named after the subdirectory.
	profilesPath := filepath.Join(pm.baseDir, ProfilesDir)
	if entries, err := os.ReadDir(profilesPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				pm.loadGroupDir(filepath.Join(profilesPath, entry.Name()), entry.Name())
				continue
			}
			if !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
				continue
			}
			profile := pm.loadProfile(filepath.Join(profilesPath, entry.Name()), false)
			profile.Group = DefaultGroupName
			pm.profiles = append(pm.profiles, profile)
		}
	}

	pm.sortProfiles()
	pm.rebuildGroups()
	pm.restoreActiveProfile()

	return nil
}

// loadGroupDir loads every .json profile directly inside dir into group.
// Nested subdirectories are not descended into (groups are a single level).
func (pm *ProfileManager) loadGroupDir(dir, group string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		profile := pm.loadProfile(filepath.Join(dir, entry.Name()), false)
		profile.Group = group
		pm.profiles = append(pm.profiles, profile)
	}
}

// sortProfiles orders profiles by group, then main config first, then by name.
func (pm *ProfileManager) sortProfiles() {
	sort.SliceStable(pm.profiles, func(i, j int) bool {
		a, b := pm.profiles[i], pm.profiles[j]
		if a.Group != b.Group {
			return a.Group < b.Group
		}
		if a.IsMain != b.IsMain {
			return a.IsMain // main config first within its group
		}
		return a.Name < b.Name
	})
}

// rebuildGroups recomputes the ordered group list from the loaded profiles,
// keeping DefaultGroupName first and the rest alphabetical.
func (pm *ProfileManager) rebuildGroups() {
	seen := make(map[string]bool)
	var rest []string
	hasDefault := false
	for _, p := range pm.profiles {
		if seen[p.Group] {
			continue
		}
		seen[p.Group] = true
		if p.Group == DefaultGroupName {
			hasDefault = true
		} else {
			rest = append(rest, p.Group)
		}
	}
	sort.Strings(rest)
	pm.groups = nil
	if hasDefault {
		pm.groups = append(pm.groups, DefaultGroupName)
	}
	pm.groups = append(pm.groups, rest...)
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

// GetProfiles returns the profiles in the current group.
func (pm *ProfileManager) GetProfiles() []*Profile {
	return pm.profilesInGroup(pm.currentGroup)
}

// GetAllProfiles returns every discovered profile across all groups, in display
// order (group, then main-first, then name). Used by the tray to pre-create a
// menu item per profile and show only the current group's.
func (pm *ProfileManager) GetAllProfiles() []*Profile {
	return pm.profiles
}

// GetProfilesInGroup returns the profiles belonging to the named group, in
// display order. Used by the tray to populate a per-group submenu.
func (pm *ProfileManager) GetProfilesInGroup(group string) []*Profile {
	return pm.profilesInGroup(group)
}

// profilesInGroup returns the profiles belonging to the named group.
func (pm *ProfileManager) profilesInGroup(group string) []*Profile {
	var result []*Profile
	for _, p := range pm.profiles {
		if p.Group == group {
			result = append(result, p)
		}
	}
	return result
}

// GetGroups returns the ordered list of group names that contain profiles.
func (pm *ProfileManager) GetGroups() []string {
	return pm.groups
}

// GetCurrentGroup returns the group whose profiles are currently surfaced.
func (pm *ProfileManager) GetCurrentGroup() string {
	return pm.currentGroup
}

// HasMultipleGroups reports whether more than one group is available.
func (pm *ProfileManager) HasMultipleGroups() bool {
	return len(pm.groups) > 1
}

// SetCurrentGroup switches to the named group and activates that group's
// remembered profile (or its first profile if none was remembered).
func (pm *ProfileManager) SetCurrentGroup(group string) error {
	profiles := pm.profilesInGroup(group)
	if len(profiles) == 0 {
		return fmt.Errorf("group not found or empty: %s", group)
	}
	pm.currentGroup = group

	// Restore the group's remembered profile if it still exists.
	if state := pm.loadState(); state != nil {
		if path, ok := state.GroupProfiles[group]; ok {
			for _, p := range profiles {
				if p.Path == path {
					pm.activeProfile = p
					pm.saveState()
					return nil
				}
			}
		}
	}

	pm.activeProfile = profiles[0]
	pm.saveState()
	return nil
}

// GetActiveProfile returns the currently active profile
func (pm *ProfileManager) GetActiveProfile() *Profile {
	return pm.activeProfile
}

// SetActiveProfile switches to the specified profile by path, making its group
// the current group.
func (pm *ProfileManager) SetActiveProfile(path string) error {
	for _, p := range pm.profiles {
		if p.Path == path {
			pm.activeProfile = p
			pm.currentGroup = p.Group
			pm.saveState()
			return nil
		}
	}
	return fmt.Errorf("profile not found: %s", path)
}

// SetActiveProfileByIndex switches to the profile at the specified index within
// the current group.
func (pm *ProfileManager) SetActiveProfileByIndex(index int) error {
	profiles := pm.GetProfiles()
	if index < 0 || index >= len(profiles) {
		return fmt.Errorf("profile index out of range: %d", index)
	}
	pm.activeProfile = profiles[index]
	pm.saveState()
	return nil
}

// groupExists reports whether a group with profiles exists.
func (pm *ProfileManager) groupExists(group string) bool {
	for _, g := range pm.groups {
		if g == group {
			return true
		}
	}
	return false
}

// profileByPathIn returns the profile with the given path from the slice, or nil.
func (pm *ProfileManager) profileByPathIn(profiles []*Profile, path string) *Profile {
	if path == "" {
		return nil
	}
	for _, p := range profiles {
		if p.Path == path {
			return p
		}
	}
	return nil
}

// GetActiveConfig loads and returns the active profile's configuration
func (pm *ProfileManager) GetActiveConfig() (*Config, error) {
	if pm.activeProfile == nil {
		return nil, fmt.Errorf("no active profile")
	}
	return Load(pm.activeProfile.Path)
}

// restoreActiveProfile restores the current group and active profile from the
// state file, falling back gracefully when the saved group/profile is gone.
func (pm *ProfileManager) restoreActiveProfile() {
	pm.currentGroup = ""
	pm.activeProfile = nil
	if len(pm.profiles) == 0 {
		return
	}

	state := pm.loadState()

	// Determine the current group: saved group, else the saved profile's group,
	// else the first available group.
	if state != nil {
		if pm.groupExists(state.ActiveGroup) {
			pm.currentGroup = state.ActiveGroup
		} else if p := pm.profileByPathIn(pm.profiles, state.ActiveProfilePath); p != nil {
			pm.currentGroup = p.Group
		}
	}
	if !pm.groupExists(pm.currentGroup) {
		pm.currentGroup = pm.groups[0]
	}

	// Determine the active profile within the current group: the group's
	// remembered profile, else the last overall active profile, else the first.
	groupProfiles := pm.profilesInGroup(pm.currentGroup)
	if state != nil {
		if p := pm.profileByPathIn(groupProfiles, state.GroupProfiles[pm.currentGroup]); p != nil {
			pm.activeProfile = p
			return
		}
		if p := pm.profileByPathIn(groupProfiles, state.ActiveProfilePath); p != nil {
			pm.activeProfile = p
			return
		}
	}
	pm.activeProfile = groupProfiles[0]
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

// saveState saves the active profile, current group, and per-group remembered
// profiles. Existing state is merged so other groups' selections are preserved.
func (pm *ProfileManager) saveState() {
	if pm.activeProfile == nil {
		return
	}

	state := pm.loadState()
	if state == nil {
		state = &appState{}
	}
	if state.GroupProfiles == nil {
		state.GroupProfiles = make(map[string]string)
	}
	state.ActiveProfilePath = pm.activeProfile.Path
	state.ActiveGroup = pm.currentGroup
	if pm.currentGroup != "" {
		state.GroupProfiles[pm.currentGroup] = pm.activeProfile.Path
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

// HasMultipleProfiles returns true if the current group has multiple profiles.
func (pm *ProfileManager) HasMultipleProfiles() bool {
	return len(pm.GetProfiles()) > 1
}

// groupDir returns the directory that holds a group's profile files: profiles/
// for the default group, or profiles/<group>/ for a subdirectory group.
func (pm *ProfileManager) groupDir(group string) string {
	base := filepath.Join(pm.baseDir, ProfilesDir)
	if group == "" || group == DefaultGroupName {
		return base
	}
	return filepath.Join(base, group)
}

// CreateProfile creates a new profile with the given name.
// Returns the path to the created profile file.
func (pm *ProfileManager) CreateProfile(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("profile name cannot be empty")
	}

	// Sanitize name for filename
	filename := pm.sanitizeFilename(name) + ".json"

	// New profiles are created in the current group's directory.
	group := pm.currentGroup
	if group == "" {
		group = DefaultGroupName
	}
	profilesPath := pm.groupDir(group)

	// Ensure the target directory exists
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

	// Add to profiles list, in the current group
	profile := &Profile{
		Path:       profilePath,
		Name:       name,
		Group:      group,
		IsMain:     false,
		loadedName: name,
	}
	pm.profiles = append(pm.profiles, profile)

	pm.sortProfiles()
	pm.rebuildGroups()

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

	pm.sortProfiles()

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
