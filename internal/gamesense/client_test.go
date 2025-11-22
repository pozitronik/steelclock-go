package gamesense

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewClient(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	if client.gameName != "TEST_GAME" {
		t.Errorf("gameName = %s, want TEST_GAME", client.gameName)
	}
}

func TestRegisterGame(t *testing.T) {
	requestReceived := false
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/game_metadata" {
			requestReceived = true
			_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.RegisterGame("Test Developer", 0)
	if err != nil {
		t.Errorf("RegisterGame() error = %v", err)
	}

	if !requestReceived {
		t.Error("RegisterGame() did not send request")
	}

	if receivedPayload["game"] != "TEST_GAME" {
		t.Errorf("game = %v, want TEST_GAME", receivedPayload["game"])
	}
}

func TestRemoveGame(t *testing.T) {
	requestReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/remove_game" {
			requestReceived = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.RemoveGame()
	if err != nil {
		t.Errorf("RemoveGame() error = %v", err)
	}

	if !requestReceived {
		t.Error("RemoveGame() did not send request")
	}
}

func TestBindScreenEvent(t *testing.T) {
	requestReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bind_game_event" {
			requestReceived = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.BindScreenEvent("TEST_EVENT", "screened-128x40")
	if err != nil {
		t.Errorf("BindScreenEvent() error = %v", err)
	}

	if !requestReceived {
		t.Error("BindScreenEvent() did not send request")
	}
}

func TestSendScreenData(t *testing.T) {
	requestReceived := false
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/game_event" {
			requestReceived = true
			_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	// Create valid 640-byte bitmap
	bitmapData := make([]int, 640)
	for i := range bitmapData {
		bitmapData[i] = i % 256
	}

	err := client.SendScreenData("TEST_EVENT", bitmapData)
	if err != nil {
		t.Errorf("SendScreenData() error = %v", err)
	}

	if !requestReceived {
		t.Error("SendScreenData() did not send request")
	}

	if receivedPayload["game"] != "TEST_GAME" {
		t.Errorf("game = %v, want TEST_GAME", receivedPayload["game"])
	}
}

func TestSendScreenDataInvalidSize(t *testing.T) {
	client := &Client{
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	// Create invalid bitmap (wrong size)
	bitmapData := make([]int, 100)

	err := client.SendScreenData("TEST_EVENT", bitmapData)
	if err == nil {
		t.Error("SendScreenData() with invalid size should return error")
	}
}

func TestSendHeartbeat(t *testing.T) {
	requestReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/game_heartbeat" {
			requestReceived = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.SendHeartbeat()
	if err != nil {
		t.Errorf("SendHeartbeat() error = %v", err)
	}

	if !requestReceived {
		t.Error("SendHeartbeat() did not send request")
	}
}

func TestRegisterGame_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.RegisterGame("Test Developer", 0)
	if err == nil {
		t.Error("RegisterGame() with HTTP 500 should return error")
	}
}

func TestBindScreenEvent_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.BindScreenEvent("TEST_EVENT", "screened-128x40")
	if err == nil {
		t.Error("BindScreenEvent() with HTTP 404 should return error")
	}
}

func TestBindScreenEvent_ValidatesBlankScreen(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bind_game_event" {
			_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.BindScreenEvent("TEST_EVENT", "screened-128x40")
	if err != nil {
		t.Fatalf("BindScreenEvent() error = %v", err)
	}

	// Verify blank screen is 640 zeros
	handlers := receivedPayload["handlers"].([]interface{})
	handler := handlers[0].(map[string]interface{})
	datas := handler["datas"].([]interface{})
	data := datas[0].(map[string]interface{})
	imageData := data["image-data"].([]interface{})

	if len(imageData) != 640 {
		t.Errorf("blank screen size = %d, want 640", len(imageData))
	}

	// Check all zeros
	for i, val := range imageData {
		if val.(float64) != 0 {
			t.Errorf("blank screen[%d] = %v, want 0", i, val)
			break
		}
	}
}

func TestSendHeartbeat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.SendHeartbeat()
	if err == nil {
		t.Error("SendHeartbeat() with HTTP 503 should return error")
	}
}

func TestRemoveGame_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.RemoveGame()
	if err == nil {
		t.Error("RemoveGame() with HTTP 400 should return error")
	}
}

func TestPost_ValidatesStatusCode(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		wantError  bool
	}{
		{"status 200 OK", http.StatusOK, false},
		{"status 201 Created", http.StatusCreated, true},
		{"status 400 Bad Request", http.StatusBadRequest, true},
		{"status 401 Unauthorized", http.StatusUnauthorized, true},
		{"status 404 Not Found", http.StatusNotFound, true},
		{"status 500 Internal Server Error", http.StatusInternalServerError, true},
		{"status 503 Service Unavailable", http.StatusServiceUnavailable, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			client := &Client{
				baseURL:         server.URL,
				gameName:        "TEST_GAME",
				gameDisplayName: "Test Game",
				httpClient:      &http.Client{},
			}

			err := client.RegisterGame("Developer", 0)
			if (err != nil) != tc.wantError {
				t.Errorf("RegisterGame() error = %v, wantError %v", err, tc.wantError)
			}
		})
	}
}

func TestSendScreenData_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		bitmapSize  int
		expectError bool
	}{
		{"valid 640 bytes", 640, false},
		{"empty bitmap", 0, true},
		{"too small 639 bytes", 639, true},
		{"too large 641 bytes", 641, true},
		{"way too large 10000 bytes", 10000, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &Client{
				baseURL:         "http://localhost:1234",
				gameName:        "TEST_GAME",
				gameDisplayName: "Test Game",
				httpClient:      &http.Client{},
			}

			bitmapData := make([]int, tc.bitmapSize)
			err := client.SendScreenData("TEST_EVENT", bitmapData)

			if tc.expectError && err == nil {
				t.Error("SendScreenData() expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("SendScreenData() unexpected error: %v", err)
			}
		})
	}
}

func TestRegisterGame_ValidatesPayload(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/game_metadata" {
			_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "MY_GAME",
		gameDisplayName: "My Game Display",
		httpClient:      &http.Client{},
	}

	err := client.RegisterGame("My Developer", 0)
	if err != nil {
		t.Fatalf("RegisterGame() error = %v", err)
	}

	// Validate all fields
	if receivedPayload["game"] != "MY_GAME" {
		t.Errorf("game = %v, want MY_GAME", receivedPayload["game"])
	}
	if receivedPayload["game_display_name"] != "My Game Display" {
		t.Errorf("game_display_name = %v, want My Game Display", receivedPayload["game_display_name"])
	}
	if receivedPayload["developer"] != "My Developer" {
		t.Errorf("developer = %v, want My Developer", receivedPayload["developer"])
	}
}

func TestPost_ContentTypeHeader(t *testing.T) {
	var contentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	_ = client.RegisterGame("Developer", 0)

	if contentType != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", contentType)
	}
}

func TestNewClient_Success(t *testing.T) {
	// Mock findCorePropsPathFunc to return valid config
	tmpDir := t.TempDir()
	corePropsPath := filepath.Join(tmpDir, "coreProps.json")

	validJSON := `{"address": "localhost:54321"}`
	if err := os.WriteFile(corePropsPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("Failed to write temp coreProps.json: %v", err)
	}

	originalFunc := findCorePropsPathFunc
	findCorePropsPathFunc = func() (string, error) {
		return corePropsPath, nil
	}
	defer func() { findCorePropsPathFunc = originalFunc }()

	client, err := NewClient("MY_GAME", "My Game")
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if client.gameName != "MY_GAME" {
		t.Errorf("gameName = %s, want MY_GAME", client.gameName)
	}

	if client.gameDisplayName != "My Game" {
		t.Errorf("gameDisplayName = %s, want My Game", client.gameDisplayName)
	}

	if client.baseURL != "http://localhost:54321" {
		t.Errorf("baseURL = %s, want http://localhost:54321", client.baseURL)
	}

	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestNewClient_DiscoveryFailure(t *testing.T) {
	originalFunc := findCorePropsPathFunc
	findCorePropsPathFunc = func() (string, error) {
		return "", os.ErrNotExist
	}
	defer func() { findCorePropsPathFunc = originalFunc }()

	_, err := NewClient("GAME", "Game")
	if err == nil {
		t.Error("NewClient() with discovery failure should return error")
	}
}

// TestPost_EmptyResponse tests handling of empty response body
func TestPost_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// No body written
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.RegisterGame("Developer", 0)
	// Should still succeed even with empty body
	if err != nil {
		t.Errorf("RegisterGame() with empty response should not error, got: %v", err)
	}
}

// TestPost_LargePayload tests handling of large request payloads
func TestPost_LargePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	// Send full-size valid bitmap data (640 bytes for 128x40 screen)
	largeBitmap := make([]int, 640)
	for i := range largeBitmap {
		largeBitmap[i] = i % 256
	}

	err := client.SendScreenData("EVENT", largeBitmap)
	if err != nil {
		t.Errorf("SendScreenData() with full bitmap error = %v", err)
	}
}

// TestRemoveGame_Success tests successful game removal
func TestRemoveGame_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/remove_game" {
			t.Errorf("Expected /remove_game path, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.RemoveGame()
	if err != nil {
		t.Errorf("RemoveGame() error = %v", err)
	}
}
