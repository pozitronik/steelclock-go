package widget

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// TestBaseWidget_GetUpdateInterval tests update interval getter
func TestBaseWidget_GetUpdateInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval float64
		expected time.Duration
	}{
		{
			name:     "1 second interval",
			interval: 1.0,
			expected: 1 * time.Second,
		},
		{
			name:     "500ms interval",
			interval: 0.5,
			expected: 500 * time.Millisecond,
		},
		{
			name:     "100ms interval",
			interval: 0.1,
			expected: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := NewBaseWidget(config.WidgetConfig{
				ID:             "test",
				UpdateInterval: tt.interval,
			})

			result := base.GetUpdateInterval()
			if result != tt.expected {
				t.Errorf("GetUpdateInterval() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestBaseWidget_GetPosition tests position getter
func TestBaseWidget_GetPosition(t *testing.T) {
	pos := config.PositionConfig{
		X: 10,
		Y: 20,
		W: 128,
		H: 40,
		Z: 5,
	}

	base := NewBaseWidget(config.WidgetConfig{
		ID:       "test",
		Position: pos,
	})

	result := base.GetPosition()
	if result.X != pos.X || result.Y != pos.Y || result.W != pos.W || result.H != pos.H || result.Z != pos.Z {
		t.Errorf("GetPosition() = %+v, want %+v", result, pos)
	}
}

// TestBaseWidget_GetAutoHideTimeout tests auto-hide timeout getter
func TestBaseWidget_GetAutoHideTimeout(t *testing.T) {
	tests := []struct {
		name     string
		timeout  float64
		expected time.Duration
	}{
		{
			name:     "2 second timeout",
			timeout:  2.0,
			expected: 2 * time.Second,
		},
		{
			name:     "500ms timeout",
			timeout:  0.5,
			expected: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := NewBaseWidget(config.WidgetConfig{
				ID: "test",
				AutoHide: &config.AutoHideConfig{
					Enabled: true,
					Timeout: tt.timeout,
				},
			})

			result := base.GetAutoHideTimeout()
			if result != tt.expected {
				t.Errorf("GetAutoHideTimeout() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestBaseWidget_AutoHide tests auto-hide functionality
func TestBaseWidget_AutoHide(t *testing.T) {
	t.Run("disabled auto-hide", func(t *testing.T) {
		base := NewBaseWidget(config.WidgetConfig{
			ID: "test",
			AutoHide: &config.AutoHideConfig{
				Enabled: false,
			},
		})

		// Should never hide when auto-hide is disabled
		if base.ShouldHide() {
			t.Error("ShouldHide() should return false when auto-hide is disabled")
		}

		// Trigger should do nothing
		base.TriggerAutoHide()

		if base.ShouldHide() {
			t.Error("ShouldHide() should still return false after TriggerAutoHide() when auto-hide is disabled")
		}
	})

	t.Run("enabled auto-hide - initial state", func(t *testing.T) {
		base := NewBaseWidget(config.WidgetConfig{
			ID: "test",
			AutoHide: &config.AutoHideConfig{
				Enabled: true,
				Timeout: 0.1, // 100ms
			},
		})

		// Should be hidden initially (never triggered)
		if !base.ShouldHide() {
			t.Error("ShouldHide() should return true initially when auto-hide is enabled and never triggered")
		}
	})

	t.Run("enabled auto-hide - after trigger", func(t *testing.T) {
		base := NewBaseWidget(config.WidgetConfig{
			ID: "test",
			AutoHide: &config.AutoHideConfig{
				Enabled: true,
				Timeout: 0.2, // 200ms
			},
		})

		// Trigger auto-hide
		base.TriggerAutoHide()

		// Should be visible immediately after trigger
		if base.ShouldHide() {
			t.Error("ShouldHide() should return false immediately after TriggerAutoHide()")
		}

		// Wait for timeout to expire
		time.Sleep(250 * time.Millisecond)

		// Should be hidden after timeout
		if !base.ShouldHide() {
			t.Error("ShouldHide() should return true after timeout expires")
		}
	})

	t.Run("enabled auto-hide - retrigger resets timer", func(t *testing.T) {
		base := NewBaseWidget(config.WidgetConfig{
			ID: "test",
			AutoHide: &config.AutoHideConfig{
				Enabled: true,
				Timeout: 0.3, // 300ms
			},
		})

		// First trigger
		base.TriggerAutoHide()

		// Wait 150ms (half the timeout)
		time.Sleep(150 * time.Millisecond)

		// Should still be visible
		if base.ShouldHide() {
			t.Error("ShouldHide() should return false before timeout expires")
		}

		// Trigger again (resets timer)
		base.TriggerAutoHide()

		// Wait another 150ms (total 300ms from first trigger, but only 150ms from second)
		time.Sleep(150 * time.Millisecond)

		// Should still be visible because timer was reset
		if base.ShouldHide() {
			t.Error("ShouldHide() should return false when timer is reset by re-trigger")
		}

		// Wait for full timeout from second trigger
		time.Sleep(200 * time.Millisecond)

		// Now should be hidden
		if !base.ShouldHide() {
			t.Error("ShouldHide() should return true after timeout from last trigger")
		}
	})
}

// TestBaseWidget_Name tests name getter
func TestBaseWidget_Name(t *testing.T) {
	tests := []struct {
		name     string
		widgetID string
	}{
		{"simple name", "test_widget"},
		{"with underscores", "my_test_widget_123"},
		{"with dashes", "test-widget-name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := NewBaseWidget(config.WidgetConfig{
				ID: tt.widgetID,
			})

			result := base.Name()
			if result != tt.widgetID {
				t.Errorf("Name() = %v, want %v", result, tt.widgetID)
			}
		})
	}
}

// TestBaseWidget_IsAutoHideEnabled tests auto-hide enabled flag
func TestBaseWidget_IsAutoHideEnabled(t *testing.T) {
	tests := []struct {
		name     string
		autoHide bool
	}{
		{"auto-hide enabled", true},
		{"auto-hide disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := NewBaseWidget(config.WidgetConfig{
				ID: "test",
				AutoHide: &config.AutoHideConfig{
					Enabled: tt.autoHide,
				},
			})

			result := base.IsAutoHideEnabled()
			if result != tt.autoHide {
				t.Errorf("IsAutoHideEnabled() = %v, want %v", result, tt.autoHide)
			}
		})
	}
}

// TestBaseWidget_GetRenderBackgroundColor tests background color getter
func TestBaseWidget_GetRenderBackgroundColor(t *testing.T) {
	tests := []struct {
		name     string
		bgColor  int
		expected uint8
	}{
		{"black background", 0, 0},
		{"white background", 255, 255},
		{"mid gray", 128, 128},
		{"transparent (-1)", -1, 0}, // Transparent becomes black
		{"dark gray", 64, 64},
		{"light gray", 192, 192},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := NewBaseWidget(config.WidgetConfig{
				ID: "test",
				Style: &config.StyleConfig{
					Background: tt.bgColor,
				},
			})

			result := base.GetRenderBackgroundColor()
			if result != tt.expected {
				t.Errorf("GetRenderBackgroundColor() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestBaseWidget_GetStyle tests style getter
func TestBaseWidget_GetStyle(t *testing.T) {
	style := &config.StyleConfig{
		Background: 128,
		Border:     255,
	}

	base := NewBaseWidget(config.WidgetConfig{
		ID:    "test",
		Style: style,
	})

	result := base.GetStyle()
	if result.Background != style.Background {
		t.Errorf("GetStyle().Background = %v, want %v", result.Background, style.Background)
	}
	if result.Border != style.Border {
		t.Errorf("GetStyle().Border = %v, want %v", result.Border, style.Border)
	}
}

// TestNewBaseWidget_DefaultInterval tests default update interval
func TestNewBaseWidget_DefaultInterval(t *testing.T) {
	base := NewBaseWidget(config.WidgetConfig{
		ID:             "test",
		UpdateInterval: 0, // Not specified
	})

	expected := 1 * time.Second // Default is 1.0 second
	result := base.GetUpdateInterval()
	if result != expected {
		t.Errorf("Default GetUpdateInterval() = %v, want %v", result, expected)
	}
}

// TestNewBaseWidget_DefaultAutoHideTimeout tests default auto-hide timeout
func TestNewBaseWidget_DefaultAutoHideTimeout(t *testing.T) {
	base := NewBaseWidget(config.WidgetConfig{
		ID: "test",
		AutoHide: &config.AutoHideConfig{
			Enabled: true,
			Timeout: 0, // Not specified
		},
	})

	expected := 2 * time.Second // Default is 2.0 seconds
	result := base.GetAutoHideTimeout()
	if result != expected {
		t.Errorf("Default GetAutoHideTimeout() = %v, want %v", result, expected)
	}
}
