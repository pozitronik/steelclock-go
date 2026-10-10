package webeditor

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleReload(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		onReload   func() error
		wantStatus int
		wantCalled bool
	}{
		{"reloads", http.MethodPost, func() error { return nil }, http.StatusOK, true},
		{"reload fails", http.MethodPost, func() error { return errors.New("bad config") }, http.StatusInternalServerError, true},
		{"no reload callback", http.MethodPost, nil, http.StatusNotImplemented, false},
		{"wrong method", http.MethodGet, func() error { return nil }, http.StatusMethodNotAllowed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _, _ := createTestServer(t)
			called := false
			server.onReload = nil
			if tt.onReload != nil {
				server.onReload = func() error {
					called = true
					return tt.onReload()
				}
			}

			req := httptest.NewRequest(tt.method, "/api/reload", nil)
			w := httptest.NewRecorder()
			createTestMux(server).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if called != tt.wantCalled {
				t.Errorf("reload called = %v, want %v", called, tt.wantCalled)
			}
		})
	}
}
