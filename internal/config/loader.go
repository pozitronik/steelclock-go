package config

import (
	"bytes"
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

	return Parse(data)
}

// Parse parses configuration JSON, applies defaults and validates the result.
// It is what Load does with a file's content, so callers that check or save
// configurations (e.g. the web editor) accept exactly what Load accepts.
func Parse(data []byte) (*Config, error) {
	// json.Unmarshal accepts null for a struct; a config must be an object
	if trimmed := bytes.TrimSpace(data); len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("failed to parse config file (invalid JSON): configuration must be a JSON object")
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

// WriteFileAtomic writes data to path through a temporary file in the same
// directory and renames it into place, so a failed write never leaves a
// truncated or partly written file behind. An existing file keeps its
// permission bits (configs may hold credentials and be private); a new file
// gets 0644, like the files the app created before.
func WriteFileAtomic(path string, data []byte) error {
	perm := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	_, err = tmp.Write(data)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chmod(tmpPath, perm)
	}
	if err == nil {
		err = os.Rename(tmpPath, path)
	}
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
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
