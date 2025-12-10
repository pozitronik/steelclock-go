package widget

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// Factory is a function that creates a widget from configuration
type Factory func(cfg config.WidgetConfig) (Widget, error)

// registry holds all registered widget factories
var registry = make(map[string]Factory)

func init() {
	// Set up callbacks in config package to use factory registry as single source of truth.
	// This avoids maintaining duplicate widget type lists and prevents import cycles.
	config.WidgetTypeChecker = IsRegistered
	config.WidgetTypesLister = RegisteredTypesList
}

// IsRegistered checks if a widget type is registered in the factory
func IsRegistered(typeName string) bool {
	_, exists := registry[typeName]
	return exists
}

// Register registers a widget factory for the given type name.
// This should be called from init() functions in widget implementation files.
func Register(typeName string, factory Factory) {
	if _, exists := registry[typeName]; exists {
		log.Printf("WARNING: Widget type '%s' is being re-registered", typeName)
	}
	registry[typeName] = factory
}

// RegisteredTypes returns a sorted list of all registered widget type names
func RegisteredTypes() []string {
	types := make([]string, 0, len(registry))
	for typeName := range registry {
		types = append(types, typeName)
	}
	sort.Strings(types)
	return types
}

// RegisteredTypesList returns a comma-separated string of registered widget types
func RegisteredTypesList() string {
	return strings.Join(RegisteredTypes(), ", ")
}

// CreateWidget creates a widget from configuration
func CreateWidget(cfg config.WidgetConfig) (Widget, error) {
	if !cfg.IsEnabled() {
		return nil, fmt.Errorf("widget %s is disabled", cfg.ID)
	}

	factory, ok := registry[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unknown widget type: %s (valid: %s)", cfg.Type, RegisteredTypesList())
	}

	return factory(cfg)
}

// CreateWidgets creates all enabled widgets from configuration.
// If a widget fails to initialize (e.g., invalid configuration), an ErrorWidget
// is created in its place to display the error on screen.
// This allows users to see which widgets have configuration problems.
func CreateWidgets(cfgs []config.WidgetConfig) ([]Widget, error) {
	var widgets []Widget
	var failures []string
	enabledCount := 0

	for _, cfg := range cfgs {
		if !cfg.IsEnabled() {
			continue
		}

		enabledCount++

		w, err := CreateWidget(cfg)
		if err != nil {
			// Log warning and create error proxy widget
			log.Printf("WARNING: Failed to create widget '%s' (type: %s): %v", cfg.ID, cfg.Type, err)

			// Create an ErrorWidget proxy to display the error on screen
			errorMsg := abbreviateError(err.Error())
			errorWidget := NewErrorWidgetWithConfig(cfg, errorMsg)
			widgets = append(widgets, errorWidget)

			failures = append(failures, fmt.Sprintf("%s (%s): %v", cfg.ID, cfg.Type, err))
			continue
		}

		widgets = append(widgets, w)
	}

	// Only fail if there were enabled widgets but ALL failed to initialize,
	// and we couldn't even create error proxies (shouldn't happen)
	if len(widgets) == 0 && enabledCount > 0 {
		return nil, fmt.Errorf("no widgets could be created - all %d enabled widget(s) failed to initialize", enabledCount)
	}

	// Log summary if some widgets failed
	if len(failures) > 0 {
		log.Printf("Application started with %d widgets (%d showing errors)",
			len(widgets), len(failures))
		log.Printf("Failed widgets:")
		for _, failure := range failures {
			log.Printf("  - %s", failure)
		}
	}

	return widgets, nil
}

// abbreviateError shortens error messages for display on small screens.
// It extracts the key part of the error message.
func abbreviateError(errMsg string) string {
	// Map of common error patterns to short display messages
	abbreviations := map[string]string{
		"api_key is required":  "NO API KEY",
		"lat/lon coordinates":  "NO COORDS",
		"location is required": "NO LOCATION",
		"unknown widget type":  "BAD TYPE",
		"failed to load font":  "FONT ERROR",
		"failed to parse":      "PARSE ERROR",
		"timeout":              "TIMEOUT",
		"connection refused":   "NO CONNECT",
		"permission denied":    "NO ACCESS",
	}

	for pattern, abbrev := range abbreviations {
		if strings.Contains(strings.ToLower(errMsg), strings.ToLower(pattern)) {
			return abbrev
		}
	}

	// Fallback: truncate to fit small screens (max ~12 chars for readability)
	if len(errMsg) > 12 {
		return "ERROR"
	}
	return strings.ToUpper(errMsg)
}
