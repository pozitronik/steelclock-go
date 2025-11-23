package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")

	configJSON := `{
		"game_name": "STEELCLOCK",
		"game_display_name": "Steel Clock",
		"refresh_rate_ms": 100,
		"display": {
			"width": 128,
			"height": 40,
			"dithering": true
		},
		"widgets": [
			{
				"type": "clock",
				"id": "main_clock",
				"enabled": true,
				"position": {
					"x": 0,
					"y": 0,
					"w": 128,
					"h": 40
				},
				"style": {
					"background_color": 0,
					"border": false,
					"border_color": 255
				},
				"properties": {
					"format": "15:04",
					"font": "",
					"font_size": 16,
					"horizontal_align": "center",
					"vertical_align": "center"
				}
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify loaded values
	if cfg.GameName != "STEELCLOCK" {
		t.Errorf("GameName = %s, want STEELCLOCK", cfg.GameName)
	}

	if cfg.GameDisplayName != "Steel Clock" {
		t.Errorf("GameDisplayName = %s, want Steel Clock", cfg.GameDisplayName)
	}

	if cfg.RefreshRateMs != 100 {
		t.Errorf("RefreshRateMs = %d, want 100", cfg.RefreshRateMs)
	}

	if cfg.Display.Width != 128 {
		t.Errorf("Display.Width = %d, want 128", cfg.Display.Width)
	}

	if cfg.Display.Height != 40 {
		t.Errorf("Display.Height = %d, want 40", cfg.Display.Height)
	}

	if len(cfg.Widgets) != 1 {
		t.Fatalf("len(Widgets) = %d, want 1", len(cfg.Widgets))
	}

	widget := cfg.Widgets[0]
	if widget.Type != "clock" {
		t.Errorf("Widget.Type = %s, want clock", widget.Type)
	}

	if widget.ID != "main_clock" {
		t.Errorf("Widget.ID = %s, want main_clock", widget.ID)
	}

	if !widget.IsEnabled() {
		t.Error("Widget.Enabled = false, want true")
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	// When config file doesn't exist, should return default config
	cfg, err := Load("/nonexistent/config.json")
	if err != nil {
		t.Errorf("Load() with nonexistent file should return default config, got error: %v", err)
	}

	// Verify we got a valid default config
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	if cfg.GameName != "STEELCLOCK" {
		t.Errorf("Default GameName = %s, want STEELCLOCK", cfg.GameName)
	}

	if cfg.GameDisplayName != "SteelClock" {
		t.Errorf("Default GameDisplayName = %s, want SteelClock", cfg.GameDisplayName)
	}

	if len(cfg.Widgets) == 0 {
		t.Error("Default config should have at least one widget")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(configPath, []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = Load(configPath)
	if err == nil {
		t.Error("Load() with invalid JSON should return error")
	}
}

func TestSaveDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "default_config.json")

	// Save default config
	err := SaveDefault(configPath)
	if err != nil {
		t.Fatalf("SaveDefault() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("SaveDefault() did not create config file")
	}

	// Load the saved config and verify it's valid
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved default config: %v", err)
	}

	if cfg.GameName != "STEELCLOCK" {
		t.Errorf("SavedDefault GameName = %s, want STEELCLOCK", cfg.GameName)
	}

	if len(cfg.Widgets) == 0 {
		t.Error("SavedDefault config should have at least one widget")
	}
}

// TestSaveDefault_CreatesParentDir tests that SaveDefault creates parent directories
func TestSaveDefault_CreatesParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir1", "subdir2", "config.json")

	err := SaveDefault(configPath)
	if err != nil {
		t.Fatalf("SaveDefault() should create parent directories, got error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("SaveDefault() did not create config file in nested directories")
	}
}

// TestSaveDefault_InvalidPath tests error handling for invalid paths
func TestSaveDefault_InvalidPath(t *testing.T) {
	// Create a regular file, then try to create a config "inside" it
	// This should fail on all platforms since you can't create a file inside a file
	tmpDir := t.TempDir()
	blockingFile := filepath.Join(tmpDir, "blockingfile")

	// Create a regular file
	if err := os.WriteFile(blockingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create blocking file: %v", err)
	}

	// Try to save to a path that treats the file as a directory
	// This should fail because blockingFile is a file, not a directory
	configPath := filepath.Join(blockingFile, "config.json")

	err := SaveDefault(configPath)
	if err == nil {
		t.Error("SaveDefault() should return error for invalid path")
	}
}

// TestValidateConfig_MissingGameName tests validation of missing game_name
func TestValidateConfig_MissingGameName(t *testing.T) {
	cfg := &Config{
		GameDisplayName: "Test",
		Display: DisplayConfig{
			Width:  128,
			Height: 40,
		},
		RefreshRateMs: 100,
		Widgets: []WidgetConfig{
			{
				Type:     "clock",
				ID:       "test",
				Enabled:  BoolPtr(true),
				Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Properties: WidgetProperties{
					Format: "%H:%M:%S",
				},
			},
		},
	}

	// Apply defaults before validation
	applyDefaults(cfg)

	// Validation should now succeed since defaults are applied
	err := validateConfig(cfg)
	if err != nil {
		t.Errorf("validateConfig() should succeed after applying defaults, got error: %v", err)
	}

	// Verify default was applied
	if cfg.GameName != "STEELCLOCK" {
		t.Errorf("Default GameName = %s, want STEELCLOCK", cfg.GameName)
	}
}

// TestValidateConfig_MissingGameDisplayName tests that missing game_display_name gets default
func TestValidateConfig_MissingGameDisplayName(t *testing.T) {
	cfg := &Config{
		GameName: "TEST",
		Display: DisplayConfig{
			Width:  128,
			Height: 40,
		},
		RefreshRateMs: 100,
		Widgets: []WidgetConfig{
			{
				Type:     "clock",
				ID:       "test",
				Enabled:  BoolPtr(true),
				Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Properties: WidgetProperties{
					Format: "%H:%M:%S",
				},
			},
		},
	}

	// Apply defaults before validation
	applyDefaults(cfg)

	// Validation should now succeed since defaults are applied
	err := validateConfig(cfg)
	if err != nil {
		t.Errorf("validateConfig() should succeed after applying defaults, got error: %v", err)
	}

	// Verify default was applied
	if cfg.GameDisplayName != "SteelClock" {
		t.Errorf("Default GameDisplayName = %s, want SteelClock", cfg.GameDisplayName)
	}
}

// TestValidateConfig_InvalidDisplayDimensions tests validation of display dimensions
func TestValidateConfig_InvalidDisplayDimensions(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"zero width", 0, 40},
		{"zero height", 128, 0},
		{"negative width", -128, 40},
		{"negative height", 128, -40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				GameName:        "TEST",
				GameDisplayName: "Test",
				Display: DisplayConfig{
					Width:  tt.width,
					Height: tt.height,
				},
				RefreshRateMs: 100,
				Widgets: []WidgetConfig{
					{
						Type:     "clock",
						ID:       "test",
						Enabled:  BoolPtr(true),
						Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
					},
				},
			}

			err := validateConfig(cfg)
			if err == nil {
				t.Errorf("validateConfig() should return error for %s", tt.name)
			}
		})
	}
}

// TestValidateConfig_InvalidRefreshRate tests validation of refresh rate
func TestValidateConfig_InvalidRefreshRate(t *testing.T) {
	cfg := &Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Display: DisplayConfig{
			Width:  128,
			Height: 40,
		},
		RefreshRateMs: 0,
		Widgets: []WidgetConfig{
			{
				Type:     "clock",
				ID:       "test",
				Enabled:  BoolPtr(true),
				Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
			},
		},
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Error("validateConfig() should return error for zero refresh_rate_ms")
	}
}

// TestValidateConfig_DeinitializeTimer tests validation of deinitialize_timer_length_ms
func TestValidateConfig_DeinitializeTimer(t *testing.T) {
	tests := []struct {
		name      string
		timerMs   int
		shouldErr bool
	}{
		{"valid minimum", 1000, false},
		{"valid middle", 30000, false},
		{"valid maximum", 60000, false},
		{"omitted (zero)", 0, false},
		{"too low", 999, true},
		{"too high", 60001, true},
		{"negative", -1000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				GameName:            "TEST",
				GameDisplayName:     "Test",
				DeinitializeTimerMs: tt.timerMs,
				Display: DisplayConfig{
					Width:  128,
					Height: 40,
				},
				RefreshRateMs: 100,
				Widgets: []WidgetConfig{
					{
						Type:     "clock",
						ID:       "test",
						Enabled:  BoolPtr(true),
						Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
						Properties: WidgetProperties{
							Format: "%H:%M:%S",
						},
					},
				},
			}

			err := validateConfig(cfg)
			if tt.shouldErr && err == nil {
				t.Errorf("validateConfig() should return error for deinitialize_timer_length_ms=%d", tt.timerMs)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("validateConfig() should not return error for deinitialize_timer_length_ms=%d, got: %v", tt.timerMs, err)
			}
		})
	}
}

// TestValidateConfig_EventBatchSize tests validation of event_batch_size
func TestValidateConfig_EventBatchSize(t *testing.T) {
	tests := []struct {
		name      string
		batchSize int
		shouldErr bool
	}{
		{"valid minimum", 1, false},
		{"valid middle", 50, false},
		{"valid maximum", 100, false},
		{"omitted (zero)", 0, false},
		{"too low", 0, false}, // 0 is treated as omitted
		{"negative", -1, true},
		{"too high", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				GameName:        "TEST",
				GameDisplayName: "Test",
				EventBatchSize:  tt.batchSize,
				Display: DisplayConfig{
					Width:  128,
					Height: 40,
				},
				RefreshRateMs: 100,
				Widgets: []WidgetConfig{
					{
						Type:     "clock",
						ID:       "test",
						Enabled:  BoolPtr(true),
						Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
						Properties: WidgetProperties{
							Format: "%H:%M:%S",
						},
					},
				},
			}

			err := validateConfig(cfg)
			if tt.shouldErr && err == nil {
				t.Errorf("validateConfig() should return error for event_batch_size=%d", tt.batchSize)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("validateConfig() should not return error for event_batch_size=%d, got: %v", tt.batchSize, err)
			}
		})
	}
}

// TestValidateConfig_SupportedResolutions tests validation of supported_resolutions
func TestValidateConfig_SupportedResolutions(t *testing.T) {
	tests := []struct {
		name        string
		resolutions []ResolutionConfig
		shouldErr   bool
	}{
		{"valid single resolution", []ResolutionConfig{{Width: 128, Height: 48}}, false},
		{"valid multiple resolutions", []ResolutionConfig{{Width: 128, Height: 36}, {Width: 128, Height: 52}}, false},
		{"empty array", []ResolutionConfig{}, false},
		{"zero width", []ResolutionConfig{{Width: 0, Height: 40}}, true},
		{"zero height", []ResolutionConfig{{Width: 128, Height: 0}}, true},
		{"negative width", []ResolutionConfig{{Width: -128, Height: 40}}, true},
		{"negative height", []ResolutionConfig{{Width: 128, Height: -40}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				GameName:             "TEST",
				GameDisplayName:      "Test",
				SupportedResolutions: tt.resolutions,
				Display: DisplayConfig{
					Width:  128,
					Height: 40,
				},
				RefreshRateMs: 100,
				Widgets: []WidgetConfig{
					{
						Type:     "clock",
						ID:       "test",
						Enabled:  BoolPtr(true),
						Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
						Properties: WidgetProperties{
							Format: "%H:%M:%S",
						},
					},
				},
			}

			err := validateConfig(cfg)
			if tt.shouldErr && err == nil {
				t.Errorf("validateConfig() should return error for resolutions=%v", tt.resolutions)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("validateConfig() should not return error for resolutions=%v, got: %v", tt.resolutions, err)
			}
		})
	}
}

// TestValidateConfig_NoWidgets tests validation when no widgets are configured
func TestValidateConfig_NoWidgets(t *testing.T) {
	cfg := &Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Display: DisplayConfig{
			Width:  128,
			Height: 40,
		},
		RefreshRateMs: 100,
		Widgets:       []WidgetConfig{},
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Error("validateConfig() should return error for no widgets")
	}
}

// TestValidateConfig_NoEnabledWidgets tests that configs with all widgets disabled are valid
// (they will show "NO WIDGETS" error display at runtime)
func TestValidateConfig_NoEnabledWidgets(t *testing.T) {
	cfg := &Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Display: DisplayConfig{
			Width:  128,
			Height: 40,
		},
		RefreshRateMs: 100,
		Widgets: []WidgetConfig{
			{
				Type:     "clock",
				ID:       "test",
				Enabled:  BoolPtr(false),
				Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
			},
		},
	}

	err := validateConfig(cfg)
	if err != nil {
		t.Errorf("validateConfig() should allow config with all widgets disabled (will show error at runtime), got error: %v", err)
	}
}

// TestValidateConfig_MissingWidgetID tests validation of missing widget ID
func TestValidateConfig_MissingWidgetID(t *testing.T) {
	cfg := &Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Display: DisplayConfig{
			Width:  128,
			Height: 40,
		},
		RefreshRateMs: 100,
		Widgets: []WidgetConfig{
			{
				Type:     "clock",
				ID:       "", // Missing ID
				Enabled:  BoolPtr(true),
				Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
			},
		},
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Error("validateConfig() should return error for missing widget ID")
	}
}

// TestValidateConfig_MissingWidgetType tests validation of missing widget type
func TestValidateConfig_MissingWidgetType(t *testing.T) {
	cfg := &Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Display: DisplayConfig{
			Width:  128,
			Height: 40,
		},
		RefreshRateMs: 100,
		Widgets: []WidgetConfig{
			{
				Type:     "", // Missing type
				ID:       "test",
				Enabled:  BoolPtr(true),
				Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
			},
		},
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Error("validateConfig() should return error for missing widget type")
	}
}

// TestValidateConfig_InvalidWidgetType tests validation of invalid widget type
func TestValidateConfig_InvalidWidgetType(t *testing.T) {
	cfg := &Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Display: DisplayConfig{
			Width:  128,
			Height: 40,
		},
		RefreshRateMs: 100,
		Widgets: []WidgetConfig{
			{
				Type:     "invalid_type",
				ID:       "test",
				Enabled:  BoolPtr(true),
				Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
			},
		},
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Error("validateConfig() should return error for invalid widget type")
	}
}

// TestValidateWidgetProperties_ClockMissingFormat tests clock widget without format
func TestValidateWidgetProperties_ClockMissingFormat(t *testing.T) {
	w := &WidgetConfig{
		Type:     "clock",
		ID:       "test",
		Enabled:  BoolPtr(true),
		Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Properties: WidgetProperties{
			Format: "", // Missing format
		},
	}

	err := validateWidgetProperties(0, w)
	if err == nil {
		t.Error("validateWidgetProperties() should return error for clock without format")
	}
}

// TestValidateWidgetProperties_NetworkMissingInterface tests network widget validation
func TestValidateWidgetProperties_NetworkMissingInterface(t *testing.T) {
	emptyInterface := ""
	w := &WidgetConfig{
		Type:     "network",
		ID:       "test",
		Enabled:  BoolPtr(true),
		Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Properties: WidgetProperties{
			Interface: &emptyInterface, // Empty interface
		},
	}

	err := validateWidgetProperties(0, w)
	if err == nil {
		t.Error("validateWidgetProperties() should return error for network with empty interface")
	}
}

// TestValidateWidgetProperties_DiskMissingName tests disk widget validation
func TestValidateWidgetProperties_DiskMissingName(t *testing.T) {
	emptyDisk := ""
	w := &WidgetConfig{
		Type:     "disk",
		ID:       "test",
		Enabled:  BoolPtr(true),
		Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Properties: WidgetProperties{
			DiskName: &emptyDisk, // Empty disk name
		},
	}

	err := validateWidgetProperties(0, w)
	if err == nil {
		t.Error("validateWidgetProperties() should return error for disk with empty disk_name")
	}
}

// TestApplyDefaults_AllWidgetTypes tests that defaults are applied for all widget types
func TestApplyDefaults_AllWidgetTypes(t *testing.T) {
	cfg := &Config{
		Widgets: []WidgetConfig{
			{Type: "clock", ID: "clock1"},
			{Type: "cpu", ID: "cpu1"},
			{Type: "memory", ID: "mem1"},
			{Type: "network", ID: "net1"},
			{Type: "disk", ID: "disk1"},
			{Type: "keyboard", ID: "kbd1"},
		},
	}

	applyDefaults(cfg)

	// Verify each widget got its defaults
	for i, w := range cfg.Widgets {
		if w.Properties.UpdateInterval == 0 {
			t.Errorf("Widget %d (%s) missing default UpdateInterval", i, w.Type)
		}

		if w.Properties.FontSize == 0 {
			t.Errorf("Widget %d (%s) missing default FontSize", i, w.Type)
		}
	}

	// Verify type-specific defaults
	if cfg.Widgets[0].Properties.Format == "" {
		t.Error("Clock widget missing default format")
	}

	if cfg.Widgets[1].Properties.DisplayMode == "" {
		t.Error("CPU widget missing default display mode")
	}
}

// TestLoad_PartialConfig tests that defaults are applied to partial configs
func TestLoad_PartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial_config.json")

	// Minimal config with many fields missing
	configJSON := `{
		"game_name": "TEST",
		"game_display_name": "Test Game",
		"widgets": [
			{
				"type": "clock",
				"id": "clock1",
				"position": {"x": 0, "y": 0, "w": 128, "h": 40}
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify defaults were applied
	if cfg.RefreshRateMs == 0 {
		t.Error("RefreshRateMs default was not applied")
	}

	if cfg.Display.Width == 0 {
		t.Error("Display.Width default was not applied")
	}

	if cfg.Widgets[0].Properties.UpdateInterval == 0 {
		t.Error("Widget UpdateInterval default was not applied")
	}
}

// TestDefaultConstants_AreDifferent tests that DefaultGameName and DefaultGameDisplay are different
// This is critical because GameSense API returns 400 error if they're the same
func TestDefaultConstants_AreDifferent(t *testing.T) {
	if DefaultGameName == DefaultGameDisplay {
		t.Errorf("DefaultGameName and DefaultGameDisplay must be different to avoid GameSense API 400 error. Both are: %s", DefaultGameName)
	}

	if DefaultGameName == "" {
		t.Error("DefaultGameName is empty")
	}

	if DefaultGameDisplay == "" {
		t.Error("DefaultGameDisplay is empty")
	}
}

// TestCreateDefault_GameNamesAreDifferent tests that CreateDefault returns different game names
func TestCreateDefault_GameNamesAreDifferent(t *testing.T) {
	cfg := CreateDefault()

	if cfg.GameName == cfg.GameDisplayName {
		t.Errorf("CreateDefault() GameName and GameDisplayName must be different. Both are: %s", cfg.GameName)
	}

	if cfg.GameName != DefaultGameName {
		t.Errorf("CreateDefault() GameName = %s, want %s", cfg.GameName, DefaultGameName)
	}

	if cfg.GameDisplayName != DefaultGameDisplay {
		t.Errorf("CreateDefault() GameDisplayName = %s, want %s", cfg.GameDisplayName, DefaultGameDisplay)
	}
}

// TestApplyDefaults_GameNamesAreDifferent tests that applyDefaults creates different names
func TestApplyDefaults_GameNamesAreDifferent(t *testing.T) {
	cfg := &Config{
		Display: DisplayConfig{
			Width:  128,
			Height: 40,
		},
		RefreshRateMs: 100,
		Widgets: []WidgetConfig{
			{
				Type:     "clock",
				ID:       "test",
				Enabled:  BoolPtr(true),
				Position: PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Properties: WidgetProperties{
					Format: "%H:%M:%S",
				},
			},
		},
	}

	applyDefaults(cfg)

	if cfg.GameName == cfg.GameDisplayName {
		t.Errorf("applyDefaults() must create different GameName and GameDisplayName. Both are: %s", cfg.GameName)
	}

	if cfg.GameName != DefaultGameName {
		t.Errorf("applyDefaults() GameName = %s, want %s", cfg.GameName, DefaultGameName)
	}

	if cfg.GameDisplayName != DefaultGameDisplay {
		t.Errorf("applyDefaults() GameDisplayName = %s, want %s", cfg.GameDisplayName, DefaultGameDisplay)
	}
}

// TestLoad_WithBothFieldsMissing tests that Load handles missing game_name and game_display_name
func TestLoad_WithBothFieldsMissing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "no_game_names.json")

	configJSON := `{
		"refresh_rate_ms": 100,
		"display": {"width": 128, "height": 40, "background_color": 0},
		"widgets": [
			{
				"type": "clock",
				"id": "test",
				"position": {"x": 0, "y": 0, "w": 128, "h": 40}
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify different defaults were applied
	if cfg.GameName == cfg.GameDisplayName {
		t.Errorf("Load() must apply different defaults. Both are: %s", cfg.GameName)
	}

	if cfg.GameName != DefaultGameName {
		t.Errorf("Load() GameName = %s, want %s", cfg.GameName, DefaultGameName)
	}

	if cfg.GameDisplayName != DefaultGameDisplay {
		t.Errorf("Load() GameDisplayName = %s, want %s", cfg.GameDisplayName, DefaultGameDisplay)
	}
}

// TestLoad_WithEmptyStrings tests that Load handles empty string game names
func TestLoad_WithEmptyStrings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty_game_names.json")

	configJSON := `{
		"game_name": "",
		"game_display_name": "",
		"refresh_rate_ms": 100,
		"display": {"width": 128, "height": 40, "background_color": 0},
		"widgets": [
			{
				"type": "clock",
				"id": "test",
				"position": {"x": 0, "y": 0, "w": 128, "h": 40}
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify different defaults were applied for empty strings
	if cfg.GameName == cfg.GameDisplayName {
		t.Errorf("Load() must apply different defaults for empty strings. Both are: %s", cfg.GameName)
	}

	if cfg.GameName != DefaultGameName {
		t.Errorf("Load() GameName = %s, want %s", cfg.GameName, DefaultGameName)
	}

	if cfg.GameDisplayName != DefaultGameDisplay {
		t.Errorf("Load() GameDisplayName = %s, want %s", cfg.GameDisplayName, DefaultGameDisplay)
	}
}

// TestLoad_WithOnlyGameName tests partial defaults application
func TestLoad_WithOnlyGameName(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "only_game_name.json")

	configJSON := `{
		"game_name": "CUSTOM_GAME",
		"refresh_rate_ms": 100,
		"display": {"width": 128, "height": 40, "background_color": 0},
		"widgets": [
			{
				"type": "clock",
				"id": "test",
				"position": {"x": 0, "y": 0, "w": 128, "h": 40}
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify custom game_name is preserved
	if cfg.GameName != "CUSTOM_GAME" {
		t.Errorf("Load() GameName = %s, want CUSTOM_GAME", cfg.GameName)
	}

	// Verify default game_display_name was applied
	if cfg.GameDisplayName != DefaultGameDisplay {
		t.Errorf("Load() GameDisplayName = %s, want %s", cfg.GameDisplayName, DefaultGameDisplay)
	}

	// Most importantly: they should be different
	if cfg.GameName == cfg.GameDisplayName {
		t.Error("Load() should create different GameName and GameDisplayName")
	}
}
