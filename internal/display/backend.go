// Package display provides abstractions for display backends.
// This package defines interfaces that decouple consumers (compositor)
// from specific implementations (GameSense API, direct HID driver).
package display

// FrameSender handles frame transmission to the display.
type FrameSender interface {
	SendScreenData(eventName string, bitmapData []byte) error
	SendScreenDataMultiRes(eventName string, resolutionData map[string][]byte) error
	SendMultipleScreenData(eventName string, frames [][]byte) error
}

// HeartbeatSender handles keep-alive messages to the backend.
type HeartbeatSender interface {
	SendHeartbeat() error
}

// BatchCapability provides information about batching support.
type BatchCapability interface {
	SupportsMultipleEvents() bool
}

// GameRegistrar handles game/application registration lifecycle.
type GameRegistrar interface {
	RegisterGame(developer string, deinitializeTimerMs int) error
	BindScreenEvent(eventName, deviceType string) error
	RemoveGame() error
}

// Client combines frame sending and heartbeat capabilities.
// Used by components that render and maintain connection (e.g., Compositor).
type Client interface {
	FrameSender
	HeartbeatSender
	BatchCapability
}

// Backend defines the full interface for display backends.
// Combines all capabilities for components that need full control.
type Backend interface {
	Client
	GameRegistrar
}
