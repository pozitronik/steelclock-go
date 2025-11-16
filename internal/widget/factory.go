package widget

import (
	"fmt"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// CreateWidget creates a widget from configuration
func CreateWidget(cfg config.WidgetConfig) (Widget, error) {
	if !cfg.Enabled {
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
	case "volume":
		return NewVolumeWidget(cfg)
	default:
		return nil, fmt.Errorf("unknown widget type: %s", cfg.Type)
	}
}

// CreateWidgets creates all enabled widgets from configuration
func CreateWidgets(cfgs []config.WidgetConfig) ([]Widget, error) {
	var widgets []Widget

	for _, cfg := range cfgs {
		if !cfg.Enabled {
			continue
		}

		w, err := CreateWidget(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create widget %s: %w", cfg.ID, err)
		}

		widgets = append(widgets, w)
	}

	return widgets, nil
}
