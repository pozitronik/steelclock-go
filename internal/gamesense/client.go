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
	SendScreenData(eventName string, bitmapData []int) error
	SendHeartbeat() error
	RemoveGame() error
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

// SendScreenData sends bitmap data to the display
func (c *Client) SendScreenData(eventName string, bitmapData []int) error {
	if len(bitmapData) != 640 {
		return fmt.Errorf("invalid bitmap size: expected 640 bytes, got %d", len(bitmapData))
	}

	payload := map[string]interface{}{
		"game":  c.gameName,
		"event": eventName,
		"data": map[string]interface{}{
			"frame": map[string]interface{}{
				"image-data-128x40": bitmapData,
			},
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
