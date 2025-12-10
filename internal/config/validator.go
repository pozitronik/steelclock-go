package config

import (
	"fmt"
)

// Validation constants
const (
	MinDeinitializeTimerMs = 1000
	MaxDeinitializeTimerMs = 60000
	MinEventBatchSize      = 1
	MaxEventBatchSize      = 100
)

// BackendTypeChecker is a callback function that checks if a backend type is registered.
// This is set by the backend package to avoid import cycles.
var BackendTypeChecker func(name string) bool

// BackendTypesLister is a callback function that returns a list of registered backend types.
// This is set by the backend package to avoid import cycles.
var BackendTypesLister func() string

// IsValidBackend checks if the given backend name is valid.
// Empty string means auto-selection (try backends by priority).
// Other values are checked against the backend registry.
func IsValidBackend(name string) bool {
	if name == "" {
		return true
	}
	if BackendTypeChecker != nil {
		return BackendTypeChecker(name)
	}
	// Fallback for tests that don't import backend packages
	return false
}

// GetValidBackendsList returns a comma-separated list of valid backend types.
func GetValidBackendsList() string {
	if BackendTypesLister != nil {
		return BackendTypesLister()
	}
	return "(no backends registered)"
}

// WidgetTypeChecker is a callback function that checks if a widget type is registered.
// This is set by the widget package to avoid import cycles.
// When set, it delegates to the widget factory's registry (single source of truth).
var WidgetTypeChecker func(typeName string) bool

// WidgetTypesLister is a callback function that returns a list of registered widget types.
// This is set by the widget package to avoid import cycles.
var WidgetTypesLister func() string

// IsValidWidgetType checks if the given type name is a valid widget type.
// It delegates to the widget factory's registry if available.
func IsValidWidgetType(typeName string) bool {
	if WidgetTypeChecker != nil {
		return WidgetTypeChecker(typeName)
	}
	// Fallback should not happen in production (widget package sets the checker),
	// but provides safety for tests that don't import widget packages.
	return false
}

// GetValidWidgetTypesList returns a sorted comma-separated list of valid widget types.
// It delegates to the widget factory's registry if available.
func GetValidWidgetTypesList() string {
	if WidgetTypesLister != nil {
		return WidgetTypesLister()
	}
	return "(no widget types registered)"
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
	if !IsValidBackend(cfg.Backend) {
		return fmt.Errorf("invalid backend '%s' (valid: %s)", cfg.Backend, GetValidBackendsList())
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
