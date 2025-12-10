// Package backend provides a registry for display backend implementations.
// Backends self-register via init() functions, similar to the widget pattern.
package backend

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/display"
)

// Factory creates a backend from configuration
type Factory func(cfg *config.Config) (display.Backend, error)

// registration holds a factory and its priority for auto-selection
type registration struct {
	factory  Factory
	priority int // Lower = higher priority (tried first in auto-selection)
}

var (
	registry   = make(map[string]registration)
	registryMu sync.RWMutex
)

func init() {
	// Set up callbacks in config package to use registry as single source of truth
	config.BackendTypeChecker = IsRegistered
	config.BackendTypesLister = RegisteredTypesList
}

// Register registers a backend factory with the given name and priority.
// Lower priority values are tried first during auto-selection.
// This should be called from init() functions in backend implementation packages.
func Register(name string, factory Factory, priority int) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		log.Printf("WARNING: Backend '%s' is being re-registered", name)
	}
	registry[name] = registration{factory: factory, priority: priority}
}

// IsRegistered checks if a backend type is registered
func IsRegistered(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, exists := registry[name]
	return exists
}

// RegisteredTypes returns a sorted list of all registered backend names
func RegisteredTypes() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	types := make([]string, 0, len(registry))
	for name := range registry {
		types = append(types, name)
	}
	sort.Strings(types)
	return types
}

// RegisteredTypesList returns a comma-separated string of registered backend types
func RegisteredTypesList() string {
	return strings.Join(RegisteredTypes(), ", ")
}

// Result holds the result of backend creation
type Result struct {
	Backend display.Backend
	Name    string // Which backend was created
}

// Create creates a backend based on configuration.
// If cfg.Backend is empty, tries all registered backends by priority.
// Otherwise, creates the specified backend.
func Create(cfg *config.Config) (Result, error) {
	if cfg.Backend == "" {
		return createAuto(cfg)
	}
	return CreateByName(cfg.Backend, cfg)
}

// CreateByName creates a specific backend by name
func CreateByName(name string, cfg *config.Config) (Result, error) {
	registryMu.RLock()
	reg, ok := registry[name]
	registryMu.RUnlock()

	if !ok {
		return Result{}, fmt.Errorf("unknown backend: %s (available: %s)", name, RegisteredTypesList())
	}

	backend, err := reg.factory(cfg)
	if err != nil {
		return Result{}, err
	}

	return Result{Backend: backend, Name: name}, nil
}

// CreateExcluding creates a backend using auto-selection, excluding specified backends.
// Used for failover when current backend fails.
func CreateExcluding(cfg *config.Config, exclude ...string) (Result, error) {
	excludeSet := make(map[string]bool, len(exclude))
	for _, name := range exclude {
		excludeSet[name] = true
	}
	return createAutoWithExclude(cfg, excludeSet)
}

// createAuto tries all registered backends in priority order until one succeeds
func createAuto(cfg *config.Config) (Result, error) {
	return createAutoWithExclude(cfg, nil)
}

// createAutoWithExclude tries backends in priority order, optionally excluding some
func createAutoWithExclude(cfg *config.Config, exclude map[string]bool) (Result, error) {
	registryMu.RLock()
	// Build sorted list by priority
	type entry struct {
		name string
		reg  registration
	}
	entries := make([]entry, 0, len(registry))
	for name, reg := range registry {
		if exclude != nil && exclude[name] {
			continue // Skip excluded backends
		}
		entries = append(entries, entry{name: name, reg: reg})
	}
	registryMu.RUnlock()

	// Sort by priority (lower first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].reg.priority < entries[j].reg.priority
	})

	var lastErr error
	for _, e := range entries {
		log.Printf("Trying backend '%s'...", e.name)
		backend, err := e.reg.factory(cfg)
		if err == nil {
			log.Printf("Backend '%s' connected successfully", e.name)
			return Result{Backend: backend, Name: e.name}, nil
		}
		log.Printf("Backend '%s' failed: %v", e.name, err)
		lastErr = err
	}

	if lastErr != nil {
		return Result{}, fmt.Errorf("all backends failed, last error: %w", lastErr)
	}
	return Result{}, fmt.Errorf("no backends registered")
}

// SnapshotRegistry returns a copy of the current registry (for tests)
func SnapshotRegistry() map[string]registration {
	registryMu.RLock()
	defer registryMu.RUnlock()
	snapshot := make(map[string]registration, len(registry))
	for k, v := range registry {
		snapshot[k] = v
	}
	return snapshot
}

// RestoreRegistry replaces the registry with a snapshot (for tests)
func RestoreRegistry(snapshot map[string]registration) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = snapshot
}
