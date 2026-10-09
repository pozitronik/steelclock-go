package spotify

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// PKCEAuth handles OAuth 2.0 PKCE authentication flow.
type PKCEAuth struct {
	clientID     string
	callbackPort int
	redirectURI  string

	codeVerifier  string
	codeChallenge string
	state         string

	mu     sync.Mutex
	server *http.Server
	result chan authResult
}

// authResult holds the result of the OAuth callback.
type authResult struct {
	code string
	err  error
}

// NewPKCEAuth creates a new PKCE authentication handler.
func NewPKCEAuth(clientID string, callbackPort int) *PKCEAuth {
	return &PKCEAuth{
		clientID:     clientID,
		callbackPort: callbackPort,
		redirectURI:  fmt.Sprintf("http://127.0.0.1:%d/callback", callbackPort),
		result:       make(chan authResult, 1),
	}
}

// StartAuth initiates the OAuth PKCE flow.
// This will:
// 1. Generate PKCE code verifier and challenge
// 2. Start local callback server
// 3. Open browser for authorization
// 4. Wait for callback with authorization code
// 5. Exchange code for tokens
func (p *PKCEAuth) StartAuth(ctx context.Context) (*TokenInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Generate PKCE parameters
	if err := p.generatePKCE(); err != nil {
		return nil, fmt.Errorf("failed to generate PKCE: %w", err)
	}

	// Generate state for CSRF protection
	p.state = generateRandomString(32)

	// Drop a result left over from an earlier attempt
	select {
	case <-p.result:
	default:
	}

	// Start callback server
	if err := p.startCallbackServer(ctx); err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}

	// Build authorization URL
	authURL := p.buildAuthURL()

	// Open browser
	log.Printf("spotify: opening browser for authorization: %s", authURL)
	if err := openBrowser(authURL); err != nil {
		log.Printf("spotify: failed to open browser: %v", err)
		log.Printf("spotify: please open this URL manually: %s", authURL)
	}

	// Wait for callback or timeout
	select {
	case result := <-p.result:
		p.stopServer()
		if result.err != nil {
			return nil, result.err
		}
		// Exchange code for token
		return p.exchangeCode(result.code)

	case <-ctx.Done():
		p.stopServer()
		return nil, ctx.Err()

	case <-time.After(5 * time.Minute):
		p.stopServer()
		return nil, fmt.Errorf("authorization timed out")
	}
}

// generatePKCE generates the code verifier and challenge.
func (p *PKCEAuth) generatePKCE() error {
	// Generate code verifier (43-128 characters from [A-Za-z0-9-._~])
	p.codeVerifier = generateRandomString(64)

	// Generate code challenge (base64url(SHA256(code_verifier)))
	hash := sha256.Sum256([]byte(p.codeVerifier))
	p.codeChallenge = base64.RawURLEncoding.EncodeToString(hash[:])

	return nil
}

// buildAuthURL constructs the authorization URL.
func (p *PKCEAuth) buildAuthURL() string {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", p.redirectURI)
	params.Set("scope", RequiredScope)
	params.Set("code_challenge_method", "S256")
	params.Set("code_challenge", p.codeChallenge)
	params.Set("state", p.state)

	return AuthURL + "?" + params.Encode()
}

// startCallbackServer starts the local HTTP server for OAuth callback.
func (p *PKCEAuth) startCallbackServer(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", p.handleCallback)

	addr := fmt.Sprintf("127.0.0.1:%d", p.callbackPort)

	// Bind the port synchronously so we detect failures (port busy, firewall
	// block, etc.) immediately instead of losing them in a goroutine.
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// The goroutines below use their own reference: stopServer sets p.server
	// to nil while they may still be running.
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}
	p.server = server

	go func() {
		if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			log.Printf("spotify: callback server error: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("spotify: callback server shutdown error: %v", err)
		}
	}()

	return nil
}

// stopServer stops the callback server.
func (p *PKCEAuth) stopServer() {
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = p.server.Shutdown(ctx)
		p.server = nil
	}
}

// handleCallback handles the OAuth callback request.
// Requests without the state of the pending attempt are rejected without
// ending that attempt, since anything (e.g. a web page) can send requests to
// this local server. Only the first valid callback is used.
func (p *PKCEAuth) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	if state := query.Get("state"); state == "" || state != p.state {
		p.writeCallbackResponse(w, http.StatusBadRequest, false, "Invalid or expired authorization request.")
		return
	}

	if errParam := query.Get("error"); errParam != "" {
		errDesc := query.Get("error_description")
		p.publishResult(authResult{err: fmt.Errorf("authorization error: %s - %s", errParam, errDesc)})
		p.writeCallbackResponse(w, http.StatusOK, false, "Authorization failed: "+errDesc)
		return
	}

	code := query.Get("code")
	if code == "" {
		p.publishResult(authResult{err: fmt.Errorf("no authorization code received")})
		p.writeCallbackResponse(w, http.StatusOK, false, "No authorization code received")
		return
	}

	p.publishResult(authResult{code: code})
	p.writeCallbackResponse(w, http.StatusOK, true, "Authorization successful! You can close this window.")
}

// publishResult hands the callback result to StartAuth. Only the first result
// is kept; later ones (e.g. a reloaded callback page) are dropped instead of
// blocking the request.
func (p *PKCEAuth) publishResult(result authResult) {
	select {
	case p.result <- result:
	default:
	}
}

// callbackPage renders the callback response. html/template escapes the
// message, which may contain text from the request URL.
var callbackPage = template.Must(template.New("callback").Parse(`<!DOCTYPE html>
<html>
<head>
    <title>Spotify Authorization</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: #191414;
            color: white;
        }
        .container {
            text-align: center;
            padding: 40px;
        }
        .icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        .message {
            font-size: 18px;
        }
        .success { color: #2ecc71; }
        .error { color: #e74c3c; }
    </style>
</head>
<body>
    <div class="container">
        {{if .Success}}<div class="icon success">&#10004;</div>{{else}}<div class="icon error">&#10006;</div>{{end}}
        <div class="message {{if .Success}}success{{else}}error{{end}}">{{.Message}}</div>
    </div>
</body>
</html>`))

// writeCallbackResponse writes the callback HTML response.
func (p *PKCEAuth) writeCallbackResponse(w http.ResponseWriter, status int, success bool, message string) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)

	data := struct {
		Success bool
		Message string
	}{success, message}
	if err := callbackPage.Execute(w, data); err != nil {
		log.Printf("spotify: failed to write callback response: %v", err)
	}
}

// exchangeCode exchanges the authorization code for tokens.
func (p *PKCEAuth) exchangeCode(code string) (*TokenInfo, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", p.redirectURI)
	data.Set("client_id", p.clientID)
	data.Set("code_verifier", p.codeVerifier)

	req, err := http.NewRequest("POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &TokenInfo{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		Scope:        tokenResp.Scope,
	}, nil
}

// generateRandomString generates a random string of the specified length.
// Uses characters from the PKCE unreserved character set: [A-Za-z0-9-._~]
func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to less secure random if crypto/rand fails
		log.Printf("spotify: crypto/rand failed, using fallback: %v", err)
		for i := range bytes {
			bytes[i] = charset[i%len(charset)]
		}
		return string(bytes)
	}

	for i := range bytes {
		bytes[i] = charset[bytes[i]%byte(len(charset))]
	}
	return string(bytes)
}

// openBrowser opens the URL in the default browser.
func openBrowser(rawURL string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use rundll32 to avoid cmd.exe, which interprets & as a command separator
		// and has complex quoting rules that conflict with Go's argument escaping.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default: // Linux and others
		cmd = exec.Command("xdg-open", rawURL)
	}

	return cmd.Start()
}
