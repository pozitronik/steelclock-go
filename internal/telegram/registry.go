// Package telegram provides a Telegram client wrapper for receiving notifications
package telegram

import (
	"fmt"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// clientRegistry manages shared Telegram client instances
type clientRegistry struct {
	mu      sync.RWMutex
	clients map[string]*Client // key: "api_id:phone_number"
	refs    map[string]int     // reference count per client
}

// global registry instance
var (
	registry     *clientRegistry
	registryOnce sync.Once
)

// getRegistry returns the singleton client registry
func getRegistry() *clientRegistry {
	registryOnce.Do(func() {
		registry = &clientRegistry{
			clients: make(map[string]*Client),
			refs:    make(map[string]int),
		}
	})
	return registry
}

// clientKey generates a unique key for a client configuration
func clientKey(cfg *config.TelegramConfig) string {
	if cfg == nil || cfg.Auth == nil {
		return ""
	}
	return fmt.Sprintf("%d:%s", cfg.Auth.APIID, cfg.Auth.PhoneNumber)
}

// GetOrCreateClient returns an existing client or creates a new one for the given config.
// The client is reference-counted; call ReleaseClient when done.
func GetOrCreateClient(cfg *config.TelegramConfig) (*Client, error) {
	reg := getRegistry()
	key := clientKey(cfg)
	if key == "" {
		return nil, fmt.Errorf("invalid telegram configuration")
	}

	reg.mu.Lock()
	defer reg.mu.Unlock()

	// Check if client already exists
	if client, ok := reg.clients[key]; ok {
		reg.refs[key]++
		return client, nil
	}

	// Create new client
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	reg.clients[key] = client
	reg.refs[key] = 1

	return client, nil
}

// ReleaseClient decrements the reference count for a client.
// When the count reaches zero, the client is disconnected and removed.
func ReleaseClient(cfg *config.TelegramConfig) {
	reg := getRegistry()
	key := clientKey(cfg)
	if key == "" {
		return
	}

	reg.mu.Lock()
	defer reg.mu.Unlock()

	if count, ok := reg.refs[key]; ok {
		count--
		if count <= 0 {
			// No more references - disconnect and remove
			if client, ok := reg.clients[key]; ok {
				client.Disconnect()
				delete(reg.clients, key)
			}
			delete(reg.refs, key)
		} else {
			reg.refs[key] = count
		}
	}
}

// GetClientRefCount returns the current reference count for a client (for testing/debugging)
func GetClientRefCount(cfg *config.TelegramConfig) int {
	reg := getRegistry()
	key := clientKey(cfg)
	if key == "" {
		return 0
	}

	reg.mu.RLock()
	defer reg.mu.RUnlock()

	return reg.refs[key]
}
