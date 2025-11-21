package widget

import (
	"fmt"
	"log"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// CreateWidget creates a widget from configuration
func CreateWidget(cfg config.WidgetConfig) (Widget, error) {
	if !cfg.IsEnabled() {
		return nil, fmt.Errorf("widget %s is disabled", cfg.ID)
	}

	switch cfg.Type {
	case "clock":
		return NewClockWidget(cfg)
	case "cpu":
		return NewCPUWidget(cfg)
	case "memory":
		return NewMemoryWidget(cfg)
	case "network":
		return NewNetworkWidget(cfg)
	case "disk":
		return NewDiskWidget(cfg)
	case "keyboard":
		return NewKeyboardWidget(cfg)
	case "keyboard_layout":
		return NewKeyboardLayoutWidget(cfg)
	case "volume":
		return NewVolumeWidget(cfg)
	case "volume_meter":
		return NewVolumeMeterWidget(cfg)
	case "doom":
		return NewDoomWidget(cfg)
	default:
		return nil, fmt.Errorf("unknown widget type: %s", cfg.Type)
	}
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
