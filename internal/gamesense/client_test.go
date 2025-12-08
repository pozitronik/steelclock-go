package gamesense

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
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
	bitmapData := make([]byte, 640)
	for i := range bitmapData {
		bitmapData[i] = byte(i % 256)
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
	bitmapData := make([]byte, 100)

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

			bitmapData := make([]byte, tc.bitmapSize)
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
	largeBitmap := make([]byte, 640)
	for i := range largeBitmap {
		largeBitmap[i] = byte(i % 256)
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

func TestGameName(t *testing.T) {
	client := &Client{
		gameName:        "MY_GAME",
		gameDisplayName: "My Game",
	}

	if client.GameName() != "MY_GAME" {
		t.Errorf("GameName() = %s, want MY_GAME", client.GameName())
	}
}

func TestSupportsMultipleEvents_Supported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/supports_multiple_game_events" && r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		gameName:   "TEST_GAME",
		httpClient: &http.Client{},
	}

	if !client.SupportsMultipleEvents() {
		t.Error("SupportsMultipleEvents() should return true when server returns 200")
	}
}

func TestSupportsMultipleEvents_NotSupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/supports_multiple_game_events" {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		gameName:   "TEST_GAME",
		httpClient: &http.Client{},
	}

	if client.SupportsMultipleEvents() {
		t.Error("SupportsMultipleEvents() should return false when server returns 404")
	}
}

func TestSupportsMultipleEvents_ServerError(t *testing.T) {
	client := &Client{
		baseURL:    "http://localhost:1", // Invalid port
		gameName:   "TEST_GAME",
		httpClient: &http.Client{Timeout: 100 * time.Millisecond},
	}

	if client.SupportsMultipleEvents() {
		t.Error("SupportsMultipleEvents() should return false on connection error")
	}
}

func TestSendScreenDataMultiRes(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/game_event" {
			_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		gameName:   "TEST_GAME",
		httpClient: &http.Client{},
	}

	resolutionData := map[string][]byte{
		"image-data-128x40": make([]byte, 640),
		"image-data-128x36": make([]byte, 576),
	}

	err := client.SendScreenDataMultiRes("TEST_EVENT", resolutionData)
	if err != nil {
		t.Errorf("SendScreenDataMultiRes() error = %v", err)
	}

	// Verify payload structure
	if receivedPayload["game"] != "TEST_GAME" {
		t.Errorf("game = %v, want TEST_GAME", receivedPayload["game"])
	}
	if receivedPayload["event"] != "TEST_EVENT" {
		t.Errorf("event = %v, want TEST_EVENT", receivedPayload["event"])
	}

	// Verify frame contains resolution data
	data := receivedPayload["data"].(map[string]interface{})
	frame := data["frame"].(map[string]interface{})
	if _, ok := frame["image-data-128x40"]; !ok {
		t.Error("frame should contain image-data-128x40")
	}
	if _, ok := frame["image-data-128x36"]; !ok {
		t.Error("frame should contain image-data-128x36")
	}
}

func TestSendScreenDataMultiRes_Empty(t *testing.T) {
	client := &Client{
		baseURL:    "http://localhost:1234",
		gameName:   "TEST_GAME",
		httpClient: &http.Client{},
	}

	err := client.SendScreenDataMultiRes("TEST_EVENT", map[string][]byte{})
	if err == nil {
		t.Error("SendScreenDataMultiRes() with empty data should return error")
	}
}

func TestSendMultipleScreenData(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/multiple_game_events" {
			_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		gameName:   "TEST_GAME",
		httpClient: &http.Client{},
	}

	frames := [][]byte{
		make([]byte, 640),
		make([]byte, 640),
		make([]byte, 640),
	}

	err := client.SendMultipleScreenData("TEST_EVENT", frames)
	if err != nil {
		t.Errorf("SendMultipleScreenData() error = %v", err)
	}

	// Verify payload structure
	if receivedPayload["game"] != "TEST_GAME" {
		t.Errorf("game = %v, want TEST_GAME", receivedPayload["game"])
	}

	events := receivedPayload["events"].([]interface{})
	if len(events) != 3 {
		t.Errorf("events length = %d, want 3", len(events))
	}
}

func TestSendMultipleScreenData_Empty(t *testing.T) {
	client := &Client{
		baseURL:    "http://localhost:1234",
		gameName:   "TEST_GAME",
		httpClient: &http.Client{},
	}

	// Empty frames should return nil (no-op)
	err := client.SendMultipleScreenData("TEST_EVENT", [][]byte{})
	if err != nil {
		t.Errorf("SendMultipleScreenData() with empty frames should not error, got: %v", err)
	}
}

func TestSendMultipleScreenData_InvalidSize(t *testing.T) {
	client := &Client{
		baseURL:    "http://localhost:1234",
		gameName:   "TEST_GAME",
		httpClient: &http.Client{},
	}

	frames := [][]byte{
		make([]byte, 640),
		make([]byte, 100), // Invalid size
	}

	err := client.SendMultipleScreenData("TEST_EVENT", frames)
	if err == nil {
		t.Error("SendMultipleScreenData() with invalid bitmap size should return error")
	}
}

func TestRegisterGame_WithDeinitializeTimer(t *testing.T) {
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
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.RegisterGame("Developer", 15000)
	if err != nil {
		t.Fatalf("RegisterGame() error = %v", err)
	}

	// Verify deinitialize timer was included
	timer, ok := receivedPayload["deinitialize_timer_length_ms"]
	if !ok {
		t.Error("RegisterGame() with timer > 0 should include deinitialize_timer_length_ms")
	}
	if timer.(float64) != 15000 {
		t.Errorf("deinitialize_timer_length_ms = %v, want 15000", timer)
	}
}

func TestRegisterGame_WithoutDeinitializeTimer(t *testing.T) {
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
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.RegisterGame("Developer", 0)
	if err != nil {
		t.Fatalf("RegisterGame() error = %v", err)
	}

	// Verify deinitialize timer was NOT included when 0
	if _, ok := receivedPayload["deinitialize_timer_length_ms"]; ok {
		t.Error("RegisterGame() with timer = 0 should not include deinitialize_timer_length_ms")
	}
}

// TestFindCorePropsPath_WithPROGRAMDATA tests the PROGRAMDATA env var path
func TestFindCorePropsPath_WithPROGRAMDATA(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	ssDir := filepath.Join(tmpDir, "SteelSeries", "SteelSeries Engine 3")
	if err := os.MkdirAll(ssDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	corePropsPath := filepath.Join(ssDir, "coreProps.json")
	if err := os.WriteFile(corePropsPath, []byte(`{"address": "localhost:12345"}`), 0644); err != nil {
		t.Fatalf("Failed to write coreProps.json: %v", err)
	}

	// Save and restore PROGRAMDATA
	oldProgramData := os.Getenv("PROGRAMDATA")
	defer func() { os.Setenv("PROGRAMDATA", oldProgramData) }()

	os.Setenv("PROGRAMDATA", tmpDir)

	path, err := findCorePropsPath()
	if err != nil {
		t.Errorf("findCorePropsPath() error = %v", err)
	}
	if path != corePropsPath {
		t.Errorf("findCorePropsPath() = %s, want %s", path, corePropsPath)
	}
}

// TestFindCorePropsPath_NoPROGRAMDATA tests when PROGRAMDATA is empty
func TestFindCorePropsPath_NoPROGRAMDATA(t *testing.T) {
	// Save and restore PROGRAMDATA
	oldProgramData := os.Getenv("PROGRAMDATA")
	defer func() { os.Setenv("PROGRAMDATA", oldProgramData) }()

	// Save and restore defaultFallbackPath
	oldFallbackPath := defaultFallbackPath
	defer func() { defaultFallbackPath = oldFallbackPath }()

	os.Setenv("PROGRAMDATA", "")
	defaultFallbackPath = "/nonexistent/fallback/path/coreProps.json"

	// Should fail since neither PROGRAMDATA nor fallback path exist
	_, err := findCorePropsPath()
	if err == nil {
		t.Error("findCorePropsPath() should error when no valid path found")
	}
}

// TestFindCorePropsPath_PROGRAMDATANotFound tests when PROGRAMDATA exists but file doesn't
func TestFindCorePropsPath_PROGRAMDATANotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore PROGRAMDATA
	oldProgramData := os.Getenv("PROGRAMDATA")
	defer func() { os.Setenv("PROGRAMDATA", oldProgramData) }()

	// Save and restore defaultFallbackPath
	oldFallbackPath := defaultFallbackPath
	defer func() { defaultFallbackPath = oldFallbackPath }()

	os.Setenv("PROGRAMDATA", tmpDir)
	defaultFallbackPath = "/nonexistent/fallback/path/coreProps.json"

	// Directory exists but no coreProps.json
	_, err := findCorePropsPath()
	if err == nil {
		t.Error("findCorePropsPath() should error when file not found")
	}
}

// TestSupportsMultipleEvents_RequestCreationError tests error when creating request fails
// This is difficult to trigger naturally, but we can test with invalid URL
func TestSupportsMultipleEvents_InvalidURL(t *testing.T) {
	client := &Client{
		baseURL:    "://invalid-url", // Invalid URL scheme
		gameName:   "TEST_GAME",
		httpClient: &http.Client{},
	}

	// Should return false on request creation error
	if client.SupportsMultipleEvents() {
		t.Error("SupportsMultipleEvents() should return false on invalid URL")
	}
}

// TestPost_InvalidURL tests post with invalid URL
func TestPost_InvalidURL(t *testing.T) {
	client := &Client{
		baseURL:    "://invalid-url",
		gameName:   "TEST_GAME",
		httpClient: &http.Client{},
	}

	err := client.RegisterGame("Developer", 0)
	if err == nil {
		t.Error("RegisterGame() with invalid URL should return error")
	}
}
