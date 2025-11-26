package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Load reads and parses a configuration file
// If the file doesn't exist, returns a default configuration
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := CreateDefault()
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file (invalid JSON): %w", err)
	}

	// Apply defaults for missing fields
	applyDefaults(&cfg)

	// Validate configuration
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// SaveDefault creates and saves a default configuration file
func SaveDefault(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	cfg := CreateDefault()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
