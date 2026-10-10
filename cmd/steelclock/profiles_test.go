//go:build !light

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// shippedConfigs returns the configuration files shipped with the application:
// the main config and every profile in a resolution group.
func shippedConfigs(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "profiles", "*", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	var result []string
	for _, f := range files {
		if filepath.Base(filepath.Dir(f)) == ".schema" {
			continue
		}
		result = append(result, f)
	}
	return append(result, filepath.Join("..", "..", config.MainConfigFile))
}

// repoRelative turns a path from shippedConfigs into a repository-relative
// name for subtest names, e.g. "profiles/128x40/clock.json".
func repoRelative(path string) string {
	return filepath.ToSlash(strings.TrimPrefix(path, filepath.Join("..", "..")+string(filepath.Separator)))
}

// TestShippedProfilesLoad checks that every shipped configuration loads with
// the widgets and backends this build registers (imports.go), exactly as the
// application loads it. Changes to validation, defaults or widget types that
// would break a shipped profile fail here instead of when a user selects it.
func TestShippedProfilesLoad(t *testing.T) {
	files := shippedConfigs(t)
	// Guard against a moved profiles directory making the test vacuous
	if len(files) < 10 {
		t.Fatalf("found only %d shipped configs: %v", len(files), files)
	}

	for _, path := range files {
		t.Run(repoRelative(path), func(t *testing.T) {
			// Load returns a default config for a missing file; make sure the
			// file is really there
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("config missing: %v", err)
			}
			cfg, err := config.Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if len(cfg.Widgets) == 0 && len(cfg.Devices) == 0 {
				t.Error("config has no widgets")
			}
		})
	}
}

// TestShippedProfilesSchemaReference checks that the relative $schema of every
// shipped configuration points to an existing file, so editors can find it.
func TestShippedProfilesSchemaReference(t *testing.T) {
	for _, path := range shippedConfigs(t) {
		t.Run(repoRelative(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Schema string `json:"$schema"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if err := checkSchemaReference(path, doc.Schema); err != nil {
				t.Error(err)
			}
		})
	}
}

// checkSchemaReference checks a config's $schema value: it must be set, and a
// local reference, resolved against the config's directory, must name a
// regular file (a directory exists but cannot be loaded as a schema).
func checkSchemaReference(configPath, schema string) error {
	if schema == "" {
		return errors.New("no $schema reference")
	}
	if strings.Contains(schema, "://") {
		return nil // remote schema; nothing to resolve locally
	}
	target := filepath.Join(filepath.Dir(configPath), filepath.FromSlash(schema))
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("$schema %q does not resolve: %w", schema, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("$schema %q is not a file", schema)
	}
	return nil
}

func TestCheckSchemaReference(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, ".schema")
	if err := os.Mkdir(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schemaDir, "config.schema.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "group", "profile.json")

	tests := []struct {
		name    string
		schema  string
		wantErr string
	}{
		{"existing file", "../.schema/config.schema.json", ""},
		{"remote schema", "https://example.com/config.schema.json", ""},
		{"missing", "", "no $schema reference"},
		{"missing file", "../.schema/other.schema.json", "does not resolve"},
		{"directory", "../.schema", "is not a file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSchemaReference(configPath, tt.schema)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("checkSchemaReference() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("checkSchemaReference() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
