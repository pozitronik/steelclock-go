package spotify

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleCallback(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		query      string
		wantStatus int
		wantResult bool   // a result is handed to StartAuth
		wantCode   string // code of a successful result
		wantErr    bool   // the result carries an error
	}{
		{"valid code", http.MethodGet, "state=expected&code=abc", http.StatusOK, true, "abc", false},
		{"provider error", http.MethodGet, "state=expected&error=access_denied", http.StatusOK, true, "", true},
		{"missing code", http.MethodGet, "state=expected", http.StatusOK, true, "", true},
		{"wrong state", http.MethodGet, "state=wrong&code=abc", http.StatusBadRequest, false, "", false},
		{"missing state", http.MethodGet, "code=abc", http.StatusBadRequest, false, "", false},
		{"error with wrong state", http.MethodGet, "state=wrong&error=access_denied", http.StatusBadRequest, false, "", false},
		{"POST request", http.MethodPost, "state=expected&code=abc", http.StatusMethodNotAllowed, false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPKCEAuth("client", 0)
			p.state = "expected"
			w := httptest.NewRecorder()
			p.handleCallback(w, httptest.NewRequest(tt.method, "/callback?"+tt.query, nil))

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			select {
			case result := <-p.result:
				if !tt.wantResult {
					t.Fatalf("unexpected result %+v: the pending attempt was ended", result)
				}
				if (result.err != nil) != tt.wantErr {
					t.Errorf("result error = %v, wantErr %v", result.err, tt.wantErr)
				}
				if result.code != tt.wantCode {
					t.Errorf("result code = %q, want %q", result.code, tt.wantCode)
				}
			default:
				if tt.wantResult {
					t.Error("no result was handed to StartAuth")
				}
			}
		})
	}
}

func TestHandleCallback_EscapesErrorDescription(t *testing.T) {
	p := NewPKCEAuth("client", 0)
	p.state = "expected"
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet,
		"/callback?state=expected&error=access_denied&error_description=%3Cscript%3Ealert(1)%3C%2Fscript%3E", nil)
	p.handleCallback(w, r)

	body := w.Body.String()
	if strings.Contains(body, "<script>") {
		t.Errorf("response contains unescaped markup:\n%s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Errorf("response does not contain the escaped description:\n%s", body)
	}
	if csp := w.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("Content-Security-Policy = %q, want a restrictive policy", csp)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestHandleCallback_RepeatedCallbackDoesNotBlock(t *testing.T) {
	p := NewPKCEAuth("client", 0)
	p.state = "expected"
	p.handleCallback(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/callback?state=expected&code=first", nil))

	done := make(chan struct{})
	go func() {
		p.handleCallback(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/callback?state=expected&code=second", nil))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		<-p.result // release the blocked handler before failing
		<-done
		t.Fatal("second callback blocked on the result channel")
	}

	if result := <-p.result; result.code != "first" {
		t.Errorf("result code = %q, want the first callback's code", result.code)
	}
}

// roundTripFunc lets a function serve as an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRefreshToken_StatusHandling(t *testing.T) {
	tests := []struct {
		name            string
		status          int
		body            string
		wantErr         bool
		wantCredentials bool
		wantAccess      string
	}{
		{"success", http.StatusOK, `{"access_token":"new-access","expires_in":3600}`, false, true, "new-access"},
		{"revoked refresh token", http.StatusBadRequest, `{"error":"invalid_grant"}`, true, false, ""},
		{"unauthorized client", http.StatusUnauthorized, `{"error":"invalid_client"}`, true, false, ""},
		{"service unavailable", http.StatusServiceUnavailable, `{"error":"temporarily_unavailable"}`, true, true, "old-access"},
		{"server error", http.StatusInternalServerError, ``, true, true, "old-access"},
		{"rate limited", http.StatusTooManyRequests, ``, true, true, "old-access"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryTokenStore()
			token := &TokenInfo{AccessToken: "old-access", RefreshToken: "refresh", ExpiresAt: time.Now().Add(-time.Minute)}
			if err := store.Save(token); err != nil {
				t.Fatal(err)
			}
			c := &httpClient{
				clientID:   "client",
				tokenStore: store,
				token:      token,
				httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.status,
						Body:       io.NopCloser(strings.NewReader(tt.body)),
						Header:     http.Header{},
					}, nil
				})},
			}

			err := c.RefreshToken()
			if (err != nil) != tt.wantErr {
				t.Fatalf("RefreshToken() error = %v, wantErr %v", err, tt.wantErr)
			}

			saved, err := store.Load()
			if err != nil {
				t.Fatal(err)
			}
			hasCredentials := c.token != nil && saved != nil
			if hasCredentials != tt.wantCredentials {
				t.Fatalf("credentials kept = %v (memory %v, store %v), want %v", hasCredentials, c.token != nil, saved != nil, tt.wantCredentials)
			}
			if tt.wantCredentials && c.token.AccessToken != tt.wantAccess {
				t.Errorf("access token = %q, want %q", c.token.AccessToken, tt.wantAccess)
			}
		})
	}
}
