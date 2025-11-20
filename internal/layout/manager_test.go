package layout

import (
	"image"
	"image/color"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

func TestNewManager(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:  128,
		Height: 40,
	}

	clockCfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "clock1",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          false,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			Format:          "15:04",
			FontSize:        12,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
		},
	}

	clockWidget, err := widget.NewClockWidget(clockCfg)
	if err != nil {
		t.Fatalf("failed to create clock widget: %v", err)
	}

	widgets := []widget.Widget{clockWidget}

	mgr := NewManager(displayCfg, widgets)

	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestManagerComposite(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:  128,
		Height: 40,
	}

	clockCfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "clock1",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          false,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			Format:          "15:04",
			FontSize:        12,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
		},
	}

	clockWidget, err := widget.NewClockWidget(clockCfg)
	if err != nil {
		t.Fatalf("failed to create clock widget: %v", err)
	}

	// Update widget before compositing
	if err := clockWidget.Update(); err != nil {
		t.Fatalf("failed to update widget: %v", err)
	}

	widgets := []widget.Widget{clockWidget}
	mgr := NewManager(displayCfg, widgets)

	img, err := mgr.Composite()
	if err != nil {
		t.Fatalf("Composite() error = %v", err)
	}

	if img == nil {
		t.Fatal("Composite() returned nil image")
	}

	if img.Bounds().Dx() != 128 {
		t.Errorf("composite width = %d, want 128", img.Bounds().Dx())
	}

	if img.Bounds().Dy() != 40 {
		t.Errorf("composite height = %d, want 40", img.Bounds().Dy())
	}
}

func TestManagerCompositeMultipleWidgets(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:  128,
		Height: 40,
	}

	clock1Cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "clock1",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 64,
			H: 40,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          true,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			Format:          "15:04",
			FontSize:        10,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
		},
	}

	clock2Cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "clock2",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 64,
			Y: 0,
			W: 64,
			H: 40,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          true,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			Format:          "15:04:05",
			FontSize:        10,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
		},
	}

	clock1, err := widget.NewClockWidget(clock1Cfg)
	if err != nil {
		t.Fatalf("failed to create clock1: %v", err)
	}

	clock2, err := widget.NewClockWidget(clock2Cfg)
	if err != nil {
		t.Fatalf("failed to create clock2: %v", err)
	}

	if err := clock1.Update(); err != nil {
		t.Fatalf("failed to update clock1: %v", err)
	}
	if err := clock2.Update(); err != nil {
		t.Fatalf("failed to update clock2: %v", err)
	}

	widgets := []widget.Widget{clock1, clock2}
	mgr := NewManager(displayCfg, widgets)

	img, err := mgr.Composite()
	if err != nil {
		t.Fatalf("Composite() error = %v", err)
	}

	if img == nil {
		t.Fatal("Composite() returned nil image")
	}
}

// mockWidgetWithRaceDetection is a widget that tracks concurrent Update() calls
type mockWidgetWithRaceDetection struct {
	id               string
	position         config.PositionConfig
	style            config.StyleConfig
	updateCounter    int32      // Tracks number of concurrent Update calls
	updateInProgress int32      // Atomic flag: 1 if Update is in progress
	raceDetected     int32      // Set to 1 if race is detected
	mu               sync.Mutex // Used to simulate concurrent access issues
}

func newMockWidgetWithRaceDetection(id string, x, y, w, h int) *mockWidgetWithRaceDetection {
	return &mockWidgetWithRaceDetection{
		id: id,
		position: config.PositionConfig{
			X: x, Y: y, W: w, H: h,
			ZOrder: 0,
		},
		style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          false,
		},
	}
}

func (m *mockWidgetWithRaceDetection) Name() string {
	return m.id
}

func (m *mockWidgetWithRaceDetection) Update() error {
	// Detect if another Update() is already running
	if atomic.CompareAndSwapInt32(&m.updateInProgress, 0, 1) {
		defer atomic.StoreInt32(&m.updateInProgress, 0)

		// Increment counter
		atomic.AddInt32(&m.updateCounter, 1)

		// Simulate some work with mutex access (exposes race condition)
		m.mu.Lock()
		defer m.mu.Unlock()

		// Sleep to increase chance of concurrent access
		time.Sleep(10 * time.Millisecond)
	} else {
		// Another Update() is already running - race detected!
		atomic.StoreInt32(&m.raceDetected, 1)
	}

	return nil
}

func (m *mockWidgetWithRaceDetection) Render() (image.Image, error) {
	img := image.NewGray(image.Rect(0, 0, m.position.W, m.position.H))

	// Fill with gray
	for y := 0; y < m.position.H; y++ {
		for x := 0; x < m.position.W; x++ {
			img.Set(x, y, color.Gray{Y: 128})
		}
	}

	return img, nil
}

func (m *mockWidgetWithRaceDetection) GetUpdateInterval() time.Duration {
	return 100 * time.Millisecond
}

func (m *mockWidgetWithRaceDetection) GetPosition() config.PositionConfig {
	return m.position
}

func (m *mockWidgetWithRaceDetection) GetStyle() config.StyleConfig {
	return m.style
}

// TestCompositeDoesNotCallUpdate verifies that Composite() does NOT call Update()
// This test exposes the race condition where Update() is called from multiple goroutines
func TestCompositeDoesNotCallUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	displayCfg := config.DisplayConfig{
		Width:  128,
		Height: 40,
	}

	mockWidget := newMockWidgetWithRaceDetection("test", 0, 0, 128, 40)
	widgets := []widget.Widget{mockWidget}
	mgr := NewManager(displayCfg, widgets)

	// Simulate the actual usage pattern:
	// 1. Background goroutine calls Update() periodically (like compositor does)
	// 2. Render loop calls Composite() which currently also calls Update()
	// This creates a race condition

	stopCh := make(chan struct{})
	var wg sync.WaitGroup

	// Background goroutine simulating widget update loop (like compositor.widgetUpdateLoop)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				_ = mockWidget.Update()
			}
		}
	}()

	// Main goroutine simulating render loop calling Composite()
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(15 * time.Millisecond)
		defer ticker.Stop()

		for i := 0; i < 20; i++ {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				_, err := mgr.Composite()
				if err != nil {
					t.Errorf("Composite() error = %v", err)
					return
				}
			}
		}
	}()

	// Let them run concurrently for a bit
	time.Sleep(400 * time.Millisecond)
	close(stopCh)
	wg.Wait()

	// Check if race was detected
	if atomic.LoadInt32(&mockWidget.raceDetected) == 1 {
		t.Error("Race condition detected: Update() called concurrently from multiple goroutines")
		t.Error("This happens because layout.Manager.Composite() calls widget.Update(),")
		t.Error("but widgets already have dedicated update loops in compositor")
	}

	// With the current implementation, Update() is called from both:
	// 1. Background update goroutine
	// 2. Composite() method
	// This test will fail with race detector or detect concurrent execution

	updateCount := atomic.LoadInt32(&mockWidget.updateCounter)
	t.Logf("Update() was called %d times", updateCount)

	// NOTE: After fix, Composite() should NOT call Update()
	// Update should only be called by the background goroutine
}
