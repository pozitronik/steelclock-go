package gamesense

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverServer(t *testing.T) {
	// Save original DI variables and restore after test
	origFunc := findCorePropsPathFunc
	origFallback := defaultFallbackPath
	defer func() {
		findCorePropsPathFunc = origFunc
		defaultFallbackPath = origFallback
	}()

	t.Run("valid coreProps", func(t *testing.T) {
		tmpDir := t.TempDir()
		propsFile := filepath.Join(tmpDir, "coreProps.json")
		os.WriteFile(propsFile, []byte(`{"address":"127.0.0.1:54321"}`), 0644)

		findCorePropsPathFunc = func() (string, error) {
			return propsFile, nil
		}

		addr, err := DiscoverServer()
		if err != nil {
			t.Fatalf("DiscoverServer() error = %v", err)
		}
		if addr != "127.0.0.1:54321" {
			t.Errorf("address = %q, want %q", addr, "127.0.0.1:54321")
		}
	})

	t.Run("empty address", func(t *testing.T) {
		tmpDir := t.TempDir()
		propsFile := filepath.Join(tmpDir, "coreProps.json")
		os.WriteFile(propsFile, []byte(`{"address":""}`), 0644)

		findCorePropsPathFunc = func() (string, error) {
			return propsFile, nil
		}

		_, err := DiscoverServer()
		if err == nil {
			t.Fatal("expected error for empty address")
		}
		if !strings.Contains(err.Error(), "no 'address' field") {
			t.Errorf("error = %q, should mention missing address", err.Error())
		}
	})

	t.Run("missing address field", func(t *testing.T) {
		tmpDir := t.TempDir()
		propsFile := filepath.Join(tmpDir, "coreProps.json")
		os.WriteFile(propsFile, []byte(`{"version":"3.0"}`), 0644)

		findCorePropsPathFunc = func() (string, error) {
			return propsFile, nil
		}

		_, err := DiscoverServer()
		if err == nil {
			t.Fatal("expected error for missing address")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		propsFile := filepath.Join(tmpDir, "coreProps.json")
		os.WriteFile(propsFile, []byte(`not json`), 0644)

		findCorePropsPathFunc = func() (string, error) {
			return propsFile, nil
		}

		_, err := DiscoverServer()
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		findCorePropsPathFunc = func() (string, error) {
			return "/nonexistent/path", nil
		}

		_, err := DiscoverServer()
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("path finder error", func(t *testing.T) {
		findCorePropsPathFunc = func() (string, error) {
			return "", os.ErrNotExist
		}

		_, err := DiscoverServer()
		if err == nil {
			t.Fatal("expected error when path finder fails")
		}
	})

	t.Run("invalid address format (no port)", func(t *testing.T) {
		tmpDir := t.TempDir()
		propsFile := filepath.Join(tmpDir, "coreProps.json")
		os.WriteFile(propsFile, []byte(`{"address":"localhost"}`), 0644)

		findCorePropsPathFunc = func() (string, error) {
			return propsFile, nil
		}

		_, err := DiscoverServer()
		if err == nil {
			t.Fatal("expected error for address without port")
		}
		if !strings.Contains(err.Error(), "invalid address format") {
			t.Errorf("error = %q, should mention invalid format", err.Error())
		}
	})

	t.Run("invalid port number", func(t *testing.T) {
		tmpDir := t.TempDir()
		propsFile := filepath.Join(tmpDir, "coreProps.json")
		os.WriteFile(propsFile, []byte(`{"address":"localhost:abc"}`), 0644)

		findCorePropsPathFunc = func() (string, error) {
			return propsFile, nil
		}

		_, err := DiscoverServer()
		if err == nil {
			t.Fatal("expected error for non-numeric port")
		}
		if !strings.Contains(err.Error(), "invalid port") {
			t.Errorf("error = %q, should mention invalid port", err.Error())
		}
	})
}

func TestFindCorePropsPath(t *testing.T) {
	// Save and restore
	origFallback := defaultFallbackPath
	defer func() {
		defaultFallbackPath = origFallback
	}()

	t.Run("finds via PROGRAMDATA env", func(t *testing.T) {
		tmpDir := t.TempDir()
		ssDir := filepath.Join(tmpDir, "SteelSeries", "SteelSeries Engine 3")
		os.MkdirAll(ssDir, 0755)
		propsFile := filepath.Join(ssDir, "coreProps.json")
		os.WriteFile(propsFile, []byte(`{}`), 0644)

		t.Setenv("PROGRAMDATA", tmpDir)
		defaultFallbackPath = "/nonexistent/path" // ensure fallback is not used

		path, err := findCorePropsPath()
		if err != nil {
			t.Fatalf("findCorePropsPath() error = %v", err)
		}
		if path != propsFile {
			t.Errorf("path = %q, want %q", path, propsFile)
		}
	})

	t.Run("falls back to default path", func(t *testing.T) {
		tmpDir := t.TempDir()
		fallbackFile := filepath.Join(tmpDir, "coreProps.json")
		os.WriteFile(fallbackFile, []byte(`{}`), 0644)

		t.Setenv("PROGRAMDATA", "/nonexistent/programdata")
		defaultFallbackPath = fallbackFile

		path, err := findCorePropsPath()
		if err != nil {
			t.Fatalf("findCorePropsPath() error = %v", err)
		}
		if path != fallbackFile {
			t.Errorf("path = %q, want %q", path, fallbackFile)
		}
	})

	t.Run("no file found", func(t *testing.T) {
		t.Setenv("PROGRAMDATA", "/nonexistent/programdata")
		defaultFallbackPath = "/nonexistent/fallback"

		_, err := findCorePropsPath()
		if err == nil {
			t.Fatal("expected error when no coreProps.json found")
		}
		if !strings.Contains(err.Error(), "cannot find coreProps.json") {
			t.Errorf("error = %q, should mention cannot find file", err.Error())
		}
	})
}
