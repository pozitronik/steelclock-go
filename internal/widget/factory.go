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

// CreateWidgets creates all enabled widgets from configuration
// If a widget fails to initialize (e.g., volume widget on system without sound),
// it logs a warning and skips that widget, continuing with the remaining widgets.
// Only returns error if NO widgets could be created.
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
			// Log warning and skip widget instead of failing completely
			log.Printf("WARNING: Failed to create widget '%s' (type: %s): %v", cfg.ID, cfg.Type, err)
			log.Printf("         Skipping widget '%s' - application will continue with remaining widgets", cfg.ID)
			failures = append(failures, fmt.Sprintf("%s (%s): %v", cfg.ID, cfg.Type, err))
			continue // Skip this widget but continue with others
		}

		widgets = append(widgets, w)
	}

	// Only fail if there were enabled widgets but ALL failed to initialize
	if len(widgets) == 0 && enabledCount > 0 {
		return nil, fmt.Errorf("no widgets could be created - all %d enabled widget(s) failed to initialize", enabledCount)
	}

	// Log summary if some widgets failed
	if len(failures) > 0 {
		log.Printf("Application started with %d/%d widgets (%d skipped due to initialization errors)",
			len(widgets), enabledCount, len(failures))
		log.Printf("Skipped widgets:")
		for _, failure := range failures {
			log.Printf("  - %s", failure)
		}
	}

	return widgets, nil
}
