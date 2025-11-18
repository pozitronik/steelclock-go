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
				ID: "test",
				Properties: config.WidgetProperties{
					UpdateInterval: tt.interval,
				},
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
		X:      10,
		Y:      20,
		W:      128,
		H:      40,
		ZOrder: 5,
	}

	base := NewBaseWidget(config.WidgetConfig{
		ID:       "test",
		Position: pos,
	})

	result := base.GetPosition()
	if result.X != pos.X || result.Y != pos.Y || result.W != pos.W || result.H != pos.H || result.ZOrder != pos.ZOrder {
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
				Properties: config.WidgetProperties{
					AutoHide:        true,
					AutoHideTimeout: tt.timeout,
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
			Properties: config.WidgetProperties{
				AutoHide: false,
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
			Properties: config.WidgetProperties{
				AutoHide:        true,
				AutoHideTimeout: 0.1, // 100ms
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
			Properties: config.WidgetProperties{
				AutoHide:        true,
				AutoHideTimeout: 0.2, // 200ms
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
			Properties: config.WidgetProperties{
				AutoHide:        true,
				AutoHideTimeout: 0.3, // 300ms
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
