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

	if !widget.Enabled {
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
