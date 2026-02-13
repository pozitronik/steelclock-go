package spotify

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

	return SpotifyAuthURL + "?" + params.Encode()
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

	p.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := p.server.Serve(listener); err != http.ErrServerClosed {
			log.Printf("spotify: callback server error: %v", err)
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
func (p *PKCEAuth) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Check for error
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		p.result <- authResult{err: fmt.Errorf("authorization error: %s - %s", errParam, errDesc)}
		p.writeCallbackResponse(w, false, "Authorization failed: "+errDesc)
		return
	}

	// Verify state
	state := r.URL.Query().Get("state")
	if state != p.state {
		p.result <- authResult{err: fmt.Errorf("state mismatch: possible CSRF attack")}
		p.writeCallbackResponse(w, false, "Security error: state mismatch")
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		p.result <- authResult{err: fmt.Errorf("no authorization code received")}
		p.writeCallbackResponse(w, false, "No authorization code received")
		return
	}

	p.result <- authResult{code: code}
	p.writeCallbackResponse(w, true, "Authorization successful! You can close this window.")
}

// writeCallbackResponse writes the callback HTML response.
func (p *PKCEAuth) writeCallbackResponse(w http.ResponseWriter, success bool, message string) {
	w.Header().Set("Content-Type", "text/html")

	color := "#e74c3c" // red for error
	if success {
		color = "#2ecc71" // green for success
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
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
            color: %s;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">%s</div>
        <div class="message">%s</div>
    </div>
</body>
</html>`, color, map[bool]string{true: "&#10004;", false: "&#10006;"}[success], message)

	_, _ = w.Write([]byte(html))
}

// exchangeCode exchanges the authorization code for tokens.
func (p *PKCEAuth) exchangeCode(code string) (*TokenInfo, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", p.redirectURI)
	data.Set("client_id", p.clientID)
	data.Set("code_verifier", p.codeVerifier)

	req, err := http.NewRequest("POST", SpotifyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

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
