package webeditor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestHandleConfig_Post_RejectsUnloadableConfig checks that a save never
// replaces a config with one the app cannot load.
func TestHandleConfig_Post_RejectsUnloadableConfig(t *testing.T) {
	invalid := []struct {
		name   string
		config string
	}{
		{"null", `null`},
		{"array", `[]`},
		{"no widgets", `{"widgets": []}`},
		{"unknown widget type", `{"widgets": [{"type": "nope", "position": {"w": 1, "h": 1}}]}`},
		{"invalid value", `{"refresh_rate_ms": -1, "widgets": [{"type": "clock", "position": {"w": 1, "h": 1}}]}`},
	}

	for _, tt := range invalid {
		t.Run("active config: "+tt.name, func(t *testing.T) {
			server, configProvider, _ := createTestServer(t)
			req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(tt.config))
			w := httptest.NewRecorder()
			createTestMux(server).ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
			}
			if configProvider.saveCalled {
				t.Error("invalid config was saved")
			}
		})

		t.Run("profile: "+tt.name, func(t *testing.T) {
			server, _, profileProvider := createTestServer(t)
			path := filepath.Join(t.TempDir(), "profile.json")
			original := fmt.Sprintf(testConfigJSON, "original")
			if err := os.WriteFile(path, []byte(original), 0644); err != nil {
				t.Fatal(err)
			}
			profileProvider.profiles = append(profileProvider.profiles, ProfileInfo{Path: path, Name: "Profile"})

			body := `{"path": ` + jsonString(t, path) + `, "config": ` + tt.config + `}`
			req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(body))
			w := httptest.NewRecorder()
			createTestMux(server).ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != original {
				t.Errorf("profile content = %q (%v), want it unchanged", data, err)
			}
		})
	}
}

// TestHandleConfig_Post_KeepsPrivateProfilePrivate saves a profile whose file
// is readable only by its owner; the save must not widen its permissions.
func TestHandleConfig_Post_KeepsPrivateProfilePrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not used on Windows")
	}

	server, _, profileProvider := createTestServer(t)
	path := filepath.Join(t.TempDir(), "private.json")
	if err := os.WriteFile(path, []byte(fmt.Sprintf(testConfigJSON, "old")), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	profileProvider.profiles = append(profileProvider.profiles, ProfileInfo{Path: path, Name: "Private"})

	body := `{"path": ` + jsonString(t, path) + `, "config": ` + fmt.Sprintf(testConfigJSON, "new") + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(body))
	w := httptest.NewRecorder()
	createTestMux(server).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("mode after save = %o, want 600", got)
	}
}

// TestHandleValidate_AgreesWithLoader checks that validation applies the same
// defaults as loading: a config that omits defaulted fields is valid.
func TestHandleValidate_AgreesWithLoader(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantValid bool
	}{
		{"omits display and refresh rate", fmt.Sprintf(testConfigJSON, "defaults"), true},
		{"null", `null`, false},
		{"no widgets", `{"widgets": []}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _, _ := createTestServer(t)
			req := httptest.NewRequest(http.MethodPost, "/api/validate", strings.NewReader(tt.config))
			w := httptest.NewRecorder()
			createTestMux(server).ServeHTTP(w, req)

			var result struct {
				Valid  bool     `json:"valid"`
				Errors []string `json:"errors"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatalf("decode response: %v (%s)", err, w.Body.String())
			}
			if result.Valid != tt.wantValid {
				t.Errorf("valid = %v (errors %v), want %v", result.Valid, result.Errors, tt.wantValid)
			}
		})
	}
}

// jsonString encodes s as a JSON string literal.
func jsonString(t *testing.T, s string) string {
	t.Helper()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
