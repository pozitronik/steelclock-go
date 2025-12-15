package app

import (
	"github.com/pozitronik/steelclock-go/internal/backend"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/display"

	// Import backend implementations for self-registration via init()
	_ "github.com/pozitronik/steelclock-go/internal/backend/direct"
	_ "github.com/pozitronik/steelclock-go/internal/backend/gamesense"
	_ "github.com/pozitronik/steelclock-go/internal/backend/webclient"
)

// CreateBackendClient creates the appropriate client based on backend configuration.
// Returns BackendUnavailableError if no backend can be created.
func CreateBackendClient(cfg *config.Config) (display.Backend, string, error) {
	result, err := backend.Create(cfg)
	if err != nil {
		return nil, "", &BackendUnavailableError{Err: err}
	}
	return result.Backend, result.Name, nil
}

// CreateBackendByName creates a specific backend by name.
// Returns BackendUnavailableError if the backend cannot be created.
func CreateBackendByName(name string, cfg *config.Config) (display.Backend, error) {
	result, err := backend.CreateByName(name, cfg)
	if err != nil {
		return nil, &BackendUnavailableError{Err: err}
	}
	return result.Backend, nil
}

// CreateBackendExcluding creates a backend using auto-selection, excluding specified backends.
// Used for failover when current backend fails.
// Returns BackendUnavailableError if no backend can be created.
func CreateBackendExcluding(cfg *config.Config, exclude ...string) (display.Backend, string, error) {
	result, err := backend.CreateExcluding(cfg, exclude...)
	if err != nil {
		return nil, "", &BackendUnavailableError{Err: err}
	}
	return result.Backend, result.Name, nil
}
