package compositor

import (
	"image"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

// mockSchedulerWidget implements widget.Widget for scheduler tests
type mockSchedulerWidget struct {
	name           string
	updateInterval time.Duration
	updateCount    atomic.Int32
}

func newMockSchedulerWidget(name string, interval time.Duration) *mockSchedulerWidget {
	return &mockSchedulerWidget{
		name:           name,
		updateInterval: interval,
	}
}

func (w *mockSchedulerWidget) Name() string                     { return w.name }
func (w *mockSchedulerWidget) Update() error                    { w.updateCount.Add(1); return nil }
func (w *mockSchedulerWidget) Render() (image.Image, error)     { return nil, nil }
func (w *mockSchedulerWidget) GetUpdateInterval() time.Duration { return w.updateInterval }
func (w *mockSchedulerWidget) GetPosition() config.PositionConfig {
	return config.PositionConfig{X: 0, Y: 0, W: 10, H: 10}
}
func (w *mockSchedulerWidget) GetStyle() config.StyleConfig { return config.StyleConfig{} }
func (w *mockSchedulerWidget) GetUpdateCount() int          { return int(w.updateCount.Load()) }

func TestNewWidgetScheduler(t *testing.T) {
	widgets := []widget.Widget{
		newMockSchedulerWidget("w1", 100*time.Millisecond),
		newMockSchedulerWidget("w2", 200*time.Millisecond),
	}

	scheduler := NewWidgetScheduler(widgets)

	if scheduler.WidgetCount() != 2 {
		t.Errorf("WidgetCount() = %d, want 2", scheduler.WidgetCount())
	}

	if scheduler.IsRunning() {
		t.Error("New scheduler should not be running")
	}
}

func TestWidgetScheduler_StartStop(t *testing.T) {
	widgets := []widget.Widget{
		newMockSchedulerWidget("w1", 50*time.Millisecond),
	}

	scheduler := NewWidgetScheduler(widgets)

	scheduler.Start()
	if !scheduler.IsRunning() {
		t.Error("Scheduler should be running after Start()")
	}

	// Allow some updates to occur
	time.Sleep(100 * time.Millisecond)

	scheduler.Stop()
	if scheduler.IsRunning() {
		t.Error("Scheduler should not be running after Stop()")
	}
}

func TestWidgetScheduler_WidgetUpdates(t *testing.T) {
	w := newMockSchedulerWidget("test", 20*time.Millisecond)
	widgets := []widget.Widget{w}

	scheduler := NewWidgetScheduler(widgets)
	scheduler.Start()

	// Wait for several update cycles
	time.Sleep(100 * time.Millisecond)

	scheduler.Stop()

	// Should have at least initial update + a few timer updates
	// Initial update + ~4 timer updates (100ms / 20ms = 5 cycles)
	updateCount := w.GetUpdateCount()
	if updateCount < 3 {
		t.Errorf("Widget should have been updated at least 3 times, got %d", updateCount)
	}
}

func TestWidgetScheduler_MultipleWidgets(t *testing.T) {
	w1 := newMockSchedulerWidget("fast", 20*time.Millisecond)
	w2 := newMockSchedulerWidget("slow", 50*time.Millisecond)
	widgets := []widget.Widget{w1, w2}

	scheduler := NewWidgetScheduler(widgets)
	scheduler.Start()

	time.Sleep(120 * time.Millisecond)

	scheduler.Stop()

	// Fast widget should have more updates than slow widget
	fastCount := w1.GetUpdateCount()
	slowCount := w2.GetUpdateCount()

	if fastCount <= slowCount {
		t.Errorf("Fast widget (%d updates) should have more updates than slow widget (%d updates)",
			fastCount, slowCount)
	}
}

func TestWidgetScheduler_DoubleStart(t *testing.T) {
	widgets := []widget.Widget{
		newMockSchedulerWidget("w1", 50*time.Millisecond),
	}

	scheduler := NewWidgetScheduler(widgets)

	scheduler.Start()
	scheduler.Start() // Should be safe to call twice

	if !scheduler.IsRunning() {
		t.Error("Scheduler should still be running")
	}

	scheduler.Stop()
}

func TestWidgetScheduler_DoubleStop(t *testing.T) {
	widgets := []widget.Widget{
		newMockSchedulerWidget("w1", 50*time.Millisecond),
	}

	scheduler := NewWidgetScheduler(widgets)

	scheduler.Start()
	scheduler.Stop()
	scheduler.Stop() // Should be safe to call twice

	if scheduler.IsRunning() {
		t.Error("Scheduler should not be running")
	}
}

func TestWidgetScheduler_StopWithoutStart(t *testing.T) {
	widgets := []widget.Widget{
		newMockSchedulerWidget("w1", 50*time.Millisecond),
	}

	scheduler := NewWidgetScheduler(widgets)

	// Should not panic
	scheduler.Stop()

	if scheduler.IsRunning() {
		t.Error("Scheduler should not be running")
	}
}

func TestWidgetScheduler_EmptyWidgets(t *testing.T) {
	scheduler := NewWidgetScheduler([]widget.Widget{})

	if scheduler.WidgetCount() != 0 {
		t.Errorf("WidgetCount() = %d, want 0", scheduler.WidgetCount())
	}

	// Should handle empty widget list gracefully
	scheduler.Start()
	time.Sleep(50 * time.Millisecond)
	scheduler.Stop()
}
