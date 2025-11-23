package tray

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// TestNewManager tests tray manager creation
func TestNewManager(t *testing.T) {
	configPath := "test_config.json"
	reloadCalled := false
	exitCalled := false

	onReload := func() error {
		reloadCalled = true
		return nil
	}

	onExit := func() {
		exitCalled = true
	}

	manager := NewManager(configPath, onReload, onExit)

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.configPath != configPath {
		t.Errorf("configPath = %s, want %s", manager.configPath, configPath)
	}

	if manager.onReload == nil {
		t.Error("onReload should not be nil")
	}

	if manager.onExit == nil {
		t.Error("onExit should not be nil")
	}

	// Test that callbacks work
	if err := manager.onReload(); err != nil {
		t.Errorf("onReload() error = %v", err)
	}

	if !reloadCalled {
		t.Error("onReload callback was not called")
	}

	manager.onExit()

	if !exitCalled {
		t.Error("onExit callback was not called")
	}
}

// TestNewManager_NilCallbacks tests that nil callbacks are handled
func TestNewManager_NilCallbacks(t *testing.T) {
	manager := NewManager("config.json", nil, nil)

	if manager == nil {
		t.Fatal("NewManager() with nil callbacks returned nil")
	}

	// Should not panic with nil callbacks
	if manager.onReload != nil {
		_ = manager.onReload()
	}

	if manager.onExit != nil {
		manager.onExit()
	}
}

// TestGetIcon tests the icon getter
func TestGetIcon(t *testing.T) {
	icon := getIcon()

	// Icon data should be available (embedded)
	if len(icon) == 0 {
		t.Log("No embedded icon found (expected in test environment)")
	} else {
		t.Logf("Icon size: %d bytes", len(icon))
	}

	// Should always return a valid byte slice (even if empty)
	if icon == nil {
		t.Error("getIcon() should never return nil")
	}
}

// TestHandleReloadConfig tests the reload config handler
func TestHandleReloadConfig(t *testing.T) {
	reloadCallCount := 0
	onReload := func() error {
		reloadCallCount++
		return nil
	}

	manager := NewManager("test.json", onReload, nil)

	// Simulate reload
	manager.handleReloadConfig()

	if reloadCallCount != 1 {
		t.Errorf("reload call count = %d, want 1", reloadCallCount)
	}

	// Test multiple reloads
	manager.handleReloadConfig()
	manager.handleReloadConfig()

	if reloadCallCount != 3 {
		t.Errorf("reload call count = %d, want 3", reloadCallCount)
	}
}

// TestHandleReloadConfig_NilCallback tests reload with nil callback
func TestHandleReloadConfig_NilCallback(t *testing.T) {
	manager := NewManager("test.json", nil, nil)

	// Should not panic with nil callback
	manager.handleReloadConfig()
}

// TestHandleReloadConfig_ErrorHandling tests reload error handling
func TestHandleReloadConfig_ErrorHandling(t *testing.T) {
	onReload := func() error {
		return os.ErrPermission
	}

	manager := NewManager("test.json", onReload, nil)

	// Should not panic on error, just log it
	manager.handleReloadConfig()
}

// TestHandleEditConfig_NonexistentFile tests editing a non-existent config
func TestHandleEditConfig_NonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent_config.json")

	manager := NewManager(configPath, nil, nil)

	// This will attempt to create default config, but won't actually open an editor in tests
	// We just verify it doesn't panic
	manager.handleEditConfig()

	// Verify that default config was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("handleEditConfig() should create default config if it doesn't exist")
	}
}

// TestHandleEditConfig_ExistingFile tests editing an existing config
func TestHandleEditConfig_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "existing_config.json")

	// Create existing config
	if err := config.SaveDefault(configPath); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	manager := NewManager(configPath, nil, nil)

	// This won't actually open an editor in tests, but verifies no panic
	manager.handleEditConfig()

	// Config should still exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("handleEditConfig() should not delete existing config")
	}
}

// TestHandleEditConfig_InvalidPath tests editing with invalid path
func TestHandleEditConfig_InvalidPath(t *testing.T) {
	// Use a path with invalid characters (platform-specific)
	invalidPath := "/\x00/invalid/path"

	manager := NewManager(invalidPath, nil, nil)

	// Should not panic, just log error
	manager.handleEditConfig()
}

// TestHandleEditConfig_RelativePath tests handling of relative paths
func TestHandleEditConfig_RelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	relativePath := "relative_config.json"
	manager := NewManager(relativePath, nil, nil)

	// Should handle relative path by converting to absolute
	manager.handleEditConfig()

	// Verify config was created in current directory
	absPath := filepath.Join(tmpDir, relativePath)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Error("handleEditConfig() should create config at relative path")
	}
}

// TestOnQuit tests the onQuit handler
func TestOnQuit(t *testing.T) {
	exitCalled := false
	onExit := func() {
		exitCalled = true
	}

	manager := NewManager("test.json", nil, onExit)

	manager.onQuit()

	if !exitCalled {
		t.Error("onQuit() should call onExit callback")
	}
}

// TestOnQuit_NilCallback tests onQuit with nil callback
func TestOnQuit_NilCallback(t *testing.T) {
	manager := NewManager("test.json", nil, nil)

	// Should not panic with nil callback
	manager.onQuit()
}

// TestManager_PathHandling tests various path edge cases
func TestManager_PathHandling(t *testing.T) {
	testCases := []struct {
		name string
		path string
	}{
		{"absolute path", "/tmp/config.json"},
		{"relative path", "config.json"},
		{"nested path", "configs/subfolder/config.json"},
		{"with spaces", "path with spaces/config.json"},
		{"with dots", "../config.json"},
		{"home directory", "~/config.json"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager(tc.path, nil, nil)

			if manager.configPath != tc.path {
				t.Errorf("configPath = %s, want %s", manager.configPath, tc.path)
			}
		})
	}
}

// TestManager_EmptyConfigPath tests manager with empty config path
func TestManager_EmptyConfigPath(t *testing.T) {
	manager := NewManager("", nil, nil)

	if manager.configPath != "" {
		t.Errorf("configPath = %s, want empty string", manager.configPath)
	}

	// Should not panic when handling empty path
	manager.handleEditConfig()
}

// TestHandleEditConfig_CreateNestedDirectory tests config creation in nested directories
func TestHandleEditConfig_CreateNestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir1", "subdir2", "config.json")

	manager := NewManager(configPath, nil, nil)

	manager.handleEditConfig()

	// Verify nested directories were created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("handleEditConfig() should create nested directories for config")
	}
}
