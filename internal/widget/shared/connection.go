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
		if err != nil {
			c.mu.Lock()
			c.connecting = false
			c.connectionError = err
			if c.onError != nil {
				c.onError(err)
			}
			c.mu.Unlock()
			return
		}

		// Wait for full connection (including authentication for some clients)
		// Connect() may return when TCP is established, but IsConnected()
		// may require additional steps (e.g., authentication)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				c.mu.Lock()
				c.connecting = false
				c.connectionError = ctx.Err()
				if c.onError != nil {
					c.onError(ctx.Err())
				}
				c.mu.Unlock()
				return
			case <-ticker.C:
				if c.connectable.IsConnected() {
					c.mu.Lock()
					c.connecting = false
					c.mu.Unlock()
					return
				}
			}
		}
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

// IsInitialState returns true if no connection attempt has been made yet
// Use this to show "Connecting..." instead of "Disconnected" on startup
func (c *ConnectionManager) IsInitialState() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastConnectionTry.IsZero()
}
