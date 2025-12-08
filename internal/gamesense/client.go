package gamesense

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// API defines the interface for GameSense API operations
type API interface {
	RegisterGame(developer string, deinitializeTimerMs int) error
	BindScreenEvent(eventName, deviceType string) error
	SendScreenData(eventName string, bitmapData []byte) error
	SendScreenDataMultiRes(eventName string, resolutionData map[string][]byte) error
	SendHeartbeat() error
	RemoveGame() error
	SupportsMultipleEvents() bool
	SendMultipleScreenData(eventName string, frames [][]byte) error
}

// Client is a GameSense API client
type Client struct {
	baseURL         string
	gameName        string
	gameDisplayName string
	httpClient      *http.Client
}

// Ensure Client implements API
var _ API = (*Client)(nil)

// NewClient creates a new GameSense API client
func NewClient(gameName, gameDisplayName string) (*Client, error) {
	address, err := DiscoverServer()
	if err != nil {
		return nil, err
	}

	//goland:noinspection HttpUrlsUsage
	return &Client{
		baseURL:         "http://" + address,
		gameName:        gameName,
		gameDisplayName: gameDisplayName,
		httpClient: &http.Client{
			Timeout: 500 * time.Millisecond,
		},
	}, nil
}

// RegisterGame registers the application with SteelSeries Engine
func (c *Client) RegisterGame(developer string, deinitializeTimerMs int) error {
	payload := map[string]interface{}{
		"game":              c.gameName,
		"game_display_name": c.gameDisplayName,
		"developer":         developer,
	}

	// Add deinitialize_timer_length_ms if specified (optional field)
	if deinitializeTimerMs > 0 {
		payload["deinitialize_timer_length_ms"] = deinitializeTimerMs
	}

	if err := c.post("/game_metadata", payload); err != nil {
		return fmt.Errorf("failed to register game: %w", err)
	}

	log.Printf("Game registered: %s", c.gameName)
	return nil
}

// BindScreenEvent creates a screen binding for displaying images
func (c *Client) BindScreenEvent(eventName, deviceType string) error {
	// Create default blank screen (640 zeros for 128x40)
	blankScreen := make([]int, 640)

	payload := map[string]interface{}{
		"game":           c.gameName,
		"event":          eventName,
		"value_optional": true,
		"handlers": []map[string]interface{}{
			{
				"device-type": deviceType,
				"zone":        "one",
				"mode":        "screen",
				"datas": []map[string]interface{}{
					{
						"has-text":   false,
						"image-data": blankScreen,
					},
				},
			},
		},
	}

	if err := c.post("/bind_game_event", payload); err != nil {
		return fmt.Errorf("failed to bind event: %w", err)
	}

	log.Printf("Event bound: %s", eventName)
	return nil
}

// bytesToInts converts []byte to []int for JSON serialization.
// GameSense API requires an array of integers in JSON format.
func bytesToInts(data []byte) []int {
	result := make([]int, len(data))
	for i, b := range data {
		result[i] = int(b)
	}
	return result
}

// SendScreenData sends bitmap data to the display
func (c *Client) SendScreenData(eventName string, bitmapData []byte) error {
	if len(bitmapData) != 640 {
		return fmt.Errorf("invalid bitmap size: expected 640 bytes, got %d", len(bitmapData))
	}

	payload := map[string]interface{}{
		"game":  c.gameName,
		"event": eventName,
		"data": map[string]interface{}{
			"frame": map[string]interface{}{
				"image-data-128x40": bytesToInts(bitmapData),
			},
		},
	}

	// Fire and forget pattern - don't check response
	_ = c.post("/game_event", payload)
	return nil
}

// SendScreenDataMultiRes sends bitmap data for multiple resolutions in a single frame
// resolutionData maps resolution keys (e.g., "image-data-128x40") to bitmap data
func (c *Client) SendScreenDataMultiRes(eventName string, resolutionData map[string][]byte) error {
	if len(resolutionData) == 0 {
		return fmt.Errorf("no resolution data provided")
	}

	// Build frame with all resolutions, converting to []int for JSON
	frameData := make(map[string]interface{})
	for key, bitmapData := range resolutionData {
		frameData[key] = bytesToInts(bitmapData)
	}

	payload := map[string]interface{}{
		"game":  c.gameName,
		"event": eventName,
		"data": map[string]interface{}{
			"frame": frameData,
		},
	}

	// Fire and forget pattern - don't check response
	_ = c.post("/game_event", payload)
	return nil
}

// SendHeartbeat sends a heartbeat to keep the game alive
func (c *Client) SendHeartbeat() error {
	payload := map[string]string{
		"game": c.gameName,
	}

	if err := c.post("/game_heartbeat", payload); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	return nil
}

// SupportsMultipleEvents checks if the GameSense API supports multiple event batching
// Returns true if supported (200 OK), false if not supported (404) or error
func (c *Client) SupportsMultipleEvents() bool {
	req, err := http.NewRequest("GET", c.baseURL+"/supports_multiple_game_events", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Failed to close response body: %v", closeErr)
		}
	}()

	supported := resp.StatusCode == http.StatusOK
	if supported {
		log.Println("Multiple event batching supported by GameSense API")
	} else {
		log.Println("Multiple event batching NOT supported by GameSense API (will use single-frame mode)")
	}

	return supported
}

// SendMultipleScreenData sends multiple bitmap frames in a single request
// This reduces HTTP overhead when supported by the GameSense API
func (c *Client) SendMultipleScreenData(eventName string, bitmaps [][]byte) error {
	if len(bitmaps) == 0 {
		return nil
	}

	// Validate all bitmaps
	for i, bitmap := range bitmaps {
		if len(bitmap) != 640 {
			return fmt.Errorf("invalid bitmap %d size: expected 640 bytes, got %d", i, len(bitmap))
		}
	}

	// Build events array, converting to []int for JSON
	events := make([]map[string]interface{}, len(bitmaps))
	for i, bitmap := range bitmaps {
		events[i] = map[string]interface{}{
			"event": eventName,
			"data": map[string]interface{}{
				"frame": map[string]interface{}{
					"image-data-128x40": bytesToInts(bitmap),
				},
			},
		}
	}

	payload := map[string]interface{}{
		"game":   c.gameName,
		"events": events,
	}

	// Fire and forget pattern - don't check response
	_ = c.post("/multiple_game_events", payload)
	return nil
}

// RemoveGame unregisters the game from SteelSeries Engine
func (c *Client) RemoveGame() error {
	payload := map[string]string{
		"game": c.gameName,
	}

	if err := c.post("/remove_game", payload); err != nil {
		return fmt.Errorf("failed to remove game: %w", err)
	}

	log.Printf("Game removed: %s", c.gameName)
	return nil
}

// GameName returns the game name used by this client
func (c *Client) GameName() string {
	return c.gameName
}

// post sends a POST request to the GameSense API
func (c *Client) post(endpoint string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+endpoint, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}
