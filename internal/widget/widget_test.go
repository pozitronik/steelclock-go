package widget

import (
	"image"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// mockWidget implements Widget interface for testing
type mockWidget struct {
	name string
}

func (m *mockWidget) Name() string                       { return m.name }
func (m *mockWidget) Update() error                      { return nil }
func (m *mockWidget) Render() (image.Image, error)       { return nil, nil }
func (m *mockWidget) GetUpdateInterval() time.Duration   { return time.Second }
func (m *mockWidget) GetPosition() config.PositionConfig { return config.PositionConfig{} }
func (m *mockWidget) GetStyle() config.StyleConfig       { return config.StyleConfig{} }

// mockStoppableWidget implements both Widget and Stoppable interfaces
type mockStoppableWidget struct {
	mockWidget
	stopped bool
}

func (m *mockStoppableWidget) Stop() {
	m.stopped = true
}

func TestStopWidget_WithStoppable(t *testing.T) {
	w := &mockStoppableWidget{mockWidget: mockWidget{name: "test"}}

	if w.stopped {
		t.Error("Widget should not be stopped initially")
	}

	StopWidget(w)

	if !w.stopped {
		t.Error("Widget should be stopped after StopWidget call")
	}
}

func TestStopWidget_WithNonStoppable(t *testing.T) {
	w := &mockWidget{name: "test"}

	// Should not panic when called on non-stoppable widget
	StopWidget(w)
}

func TestStopWidgets_Mixed(t *testing.T) {
	stoppable1 := &mockStoppableWidget{mockWidget: mockWidget{name: "stoppable1"}}
	stoppable2 := &mockStoppableWidget{mockWidget: mockWidget{name: "stoppable2"}}
	nonStoppable := &mockWidget{name: "nonStoppable"}

	widgets := []Widget{stoppable1, nonStoppable, stoppable2}

	StopWidgets(widgets)

	if !stoppable1.stopped {
		t.Error("stoppable1 should be stopped")
	}
	if !stoppable2.stopped {
		t.Error("stoppable2 should be stopped")
	}
}

func TestStopWidgets_Empty(t *testing.T) {
	// Should not panic with empty slice
	StopWidgets([]Widget{})
}

func TestStopWidgets_Nil(t *testing.T) {
	// Should not panic with nil slice
	StopWidgets(nil)
}

// Verify that existing widgets with Stop() method implement Stoppable
func TestStoppableInterfaceCompliance(t *testing.T) {
	// This test verifies at compile time that widgets with Stop()
	// can be used as Stoppable. We use type assertions to verify.

	// Create minimal configs for widgets that have Stop() methods
	batteryConfig := config.WidgetConfig{
		Type:     "battery",
		ID:       "test_battery",
		Enabled:  config.BoolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 64, H: 20},
	}

	// BatteryWidget has Stop() - verify it implements Stoppable
	battery, err := NewBatteryWidget(batteryConfig)
	if err != nil {
		t.Skipf("Could not create BatteryWidget: %v", err)
	}

	// Type assertion should succeed
	if _, ok := interface{}(battery).(Stoppable); !ok {
		t.Error("BatteryWidget should implement Stoppable interface")
	}
}
