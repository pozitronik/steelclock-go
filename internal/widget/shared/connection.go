package shared

import (
	"context"
	"sync"
	"time"
)

// Connectable defines the interface for something that can connect
type Connectable interface {
	Connect(ctx context.Context) error
	IsConnected() bool
}

// ConnectionManager handles connection lifecycle with automatic retry
type ConnectionManager struct {
	mu                sync.RWMutex
	connectable       Connectable
	connecting        bool
	lastConnectionTry time.Time
	reconnectInterval time.Duration
	connectionTimeout time.Duration
	connectionError   error
	onError           func(error)
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(connectable Connectable, reconnectInterval, connectionTimeout time.Duration) *ConnectionManager {
	return &ConnectionManager{
		connectable:       connectable,
		reconnectInterval: reconnectInterval,
		connectionTimeout: connectionTimeout,
	}
}

// SetErrorCallback sets the callback for connection errors
func (c *ConnectionManager) SetErrorCallback(fn func(error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onError = fn
}

// Update checks connection state and initiates reconnection if needed
// Must be called with external lock held if widget has its own mutex
// Returns true if currently connecting
func (c *ConnectionManager) Update() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connectable.IsConnected() {
		return false
	}

	if c.connecting {
		return true
	}

	if time.Since(c.lastConnectionTry) <= c.reconnectInterval {
		return false
	}

	// Start connection attempt
	c.connecting = true
	c.lastConnectionTry = time.Now()
	c.connectionError = nil

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.connectionTimeout)
		defer cancel()

		err := c.connectable.Connect(ctx)

		c.mu.Lock()
		c.connecting = false
		if err != nil {
			c.connectionError = err
			if c.onError != nil {
				c.onError(err)
			}
		}
		c.mu.Unlock()
	}()

	return true
}

// IsConnecting returns true if a connection attempt is in progress
func (c *ConnectionManager) IsConnecting() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connecting
}

// IsConnected returns true if the underlying connectable is connected
func (c *ConnectionManager) IsConnected() bool {
	return c.connectable.IsConnected()
}

// GetError returns the last connection error
func (c *ConnectionManager) GetError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connectionError
}

// ClearError clears the connection error
func (c *ConnectionManager) ClearError() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connectionError = nil
}

// ResetConnectionTimer resets the reconnection timer to allow immediate retry
func (c *ConnectionManager) ResetConnectionTimer() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastConnectionTry = time.Time{}
}
