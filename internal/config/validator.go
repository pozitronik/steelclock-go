package config

import (
	"fmt"
	"sort"
	"strings"
)

// Validation constants
const (
	MinDeinitializeTimerMs = 1000
	MaxDeinitializeTimerMs = 60000
	MinEventBatchSize      = 1
	MaxEventBatchSize      = 100
)

// ValidBackends contains valid backend values
var ValidBackends = map[string]bool{
	"":          true, // Empty = default (gamesense)
	"gamesense": true,
	"direct":    true,
	"any":       true,
}

// ValidWidgetTypes contains all valid widget type names.
// This is the single source of truth for widget types.
var ValidWidgetTypes = map[string]bool{
	"clock":            true,
	"cpu":              true,
	"memory":           true,
	"network":          true,
	"disk":             true,
	"keyboard":         true,
	"keyboard_layout":  true,
	"volume":           true,
	"volume_meter":     true,
	"audio_visualizer": true,
	"doom":             true,
	"winamp":           true,
	"matrix":           true,
	"weather":          true,
	"battery":          true,
	"game_of_life":     true,
	"hyperspace":       true,
	"starwars_intro":   true,
	"telegram":         true,
	"telegram_unread":  true,
}

// IsValidWidgetType checks if the given type name is a valid widget type
func IsValidWidgetType(typeName string) bool {
	return ValidWidgetTypes[typeName]
}

// GetValidWidgetTypesList returns a sorted comma-separated list of valid widget types
func GetValidWidgetTypesList() string {
	types := make([]string, 0, len(ValidWidgetTypes))
	for t := range ValidWidgetTypes {
		types = append(types, t)
	}
	sort.Strings(types)
	return strings.Join(types, ", ")
}

// Validate checks that the configuration is valid
func Validate(cfg *Config) error {
	if err := validateGlobalConfig(cfg); err != nil {
		return err
	}

	if err := validateDisplayConfig(cfg); err != nil {
		return err
	}

	if err := validateWidgets(cfg); err != nil {
		return err
	}

	return nil
}

// validateGlobalConfig validates global configuration settings
func validateGlobalConfig(cfg *Config) error {
	if !ValidBackends[cfg.Backend] {
		return fmt.Errorf("invalid backend '%s' (valid: gamesense, direct, any)", cfg.Backend)
	}

	if cfg.DeinitializeTimerMs != 0 {
		if cfg.DeinitializeTimerMs < MinDeinitializeTimerMs || cfg.DeinitializeTimerMs > MaxDeinitializeTimerMs {
			return fmt.Errorf("deinitialize_timer_ms must be between %d and %d (got %d)",
				MinDeinitializeTimerMs, MaxDeinitializeTimerMs, cfg.DeinitializeTimerMs)
		}
	}

	if cfg.EventBatchSize != 0 {
		if cfg.EventBatchSize < MinEventBatchSize || cfg.EventBatchSize > MaxEventBatchSize {
			return fmt.Errorf("event_batch_size must be between %d and %d (got %d)",
				MinEventBatchSize, MaxEventBatchSize, cfg.EventBatchSize)
		}
	}

	return nil
}

// validateDisplayConfig validates display configuration settings
func validateDisplayConfig(cfg *Config) error {
	if cfg.Display.Width <= 0 {
		return fmt.Errorf("display width must be positive (got %d)", cfg.Display.Width)
	}

	if cfg.Display.Height <= 0 {
		return fmt.Errorf("display height must be positive (got %d)", cfg.Display.Height)
	}

	if cfg.RefreshRateMs <= 0 {
		return fmt.Errorf("refresh_rate_ms must be positive (got %d)", cfg.RefreshRateMs)
	}

	for i, res := range cfg.SupportedResolutions {
		if res.Width <= 0 {
			return fmt.Errorf("supported_resolutions[%d]: width must be positive (got %d)", i, res.Width)
		}
		if res.Height <= 0 {
			return fmt.Errorf("supported_resolutions[%d]: height must be positive (got %d)", i, res.Height)
		}
	}

	return nil
}

// validateWidgets validates all widget configurations
func validateWidgets(cfg *Config) error {
	if len(cfg.Widgets) == 0 {
		return fmt.Errorf("at least one widget must be configured")
	}

	// Generate IDs for widgets
	generateWidgetIDs(cfg.Widgets)

	for i := range cfg.Widgets {
		w := &cfg.Widgets[i]

		if err := validateWidgetType(i, w); err != nil {
			return err
		}

		if w.IsEnabled() {
			if err := validateWidgetProperties(i, w); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateWidgetType validates the widget type
func validateWidgetType(index int, w *WidgetConfig) error {
	if w.Type == "" {
		return fmt.Errorf("widget[%d]: type is required", index)
	}

	if !IsValidWidgetType(w.Type) {
		return fmt.Errorf("widget[%d]: invalid type '%s' (valid: %s)", index, w.Type, GetValidWidgetTypesList())
	}

	return nil
}

// validateWidgetProperties validates type-specific widget properties
func validateWidgetProperties(_ int, _ *WidgetConfig) error {
	// Network and disk widgets support auto-detection when interface/disk is omitted
	// (sums all interfaces/disks), so no validation required
	return nil
}
