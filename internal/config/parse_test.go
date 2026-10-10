package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const parseTestWidgets = `"widgets": [{"type": "clock", "position": {"x": 0, "y": 0, "w": 128, "h": 40}}]`

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr string
	}{
		{"minimal config with defaults", `{` + parseTestWidgets + `}`, ""},
		{"full display block", `{"display": {"width": 128, "height": 64}, "refresh_rate_ms": 50, ` + parseTestWidgets + `}`, ""},
		{"leading whitespace", " \n\t{" + parseTestWidgets + "}", ""},
		{"null", `null`, "must be a JSON object"},
		{"array", `[1, 2]`, "must be a JSON object"},
		{"string", `"config"`, "must be a JSON object"},
		{"empty", ``, "must be a JSON object"},
		{"invalid JSON", `{"widgets": }`, "invalid JSON"},
		{"no widgets", `{"widgets": []}`, "at least one widget"},
		{"unknown widget type", `{"widgets": [{"type": "nope"}]}`, "invalid type"},
		{"invalid value", `{"refresh_rate_ms": -1, ` + parseTestWidgets + `}`, "refresh_rate_ms must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse([]byte(tt.data))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Parse() error = %v", err)
				}
				if cfg.Display.Width == 0 || cfg.RefreshRateMs == 0 {
					t.Errorf("defaults not applied: %+v", cfg.Display)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Parse() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

// TestParse_AgreesWithLoad checks that Parse accepts and rejects exactly what
// Load does for the same content.
func TestParse_AgreesWithLoad(t *testing.T) {
	inputs := []string{
		`{` + parseTestWidgets + `}`,
		`{"display": {"width": 0}, ` + parseTestWidgets + `}`,
		`{"widgets": []}`,
		`null`,
		`{"widgets": [{"type": "clock", "update_interval": -1, "position": {"w": 1, "h": 1}}]}`,
	}

	for _, input := range inputs {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(input), 0644); err != nil {
			t.Fatal(err)
		}
		_, loadErr := Load(path)
		_, parseErr := Parse([]byte(input))
		if (loadErr == nil) != (parseErr == nil) {
			t.Errorf("%s: Load() error = %v, Parse() error = %v", input, loadErr, parseErr)
		}
	}
}

func TestWriteFileAtomic(t *testing.T) {
	t.Run("creates file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.json")
		if err := WriteFileAtomic(path, []byte("new")); err != nil {
			t.Fatalf("WriteFileAtomic() error = %v", err)
		}
		assertDirContent(t, dir, map[string]string{"config.json": "new"})
	})

	t.Run("replaces file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.json")
		if err := os.WriteFile(path, []byte("a much longer previous content"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := WriteFileAtomic(path, []byte("new")); err != nil {
			t.Fatalf("WriteFileAtomic() error = %v", err)
		}
		assertDirContent(t, dir, map[string]string{"config.json": "new"})
	})

	t.Run("missing directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing", "config.json")
		if err := WriteFileAtomic(path, []byte("new")); err == nil {
			t.Error("WriteFileAtomic() error = nil, want error")
		}
	})

	t.Run("target is a directory", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "config.json")
		if err := os.Mkdir(target, 0755); err != nil {
			t.Fatal(err)
		}
		if err := WriteFileAtomic(target, []byte("new")); err == nil {
			t.Error("WriteFileAtomic() error = nil, want error")
		}
		// The temporary file must not be left behind
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Errorf("directory has %d entries, want only the target", len(entries))
		}
	})
}

// TestWriteFileAtomic_Permissions checks that replacing a file keeps its
// permission bits, so a private config stays private.
func TestWriteFileAtomic_Permissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not used on Windows")
	}

	tests := []struct {
		name     string
		existing os.FileMode // 0 = no existing file
		want     os.FileMode
	}{
		{"private file stays private", 0600, 0600},
		{"group-readable file keeps its mode", 0640, 0640},
		{"new file", 0, 0644},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if tt.existing != 0 {
				if err := os.WriteFile(path, []byte("old"), tt.existing); err != nil {
					t.Fatal(err)
				}
				// WriteFile applies the umask; set the exact mode
				if err := os.Chmod(path, tt.existing); err != nil {
					t.Fatal(err)
				}
			}

			if err := WriteFileAtomic(path, []byte("new")); err != nil {
				t.Fatalf("WriteFileAtomic() error = %v", err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != tt.want {
				t.Errorf("mode = %o, want %o", got, tt.want)
			}
		})
	}
}

// assertDirContent checks that dir holds exactly the given files.
func assertDirContent(t *testing.T, dir string, want map[string]string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(want) {
		t.Errorf("directory has %d entries, want %d", len(entries), len(want))
	}
	for name, content := range want {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(data) != content {
			t.Errorf("%s = %q (%v), want %q", name, data, err, content)
		}
	}
}
