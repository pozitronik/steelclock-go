package webeditor

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// Mock implementations for testing

// mockConfigProvider implements ConfigProvider for testing
type mockConfigProvider struct {
	mu         sync.Mutex
	configPath string
	configData []byte
	loadErr    error
	saveErr    error
	saveCalled bool
	savedData  []byte
}

func (m *mockConfigProvider) GetConfigPath() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.configPath
}

func (m *mockConfigProvider) Load() ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return m.configData, nil
}

func (m *mockConfigProvider) Save(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveCalled = true
	m.savedData = data
	return m.saveErr
}

// mockProfileProvider implements ProfileProvider for testing
type mockProfileProvider struct {
	mu              sync.Mutex
	profiles        []ProfileInfo
	activeProfile   *ProfileInfo
	setActiveErr    error
	createErr       error
	renameErr       error
	createPath      string
	renamePath      string
	setActiveCalled bool
	createCalled    bool
	renameCalled    bool
}

func (m *mockProfileProvider) GetProfiles() []ProfileInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.profiles
}

func (m *mockProfileProvider) GetActiveProfile() *ProfileInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeProfile
}

func (m *mockProfileProvider) SetActiveProfile(_ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setActiveCalled = true
	return m.setActiveErr
}

func (m *mockProfileProvider) CreateProfile(_ string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCalled = true
	if m.createErr != nil {
		return "", m.createErr
	}
	return m.createPath, nil
}

func (m *mockProfileProvider) RenameProfile(_, _ string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.renameCalled = true
	if m.renameErr != nil {
		return "", m.renameErr
	}
	return m.renamePath, nil
}

// mockPreviewProvider implements PreviewProvider for testing
type mockPreviewProvider struct {
	mu        sync.Mutex
	frameData []byte
	frameNum  uint64
	timestamp time.Time
	config    PreviewDisplayConfig
	wsHandled bool
	wsHandler func(w http.ResponseWriter, r *http.Request)
}

func (m *mockPreviewProvider) GetCurrentFrame() (data []byte, frameNum uint64, timestamp time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.frameData, m.frameNum, m.timestamp
}

func (m *mockPreviewProvider) GetPreviewConfig() PreviewDisplayConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.config
}

func (m *mockPreviewProvider) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.wsHandled = true
	handler := m.wsHandler
	m.mu.Unlock()

	if handler != nil {
		handler(w, r)
	}
}

// Test helper functions

func createTestServer(t *testing.T) (*Server, *mockConfigProvider, *mockProfileProvider) {
	t.Helper()

	configProvider := &mockConfigProvider{
		configPath: "/test/config.json",
		configData: []byte(`{"config_name": "test"}`),
	}

	profileProvider := &mockProfileProvider{
		profiles: []ProfileInfo{
			{Path: "/test/profile1.json", Name: "Profile 1", IsActive: true},
			{Path: "/test/profile2.json", Name: "Profile 2", IsActive: false},
		},
		activeProfile: &ProfileInfo{Path: "/test/profile1.json", Name: "Profile 1", IsActive: true},
	}

	// Create temp schema file
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.json")
	if err := os.WriteFile(schemaPath, []byte(`{"type": "object"}`), 0644); err != nil {
		t.Fatalf("Failed to create schema file: %v", err)
	}

	server := NewServer(
		configProvider,
		profileProvider,
		schemaPath,
		func() error { return nil },
		func(path string) error { return nil },
	)

	return server, configProvider, profileProvider
}

func createTestMux(s *Server) *http.ServeMux {
	mux := http.NewServeMux()
	s.registerHandlers(mux)
	return mux
}

// Server lifecycle tests

func TestNewServer(t *testing.T) {
	configProvider := &mockConfigProvider{}
	profileProvider := &mockProfileProvider{}

	server := NewServer(
		configProvider,
		profileProvider,
		"/test/schema.json",
		func() error { return nil },
		func(path string) error { return nil },
	)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.configProvider != configProvider {
		t.Error("configProvider not set correctly")
	}

	if server.profileProvider != profileProvider {
		t.Error("profileProvider not set correctly")
	}

	if server.schemaPath != "/test/schema.json" {
		t.Error("schemaPath not set correctly")
	}

	if server.running {
		t.Error("Server should not be running initially")
	}
}

func TestServer_SetPreviewProvider(t *testing.T) {
	server, _, _ := createTestServer(t)

	previewProvider := &mockPreviewProvider{
		config: PreviewDisplayConfig{Width: 128, Height: 40, TargetFPS: 30},
	}

	server.SetPreviewProvider(previewProvider)

	if server.previewProvider != previewProvider {
		t.Error("previewProvider not set correctly")
	}
}

func TestServer_SetPreviewOverrideCallback(t *testing.T) {
	server, _, _ := createTestServer(t)

	callbackCalled := false
	callback := func(enable bool) error {
		callbackCalled = true
		return nil
	}

	server.SetPreviewOverrideCallback(callback)

	if server.onPreviewOverride == nil {
		t.Error("onPreviewOverride not set")
	}

	// Verify callback works
	_ = server.onPreviewOverride(true)
	if !callbackCalled {
		t.Error("Callback was not invoked")
	}
}

func TestServer_IsRunning(t *testing.T) {
	server, _, _ := createTestServer(t)

	if server.IsRunning() {
		t.Error("Server should not be running initially")
	}
}

func TestServer_GetURL_NotRunning(t *testing.T) {
	server, _, _ := createTestServer(t)

	url := server.GetURL()
	if url != "" {
		t.Errorf("GetURL should return empty string when not running, got %q", url)
	}
}

func TestServer_Stop_NotRunning(t *testing.T) {
	server, _, _ := createTestServer(t)

	// Should not error when stopping a non-running server
	err := server.Stop()
	if err != nil {
		t.Errorf("Stop on non-running server should not error: %v", err)
	}
}

// Handler tests - Schema

func TestHandleGetSchema_Success(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/schema", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "object") {
		t.Errorf("Expected schema content, got %q", string(body))
	}
}

func TestHandleGetSchema_MethodNotAllowed(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodPost, "/api/schema", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleGetSchema_NotFound(t *testing.T) {
	configProvider := &mockConfigProvider{}
	profileProvider := &mockProfileProvider{}

	server := NewServer(
		configProvider,
		profileProvider,
		"/nonexistent/schema.json",
		func() error { return nil },
		func(path string) error { return nil },
	)

	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/schema", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

// Handler tests - Config

func TestHandleConfig_Get_Success(t *testing.T) {
	server, configProvider, _ := createTestServer(t)
	configProvider.configData = []byte(`{"config_name": "test_config"}`)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "test_config") {
		t.Errorf("Expected config content, got %q", string(body))
	}
}

func TestHandleConfig_Get_Error(t *testing.T) {
	server, configProvider, _ := createTestServer(t)
	configProvider.loadErr = errors.New("load failed")
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestHandleConfig_Post_Success(t *testing.T) {
	server, configProvider, _ := createTestServer(t)
	mux := createTestMux(server)

	body := `{"config_name": "new_config"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if !configProvider.saveCalled {
		t.Error("Save was not called on configProvider")
	}
}

func TestHandleConfig_Post_InvalidJSON(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleConfig_Post_ForbiddenOrigin(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	body := `{"config_name": "test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}
}

func TestHandleConfig_Post_LocalhostOrigin(t *testing.T) {
	server, configProvider, _ := createTestServer(t)
	mux := createTestMux(server)

	body := `{"config_name": "test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if !configProvider.saveCalled {
		t.Error("Save should be called for localhost origin")
	}
}

func TestHandleConfig_Post_NoOrigin(t *testing.T) {
	server, configProvider, _ := createTestServer(t)
	mux := createTestMux(server)

	body := `{"config_name": "test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Origin header
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 (no origin = allowed), got %d", resp.StatusCode)
	}

	if !configProvider.saveCalled {
		t.Error("Save should be called when no origin header")
	}
}

func TestHandleConfig_Post_WrappedFormat(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	// Create temp file for saving
	tmpDir := t.TempDir()
	savePath := filepath.Join(tmpDir, "save.json")

	// Use filepath.ToSlash for cross-platform JSON compatibility
	savePathJSON := filepath.ToSlash(savePath)

	body := `{"path": "` + savePathJSON + `", "config": {"config_name": "wrapped"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	// Verify file was saved
	savedData, err := os.ReadFile(savePath)
	if err != nil {
		t.Errorf("Failed to read saved file: %v", err)
	}

	if !strings.Contains(string(savedData), "wrapped") {
		t.Errorf("Expected wrapped config content, got %q", string(savedData))
	}
}

func TestHandleConfig_MethodNotAllowed(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodPut, "/api/config", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

// Handler tests - Load Config By Path

func TestHandleLoadConfigByPath_Success(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(configPath, []byte(`{"loaded": true}`), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Use filepath.ToSlash for cross-platform JSON compatibility
	configPathJSON := filepath.ToSlash(configPath)

	body := `{"path": "` + configPathJSON + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config/load", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(respBody), "loaded") {
		t.Errorf("Expected loaded config, got %q", string(respBody))
	}
}

func TestHandleLoadConfigByPath_MissingPath(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/config/load", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleLoadConfigByPath_FileNotFound(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	body := `{"path": "/nonexistent/file.json"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config/load", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestHandleLoadConfigByPath_MethodNotAllowed(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/config/load", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

// Handler tests - Validate

func TestHandleValidate_ValidConfig(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	// Set up widget type checker to accept "clock" type
	oldChecker := config.WidgetTypeChecker
	config.WidgetTypeChecker = func(typeName string) bool {
		return typeName == "clock"
	}
	defer func() { config.WidgetTypeChecker = oldChecker }()

	// Minimal valid config with all required fields
	body := `{
		"backend": "",
		"refresh_rate_ms": 100,
		"display": {"width": 128, "height": 40},
		"widgets": [{"type": "clock", "position": {"w": 64, "h": 40}}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if valid, ok := result["valid"].(bool); !ok || !valid {
		errList := result["errors"]
		t.Errorf("Expected valid=true, got %v (errors: %v)", result["valid"], errList)
	}
}

func TestHandleValidate_InvalidJSON(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPost, "/api/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if valid, ok := result["valid"].(bool); !ok || valid {
		t.Errorf("Expected valid=false for invalid JSON, got %v", result["valid"])
	}
}

func TestHandleValidate_MethodNotAllowed(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/validate", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

// Handler tests - Profiles

func TestHandleProfiles_List(t *testing.T) {
	server, _, profileProvider := createTestServer(t)
	profileProvider.profiles = []ProfileInfo{
		{Path: "/p1.json", Name: "Profile 1", IsActive: true},
		{Path: "/p2.json", Name: "Profile 2", IsActive: false},
	}
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/profiles", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var profiles []ProfileInfo
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(profiles))
	}
}

func TestHandleProfiles_List_NilProvider(t *testing.T) {
	configProvider := &mockConfigProvider{}

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.json")
	_ = os.WriteFile(schemaPath, []byte(`{}`), 0644)

	server := NewServer(
		configProvider,
		nil, // nil profile provider
		schemaPath,
		func() error { return nil },
		func(path string) error { return nil },
	)

	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/profiles", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var profiles []ProfileInfo
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(profiles) != 0 {
		t.Errorf("Expected empty profiles for nil provider, got %d", len(profiles))
	}
}

func TestHandleProfiles_Create_Success(t *testing.T) {
	server, _, profileProvider := createTestServer(t)
	profileProvider.createPath = "/new/profile.json"
	mux := createTestMux(server)

	body := `{"name": "New Profile"}`
	req := httptest.NewRequest(http.MethodPost, "/api/profiles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if !profileProvider.createCalled {
		t.Error("CreateProfile was not called")
	}
}

func TestHandleProfiles_Create_EmptyName(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	body := `{"name": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/profiles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleProfiles_Create_ForbiddenOrigin(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	body := `{"name": "New Profile"}`
	req := httptest.NewRequest(http.MethodPost, "/api/profiles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}
}

func TestHandleProfiles_MethodNotAllowed(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodPut, "/api/profiles", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

// Handler tests - Active Profile

func TestHandleActiveProfile_Success(t *testing.T) {
	configProvider := &mockConfigProvider{
		configPath: "/test/config.json",
		configData: []byte(`{"config_name": "test"}`),
	}

	profileProvider := &mockProfileProvider{
		profiles: []ProfileInfo{
			{Path: "/test/profile1.json", Name: "Profile 1", IsActive: true},
		},
	}

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.json")
	_ = os.WriteFile(schemaPath, []byte(`{}`), 0644)

	// Track if onProfileSwitch callback was called
	profileSwitchCalled := false
	profileSwitchPath := ""

	server := NewServer(
		configProvider,
		profileProvider,
		schemaPath,
		func() error { return nil },
		func(path string) error {
			profileSwitchCalled = true
			profileSwitchPath = path
			return nil
		},
	)

	mux := createTestMux(server)

	body := `{"path": "/test/profile.json"}`
	req := httptest.NewRequest(http.MethodPost, "/api/profiles/active", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	if !profileSwitchCalled {
		t.Error("onProfileSwitch callback was not called")
	}

	if profileSwitchPath != "/test/profile.json" {
		t.Errorf("Expected path '/test/profile.json', got %q", profileSwitchPath)
	}
}

func TestHandleActiveProfile_NilProvider(t *testing.T) {
	configProvider := &mockConfigProvider{}

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.json")
	_ = os.WriteFile(schemaPath, []byte(`{}`), 0644)

	server := NewServer(
		configProvider,
		nil, // nil profile provider
		schemaPath,
		func() error { return nil },
		func(path string) error { return nil },
	)

	mux := createTestMux(server)

	body := `{"path": "/test/profile.json"}`
	req := httptest.NewRequest(http.MethodPost, "/api/profiles/active", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("Expected status 501, got %d", resp.StatusCode)
	}
}

func TestHandleActiveProfile_MethodNotAllowed(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/active", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

// Handler tests - Rename Profile

func TestHandleRenameProfile_Success(t *testing.T) {
	server, _, profileProvider := createTestServer(t)
	profileProvider.renamePath = "/new/path.json"
	mux := createTestMux(server)

	body := `{"path": "/old/path.json", "new_name": "New Name"}`
	req := httptest.NewRequest(http.MethodPost, "/api/profiles/rename", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if !profileProvider.renameCalled {
		t.Error("RenameProfile was not called")
	}
}

func TestHandleRenameProfile_MissingPath(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	body := `{"new_name": "New Name"}`
	req := httptest.NewRequest(http.MethodPost, "/api/profiles/rename", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleRenameProfile_MissingNewName(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	body := `{"path": "/old/path.json"}`
	req := httptest.NewRequest(http.MethodPost, "/api/profiles/rename", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleRenameProfile_MethodNotAllowed(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/rename", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

// Handler tests - Preview

func TestHandlePreviewInfo_Available(t *testing.T) {
	server, _, _ := createTestServer(t)
	server.SetPreviewProvider(&mockPreviewProvider{
		config: PreviewDisplayConfig{Width: 128, Height: 40, TargetFPS: 30},
	})
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/preview", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !result["available"].(bool) {
		t.Error("Expected available=true")
	}

	if int(result["width"].(float64)) != 128 {
		t.Errorf("Expected width=128, got %v", result["width"])
	}
}

func TestHandlePreviewInfo_NotAvailable(t *testing.T) {
	server, _, _ := createTestServer(t)
	// No preview provider set
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/preview", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["available"].(bool) {
		t.Error("Expected available=false when no provider")
	}
}

func TestHandlePreviewInfo_MethodNotAllowed(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodPost, "/api/preview", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandlePreviewFrame_Success(t *testing.T) {
	server, _, _ := createTestServer(t)
	now := time.Now()
	server.SetPreviewProvider(&mockPreviewProvider{
		frameData: []byte{0xFF, 0x00, 0xFF},
		frameNum:  42,
		timestamp: now,
		config:    PreviewDisplayConfig{Width: 128, Height: 40, TargetFPS: 30},
	})
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/preview/frame", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if int(result["frame_number"].(float64)) != 42 {
		t.Errorf("Expected frame_number=42, got %v", result["frame_number"])
	}
}

func TestHandlePreviewFrame_NoFrame(t *testing.T) {
	server, _, _ := createTestServer(t)
	server.SetPreviewProvider(&mockPreviewProvider{
		frameData: nil, // No frame yet
		config:    PreviewDisplayConfig{Width: 128, Height: 40, TargetFPS: 30},
	})
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/preview/frame", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["frame"] != nil {
		t.Errorf("Expected frame=nil, got %v", result["frame"])
	}
}

func TestHandlePreviewFrame_NotAvailable(t *testing.T) {
	server, _, _ := createTestServer(t)
	// No preview provider
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/preview/frame", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("Expected status 501, got %d", resp.StatusCode)
	}
}

func TestHandlePreviewWebSocket_Delegates(t *testing.T) {
	server, _, _ := createTestServer(t)
	provider := &mockPreviewProvider{
		config: PreviewDisplayConfig{Width: 128, Height: 40, TargetFPS: 30},
		wsHandler: func(w http.ResponseWriter, r *http.Request) {
			// Just write a response to indicate handler was called
			w.WriteHeader(http.StatusOK)
		},
	}
	server.SetPreviewProvider(provider)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/preview/ws", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if !provider.wsHandled {
		t.Error("WebSocket handler was not called")
	}
}

func TestHandlePreviewWebSocket_NotAvailable(t *testing.T) {
	server, _, _ := createTestServer(t)
	// No preview provider
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/preview/ws", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("Expected status 501, got %d", resp.StatusCode)
	}
}

// Handler tests - Preview Override

func TestHandlePreviewOverride_Enable(t *testing.T) {
	server, _, _ := createTestServer(t)

	overrideEnabled := false
	server.SetPreviewOverrideCallback(func(enable bool) error {
		overrideEnabled = enable
		return nil
	})

	mux := createTestMux(server)

	body := `{"enable": true}`
	req := httptest.NewRequest(http.MethodPost, "/api/preview/override", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if !overrideEnabled {
		t.Error("Override callback was not called with enable=true")
	}
}

func TestHandlePreviewOverride_Disable(t *testing.T) {
	server, _, _ := createTestServer(t)

	overrideEnabled := true
	server.SetPreviewOverrideCallback(func(enable bool) error {
		overrideEnabled = enable
		return nil
	})

	mux := createTestMux(server)

	body := `{"enable": false}`
	req := httptest.NewRequest(http.MethodPost, "/api/preview/override", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if overrideEnabled {
		t.Error("Override callback was not called with enable=false")
	}
}

func TestHandlePreviewOverride_NoCallback(t *testing.T) {
	server, _, _ := createTestServer(t)
	// No override callback set
	mux := createTestMux(server)

	body := `{"enable": true}`
	req := httptest.NewRequest(http.MethodPost, "/api/preview/override", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("Expected status 501, got %d", resp.StatusCode)
	}
}

func TestHandlePreviewOverride_ForbiddenOrigin(t *testing.T) {
	server, _, _ := createTestServer(t)
	server.SetPreviewOverrideCallback(func(enable bool) error { return nil })
	mux := createTestMux(server)

	body := `{"enable": true}`
	req := httptest.NewRequest(http.MethodPost, "/api/preview/override", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}
}

func TestHandlePreviewOverride_MethodNotAllowed(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/api/preview/override", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandlePreviewOverride_InvalidJSON(t *testing.T) {
	server, _, _ := createTestServer(t)
	server.SetPreviewOverrideCallback(func(enable bool) error { return nil })
	mux := createTestMux(server)

	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPost, "/api/preview/override", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8384")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

// Handler tests - Preview redirect

func TestPreviewRedirect(t *testing.T) {
	server, _, _ := createTestServer(t)
	mux := createTestMux(server)

	req := httptest.NewRequest(http.MethodGet, "/preview", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected status 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/preview.html" {
		t.Errorf("Expected redirect to /preview.html, got %q", location)
	}
}

// Helper function tests

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	respondJSON(w, data)

	resp := w.Result()
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"key":"value"`) {
		t.Errorf("Expected JSON output, got %q", string(body))
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()

	respondError(w, "test error", http.StatusBadRequest)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", ct)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["error"] != "test error" {
		t.Errorf("Expected error='test error', got %v", result["error"])
	}
}

// Concurrency tests

func TestServer_ConcurrentAccess(t *testing.T) {
	server, _, _ := createTestServer(t)

	var wg sync.WaitGroup
	const goroutines = 10

	// Test concurrent IsRunning calls
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = server.IsRunning()
		}()
	}
	wg.Wait()

	// Test concurrent GetURL calls
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = server.GetURL()
		}()
	}
	wg.Wait()
}

func TestServer_ConcurrentProviderAccess(t *testing.T) {
	server, _, _ := createTestServer(t)

	var wg sync.WaitGroup
	const goroutines = 10

	// Test concurrent SetPreviewProvider calls
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			server.SetPreviewProvider(&mockPreviewProvider{})
		}()
	}
	wg.Wait()
}
