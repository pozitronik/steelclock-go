package webeditor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWithAccessPolicy(t *testing.T) {
	const (
		local  = "127.0.0.1:50000"
		remote = "192.168.1.5:50000"
		host   = "127.0.0.1:8384"
	)

	tests := []struct {
		name       string
		method     string
		path       string
		remoteAddr string
		host       string
		origin     string
		allowed    bool
	}{
		{"local read", http.MethodGet, "/api/config", local, host, "", true},
		{"local read via localhost", http.MethodGet, "/api/config", local, "localhost:8384", "", true},
		{"local read via IPv6 loopback", http.MethodGet, "/api/config", "[::1]:50000", "[::1]:8384", "", true},
		{"local read of static page", http.MethodGet, "/", local, host, "", true},
		{"remote read", http.MethodGet, "/api/config", remote, "192.168.1.2:8384", "", false},
		{"remote static page", http.MethodGet, "/", remote, "192.168.1.2:8384", "", false},
		{"remote preview socket", http.MethodGet, "/api/preview/ws", remote, "192.168.1.2:8384", "", false},
		{"remote profile load", http.MethodPost, "/api/config/load", remote, "192.168.1.2:8384", "", false},
		{"remote claude status update (WSL hook)", http.MethodPost, "/api/claude-status", "172.20.0.5:40000", "172.20.0.1:8384", "", true},
		{"remote claude status read", http.MethodGet, "/api/claude-status", remote, "192.168.1.2:8384", "", true},
		{"remote claude status from a web page", http.MethodPost, "/api/claude-status", remote, "192.168.1.2:8384", "http://evil.example", false},
		{"rebound host name", http.MethodGet, "/api/config", local, "evil.example:8384", "", false},
		{"rebound host name posting", http.MethodPost, "/api/config", local, "evil.example:8384", "http://evil.example:8384", false},
		{"same origin", http.MethodPost, "/api/config", local, host, "http://127.0.0.1:8384", true},
		{"localhost origin", http.MethodPost, "/api/config", local, host, "http://localhost:8384", true},
		{"no origin", http.MethodPost, "/api/config", local, host, "", true},
		{"look-alike localhost origin", http.MethodPost, "/api/config", local, host, "http://localhost.attacker.invalid", false},
		{"look-alike IP origin", http.MethodPost, "/api/config", local, host, "http://127.0.0.1.attacker.invalid:8384", false},
		{"other local port", http.MethodPost, "/api/config", local, host, "http://127.0.0.1:9999", false},
		{"https origin", http.MethodPost, "/api/config", local, host, "https://127.0.0.1:8384", false},
		{"opaque origin", http.MethodPost, "/api/config", local, host, "null", false},
		{"foreign origin", http.MethodPost, "/api/profiles/active", local, host, "http://evil.example", false},
		{"foreign origin reading", http.MethodGet, "/api/config", local, host, "http://evil.example", true},
	}

	reached := false
	handler := withAccessPolicy(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached = false
			r := httptest.NewRequest(tt.method, tt.path, nil)
			r.RemoteAddr = tt.remoteAddr
			r.Host = tt.host
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			if reached != tt.allowed {
				t.Errorf("request reached handler = %v, want %v (status %d)", reached, tt.allowed, w.Code)
			}
			if !tt.allowed && w.Code != http.StatusForbidden {
				t.Errorf("status = %d, want 403", w.Code)
			}
		})
	}
}

// TestProfilePathsOnly checks that load and save by path only work for listed
// profiles, so the editor cannot read or overwrite other files.
func TestProfilePathsOnly(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(outside, []byte("synthetic secret"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		noProfiles  bool // -config mode: no profile provider
		endpoint    string
		body        map[string]interface{}
		wantStatus  int
		wantContent string // content of outside after the request
	}{
		{"load unlisted file", false, "/api/config/load", map[string]interface{}{"path": outside}, http.StatusForbidden, "synthetic secret"},
		{"load traversal to unlisted file", false, "/api/config/load", map[string]interface{}{"path": filepath.Join(dir, "x", "..", "secret.txt")}, http.StatusForbidden, "synthetic secret"},
		{"save to unlisted file", false, "/api/config", map[string]interface{}{"path": outside, "config": map[string]string{"x": "y"}}, http.StatusForbidden, "synthetic secret"},
		{"load in -config mode", true, "/api/config/load", map[string]interface{}{"path": outside}, http.StatusForbidden, "synthetic secret"},
		{"save in -config mode", true, "/api/config", map[string]interface{}{"path": outside, "config": map[string]string{"x": "y"}}, http.StatusForbidden, "synthetic secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, configProvider, _ := createTestServer(t)
			if tt.noProfiles {
				server.profileProvider = nil
			}
			body, err := json.Marshal(tt.body)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, tt.endpoint, strings.NewReader(string(body)))
			w := httptest.NewRecorder()
			createTestMux(server).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "synthetic secret") {
				t.Error("response disclosed the file content")
			}
			if configProvider.saveCalled {
				t.Error("active config was saved instead")
			}
			data, err := os.ReadFile(outside)
			if err != nil || string(data) != tt.wantContent {
				t.Errorf("file content = %q (%v), want %q", data, err, tt.wantContent)
			}
		})
	}
}
