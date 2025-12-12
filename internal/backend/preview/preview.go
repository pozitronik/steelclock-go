// Package preview provides a display backend for web-based preview.
// This backend stores rendered frames in memory and broadcasts them
// to connected WebSocket clients for live preview in the web editor.
package preview

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/pozitronik/steelclock-go/internal/backend"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/display"
)

// Priority for auto-selection (highest = tried last, preview should not be auto-selected)
const Priority = 1000

func init() {
	backend.Register("preview", newBackend, Priority)
}

// Config holds preview backend configuration
type Config struct {
	// TargetFPS limits the frame rate sent to clients (0 = unlimited)
	TargetFPS int
	// Width is the display width in pixels
	Width int
	// Height is the display height in pixels
	Height int
}

// Client implements display.Backend for web preview
type Client struct {
	config Config

	// Frame storage
	mu          sync.RWMutex
	lastFrame   []byte    // Packed bits (1 bit per pixel)
	lastUpdate  time.Time // Time of last frame update
	frameNumber uint64    // Incrementing frame counter

	// WebSocket broadcast
	subscribers   map[*subscriber]struct{}
	subscribersMu sync.RWMutex

	// Rate limiting
	minFrameInterval time.Duration
	lastBroadcast    time.Time
}

// subscriber represents a connected WebSocket client
type subscriber struct {
	conn     *websocket.Conn
	ctx      context.Context
	cancel   context.CancelFunc
	sendChan chan []byte
}

// FrameMessage is the JSON structure sent to WebSocket clients
type FrameMessage struct {
	Type        string `json:"type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Frame       []byte `json:"frame"` // Packed bits
	FrameNumber uint64 `json:"frame_number"`
	Timestamp   int64  `json:"timestamp"` // Unix milliseconds
}

// newBackend creates a preview backend from configuration
func newBackend(cfg *config.Config) (display.Backend, error) {
	previewCfg := Config{
		TargetFPS: 30, // Default 30 FPS
		Width:     cfg.Display.Width,
		Height:    cfg.Display.Height,
	}

	// Override from config if available
	if cfg.Preview != nil {
		if cfg.Preview.TargetFPS > 0 {
			previewCfg.TargetFPS = cfg.Preview.TargetFPS
		}
	}

	client := NewClient(previewCfg)

	log.Printf("Preview backend created (width: %d, height: %d, target FPS: %d)",
		previewCfg.Width, previewCfg.Height, previewCfg.TargetFPS)

	return client, nil
}

// NewClient creates a new preview client
func NewClient(cfg Config) *Client {
	var minInterval time.Duration
	if cfg.TargetFPS > 0 {
		minInterval = time.Second / time.Duration(cfg.TargetFPS)
	}

	return &Client{
		config:           cfg,
		subscribers:      make(map[*subscriber]struct{}),
		minFrameInterval: minInterval,
	}
}

// SendScreenData implements display.FrameSender
func (c *Client) SendScreenData(_ string, bitmapData []byte) error {
	c.storeAndBroadcast(bitmapData)
	return nil
}

// SendScreenDataMultiRes implements display.FrameSender
func (c *Client) SendScreenDataMultiRes(_ string, resolutionData map[string][]byte) error {
	// Use the first (main) resolution data
	for _, data := range resolutionData {
		c.storeAndBroadcast(data)
		return nil
	}
	return nil
}

// SendMultipleScreenData implements display.FrameSender
func (c *Client) SendMultipleScreenData(_ string, frames [][]byte) error {
	// Send only the last frame for preview
	if len(frames) > 0 {
		c.storeAndBroadcast(frames[len(frames)-1])
	}
	return nil
}

// SendHeartbeat implements display.HeartbeatSender
func (c *Client) SendHeartbeat() error {
	// No-op for preview backend
	return nil
}

// SupportsMultipleEvents implements display.BatchCapability
func (c *Client) SupportsMultipleEvents() bool {
	return false
}

// RegisterGame implements display.GameRegistrar
func (c *Client) RegisterGame(_ string, _ int) error {
	// No-op for preview backend
	return nil
}

// BindScreenEvent implements display.GameRegistrar
func (c *Client) BindScreenEvent(_ string, _ string) error {
	// No-op for preview backend
	return nil
}

// RemoveGame implements display.GameRegistrar
func (c *Client) RemoveGame() error {
	// No-op for preview backend
	return nil
}

// storeAndBroadcast stores the frame and broadcasts to subscribers
func (c *Client) storeAndBroadcast(data []byte) {
	now := time.Now()

	// Rate limiting
	if c.minFrameInterval > 0 {
		c.mu.RLock()
		elapsed := now.Sub(c.lastBroadcast)
		c.mu.RUnlock()

		if elapsed < c.minFrameInterval {
			// Store frame but don't broadcast yet
			c.mu.Lock()
			c.lastFrame = append(c.lastFrame[:0], data...)
			c.lastUpdate = now
			c.frameNumber++
			c.mu.Unlock()
			return
		}
	}

	// Store frame
	c.mu.Lock()
	c.lastFrame = append(c.lastFrame[:0], data...)
	c.lastUpdate = now
	c.frameNumber++
	frameNum := c.frameNumber
	c.lastBroadcast = now
	c.mu.Unlock()

	// Broadcast to all subscribers
	c.broadcast(data, frameNum, now)
}

// broadcast sends frame to all connected subscribers
func (c *Client) broadcast(data []byte, frameNum uint64, timestamp time.Time) {
	msg := FrameMessage{
		Type:        "frame",
		Width:       c.config.Width,
		Height:      c.config.Height,
		Frame:       data,
		FrameNumber: frameNum,
		Timestamp:   timestamp.UnixMilli(),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Preview: failed to marshal frame message: %v", err)
		return
	}

	c.subscribersMu.RLock()
	defer c.subscribersMu.RUnlock()

	for sub := range c.subscribers {
		select {
		case sub.sendChan <- msgBytes:
			// Message queued
		default:
			// Channel full, skip this frame for slow client
		}
	}
}

// GetCurrentFrame returns the current frame data for static preview
func (c *Client) GetCurrentFrame() ([]byte, uint64, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastFrame == nil {
		return nil, 0, time.Time{}
	}

	frame := make([]byte, len(c.lastFrame))
	copy(frame, c.lastFrame)
	return frame, c.frameNumber, c.lastUpdate
}

// GetConfig returns the preview configuration
func (c *Client) GetConfig() Config {
	return c.config
}

// SubscriberCount returns the number of connected WebSocket clients
func (c *Client) SubscriberCount() int {
	c.subscribersMu.RLock()
	defer c.subscribersMu.RUnlock()
	return len(c.subscribers)
}

// HandleWebSocket handles a WebSocket connection for live preview
func (c *Client) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // Allow all origins for local use
	})
	if err != nil {
		log.Printf("Preview: failed to accept WebSocket: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	sub := &subscriber{
		conn:     conn,
		ctx:      ctx,
		cancel:   cancel,
		sendChan: make(chan []byte, 10), // Buffer a few frames
	}

	// Register subscriber
	c.subscribersMu.Lock()
	c.subscribers[sub] = struct{}{}
	c.subscribersMu.Unlock()

	log.Printf("Preview: client connected (total: %d)", c.SubscriberCount())

	// Send current frame immediately if available
	frame, frameNum, timestamp := c.GetCurrentFrame()
	if frame != nil {
		msg := FrameMessage{
			Type:        "frame",
			Width:       c.config.Width,
			Height:      c.config.Height,
			Frame:       frame,
			FrameNumber: frameNum,
			Timestamp:   timestamp.UnixMilli(),
		}
		if msgBytes, err := json.Marshal(msg); err == nil {
			select {
			case sub.sendChan <- msgBytes:
			default:
			}
		}
	}

	// Send config info
	configMsg := map[string]interface{}{
		"type":       "config",
		"width":      c.config.Width,
		"height":     c.config.Height,
		"target_fps": c.config.TargetFPS,
	}
	if configBytes, err := json.Marshal(configMsg); err == nil {
		select {
		case sub.sendChan <- configBytes:
		default:
		}
	}

	// Start send loop
	go c.sendLoop(sub)

	// Read loop (for future commands like pause/resume)
	c.readLoop(sub)

	// Cleanup
	cancel()
	c.subscribersMu.Lock()
	delete(c.subscribers, sub)
	c.subscribersMu.Unlock()

	_ = conn.Close(websocket.StatusNormalClosure, "")
	log.Printf("Preview: client disconnected (total: %d)", c.SubscriberCount())
}

// sendLoop sends messages to a WebSocket client
func (c *Client) sendLoop(sub *subscriber) {
	for {
		select {
		case <-sub.ctx.Done():
			return
		case msg := <-sub.sendChan:
			ctx, cancel := context.WithTimeout(sub.ctx, 5*time.Second)
			err := sub.conn.Write(ctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				sub.cancel()
				return
			}
		}
	}
}

// readLoop reads messages from a WebSocket client
func (c *Client) readLoop(sub *subscriber) {
	for {
		_, _, err := sub.conn.Read(sub.ctx)
		if err != nil {
			return
		}
		// Future: handle client commands (pause, resume, request frame, etc.)
	}
}
