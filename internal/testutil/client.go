// Package testutil provides testing utilities for SteelClock.
// This package should only be imported by _test.go files.
package testutil

import (
	"fmt"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/gamesense"
)

// Frame represents a captured frame with metadata
type Frame struct {
	Data      []byte    // Bitmap data (640 bytes for 128x40)
	EventName string    // Event name used when sending
	Timestamp time.Time // When the frame was received
	Index     int       // Sequential frame number
}

// CallRecord records details of an API call
type CallRecord struct {
	Method    string
	Args      []interface{}
	Timestamp time.Time
	Error     error
}

// TestClient implements gamesense.API for testing purposes.
// It captures frames, tracks calls, and supports error injection.
type TestClient struct {
	mu sync.RWMutex

	// Configuration
	width  int
	height int

	// Frame storage
	frames       []Frame
	maxFrames    int // Maximum frames to keep in history (0 = unlimited)
	frameCount   int // Total frames received (even if history is limited)
	lastFrame    *Frame
	frameChannel chan Frame // Optional channel for async frame notifications

	// Call tracking
	calls []CallRecord

	// State
	registered   bool
	eventsBound  map[string]bool
	gameName     string
	developer    string
	deviceType   string
	startTime    time.Time
	lastSendTime time.Time

	// Error injection
	registerError      error
	bindError          error
	sendError          error
	sendErrorCount     int // Number of sends to fail (0 = forever if sendError set)
	sendErrorRemaining int // Remaining error count
	heartbeatError     error
	removeError        error

	// Behavior flags
	supportsMultipleEvents bool
	paused                 bool // When true, SendScreenData does nothing
}

// Ensure TestClient implements gamesense.API
var _ gamesense.API = (*TestClient)(nil)

// TestClientOption is a functional option for configuring TestClient
type TestClientOption func(*TestClient)

// WithMaxFrames sets the maximum number of frames to keep in history
func WithMaxFrames(n int) TestClientOption {
	return func(c *TestClient) {
		c.maxFrames = n
	}
}

// WithFrameChannel sets a channel to receive frame notifications
func WithFrameChannel(ch chan Frame) TestClientOption {
	return func(c *TestClient) {
		c.frameChannel = ch
	}
}

// WithDimensions sets the expected frame dimensions
func WithDimensions(width, height int) TestClientOption {
	return func(c *TestClient) {
		c.width = width
		c.height = height
	}
}

// WithMultipleEventsSupport enables/disables multiple events support
func WithMultipleEventsSupport(supported bool) TestClientOption {
	return func(c *TestClient) {
		c.supportsMultipleEvents = supported
	}
}

// NewTestClient creates a new test client with optional configuration
func NewTestClient(opts ...TestClientOption) *TestClient {
	c := &TestClient{
		width:       128,
		height:      40,
		maxFrames:   100, // Default to keeping last 100 frames
		eventsBound: make(map[string]bool),
		startTime:   time.Now(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// RegisterGame implements gamesense.API
func (c *TestClient) RegisterGame(developer string, deinitializeTimerMs int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.recordCall("RegisterGame", developer, deinitializeTimerMs)

	if c.registerError != nil {
		return c.registerError
	}

	c.registered = true
	c.developer = developer
	return nil
}

// BindScreenEvent implements gamesense.API
func (c *TestClient) BindScreenEvent(eventName, deviceType string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.recordCall("BindScreenEvent", eventName, deviceType)

	if c.bindError != nil {
		return c.bindError
	}

	c.eventsBound[eventName] = true
	c.deviceType = deviceType
	return nil
}

// SendScreenData implements gamesense.API
func (c *TestClient) SendScreenData(eventName string, bitmapData []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.recordCall("SendScreenData", eventName, len(bitmapData))

	if c.paused {
		return nil
	}

	if c.sendError != nil {
		if c.sendErrorCount == 0 || c.sendErrorRemaining > 0 {
			if c.sendErrorRemaining > 0 {
				c.sendErrorRemaining--
			}
			return c.sendError
		}
	}

	c.captureFrame(eventName, bitmapData)
	return nil
}

// SendScreenDataMultiRes implements gamesense.API
func (c *TestClient) SendScreenDataMultiRes(eventName string, resolutionData map[string][]byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.recordCall("SendScreenDataMultiRes", eventName, len(resolutionData))

	if c.paused {
		return nil
	}

	if c.sendError != nil {
		if c.sendErrorCount == 0 || c.sendErrorRemaining > 0 {
			if c.sendErrorRemaining > 0 {
				c.sendErrorRemaining--
			}
			return c.sendError
		}
	}

	// Capture the frame for our configured resolution
	key := fmt.Sprintf("image-data-%dx%d", c.width, c.height)
	if data, ok := resolutionData[key]; ok {
		c.captureFrame(eventName, data)
	}

	return nil
}

// SendHeartbeat implements gamesense.API
func (c *TestClient) SendHeartbeat() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.recordCall("SendHeartbeat")

	if c.heartbeatError != nil {
		return c.heartbeatError
	}

	return nil
}

// RemoveGame implements gamesense.API
func (c *TestClient) RemoveGame() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.recordCall("RemoveGame")

	if c.removeError != nil {
		return c.removeError
	}

	c.registered = false
	return nil
}

// SupportsMultipleEvents implements gamesense.API
func (c *TestClient) SupportsMultipleEvents() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.recordCallUnlocked("SupportsMultipleEvents")
	return c.supportsMultipleEvents
}

// SendMultipleScreenData implements gamesense.API
func (c *TestClient) SendMultipleScreenData(eventName string, frames [][]byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.recordCall("SendMultipleScreenData", eventName, len(frames))

	if c.paused {
		return nil
	}

	if c.sendError != nil {
		if c.sendErrorCount == 0 || c.sendErrorRemaining > 0 {
			if c.sendErrorRemaining > 0 {
				c.sendErrorRemaining--
			}
			return c.sendError
		}
	}

	// Capture all frames
	for _, frameData := range frames {
		c.captureFrame(eventName, frameData)
	}

	return nil
}

// captureFrame stores a frame in history (must be called with lock held)
func (c *TestClient) captureFrame(eventName string, data []byte) {
	now := time.Now()
	c.lastSendTime = now
	c.frameCount++

	frame := Frame{
		Data:      make([]byte, len(data)),
		EventName: eventName,
		Timestamp: now,
		Index:     c.frameCount,
	}
	copy(frame.Data, data)

	c.lastFrame = &frame

	// Add to history
	c.frames = append(c.frames, frame)

	// Trim history if needed
	if c.maxFrames > 0 && len(c.frames) > c.maxFrames {
		c.frames = c.frames[len(c.frames)-c.maxFrames:]
	}

	// Notify channel if set (non-blocking)
	if c.frameChannel != nil {
		select {
		case c.frameChannel <- frame:
		default:
		}
	}
}

// recordCall adds a call to the history (must be called with lock held)
func (c *TestClient) recordCall(method string, args ...interface{}) {
	c.calls = append(c.calls, CallRecord{
		Method:    method,
		Args:      args,
		Timestamp: time.Now(),
	})
}

// recordCallUnlocked is for methods that already hold RLock
func (c *TestClient) recordCallUnlocked(_ string, _ ...interface{}) {
	// Note: This is a simplified version that doesn't modify calls
	// to avoid lock upgrade issues. Full tracking requires Lock.
}

// --- Error Injection Methods ---

// SetRegisterError sets an error to return from RegisterGame
func (c *TestClient) SetRegisterError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.registerError = err
}

// SetBindError sets an error to return from BindScreenEvent
func (c *TestClient) SetBindError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bindError = err
}

// SetSendError sets an error to return from send methods
// count specifies how many sends should fail (0 = all sends fail)
func (c *TestClient) SetSendError(err error, count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sendError = err
	c.sendErrorCount = count
	c.sendErrorRemaining = count
}

// SetHeartbeatError sets an error to return from SendHeartbeat
func (c *TestClient) SetHeartbeatError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.heartbeatError = err
}

// SetRemoveError sets an error to return from RemoveGame
func (c *TestClient) SetRemoveError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeError = err
}

// ClearErrors removes all injected errors
func (c *TestClient) ClearErrors() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clearErrorsLocked()
}

// clearErrorsLocked removes all injected errors (caller must hold lock)
func (c *TestClient) clearErrorsLocked() {
	c.registerError = nil
	c.bindError = nil
	c.sendError = nil
	c.sendErrorCount = 0
	c.sendErrorRemaining = 0
	c.heartbeatError = nil
	c.removeError = nil
}

// --- Control Methods ---

// Pause stops capturing frames (sends succeed but aren't stored)
func (c *TestClient) Pause() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = true
}

// Resume resumes frame capture
func (c *TestClient) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = false
}

// Reset clears all state and history
func (c *TestClient) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.frames = nil
	c.frameCount = 0
	c.lastFrame = nil
	c.calls = nil
	c.registered = false
	c.eventsBound = make(map[string]bool)
	c.gameName = ""
	c.developer = ""
	c.deviceType = ""
	c.startTime = time.Now()
	c.lastSendTime = time.Time{}
	c.paused = false

	c.clearErrorsLocked()
}

// --- Query Methods ---

// FrameCount returns the total number of frames received
func (c *TestClient) FrameCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.frameCount
}

// Frames returns a copy of the frame history
func (c *TestClient) Frames() []Frame {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]Frame, len(c.frames))
	copy(result, c.frames)
	return result
}

// LastFrame returns the most recent frame, or nil if none
func (c *TestClient) LastFrame() *Frame {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastFrame == nil {
		return nil
	}

	frame := *c.lastFrame
	frame.Data = make([]byte, len(c.lastFrame.Data))
	copy(frame.Data, c.lastFrame.Data)
	return &frame
}

// Frame returns a specific frame by index (1-based), or nil if not in history
func (c *TestClient) Frame(index int) *Frame {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, f := range c.frames {
		if f.Index == index {
			frame := f
			frame.Data = make([]byte, len(f.Data))
			copy(frame.Data, f.Data)
			return &frame
		}
	}
	return nil
}

// Calls returns a copy of the call history
func (c *TestClient) Calls() []CallRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]CallRecord, len(c.calls))
	copy(result, c.calls)
	return result
}

// CallCount returns the number of calls to a specific method
func (c *TestClient) CallCount(method string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, call := range c.calls {
		if call.Method == method {
			count++
		}
	}
	return count
}

// IsRegistered returns whether RegisterGame was called successfully
func (c *TestClient) IsRegistered() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.registered
}

// IsBound returns whether a specific event has been bound
func (c *TestClient) IsBound(eventName string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.eventsBound[eventName]
}

// LastSendTime returns when the last frame was sent
func (c *TestClient) LastSendTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastSendTime
}

// Duration returns how long since the client was created or reset
func (c *TestClient) Duration() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.startTime)
}
