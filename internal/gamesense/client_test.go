package gamesense

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	err := client.RegisterGame("Test Developer")
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
