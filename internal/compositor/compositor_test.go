package compositor

import (
	"errors"
	"image"
	"sync"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/layout"
	"github.com/pozitronik/steelclock-go/internal/testutil"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

// mockWidget implements widget.Widget for testing
type mockWidget struct {
	name           string
	updateInterval time.Duration
	position       config.PositionConfig
	style          config.StyleConfig
	updateCalls    int
	updateErr      error
	renderCalls    int
	renderResult   image.Image
	renderErr      error
	mu             sync.Mutex
}

func newMockWidget(name string, x, y, w, h int) *mockWidget {
	return &mockWidget{
		name:           name,
		updateInterval: 100 * time.Millisecond,
		position: config.PositionConfig{
			X: x, Y: y, W: w, H: h,
		},
		style: config.StyleConfig{
			Background: 0,
		},
	}
}

func (m *mockWidget) Name() string {
	return m.name
}

func (m *mockWidget) Update() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCalls++
	return m.updateErr
}

func (m *mockWidget) Render() (image.Image, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.renderCalls++
	if m.renderErr != nil {
		return nil, m.renderErr
	}
	if m.renderResult != nil {
		return m.renderResult, nil
	}
	// Return default gray image
	img := image.NewGray(image.Rect(0, 0, m.position.W, m.position.H))
	return img, nil
}

func (m *mockWidget) GetUpdateInterval() time.Duration {
	return m.updateInterval
}

func (m *mockWidget) GetPosition() config.PositionConfig {
	return m.position
}

func (m *mockWidget) GetStyle() config.StyleConfig {
	return m.style
}

func (m *mockWidget) GetUpdateCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateCalls
}

func (m *mockWidget) GetRenderCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.renderCalls
}

// Helper to create layout manager
func createLayoutManager(widgets []widget.Widget) *layout.Manager {
	displayCfg := config.DisplayConfig{
		Width:      128,
		Height:     40,
		Background: 0,
	}
	return layout.NewManager(displayCfg, widgets)
}

// TestNewCompositor tests compositor creation
func TestNewCompositor(t *testing.T) {
	client := testutil.NewTestClient()
	widgets := []widget.Widget{
		newMockWidget("widget1", 0, 0, 64, 40),
	}
	displayCfg := config.DisplayConfig{
		Width:      128,
		Height:     40,
		Background: 0,
	}
	layoutMgr := layout.NewManager(displayCfg, widgets)
	cfg := &config.Config{
		RefreshRateMs: 100,
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	if comp == nil {
		t.Fatal("NewCompositor() returned nil")
	}

	if comp.client == nil {
		t.Error("client is nil")
	}

	if comp.layoutManager == nil {
		t.Error("layoutManager is nil")
	}

	if comp.refreshRate != 100*time.Millisecond {
		t.Errorf("refreshRate = %v, want 100ms", comp.refreshRate)
	}

	if comp.eventName != "STEELCLOCK_DISPLAY" {
		t.Errorf("eventName = %s, want STEELCLOCK_DISPLAY", comp.eventName)
	}

	if len(comp.widgets) != 1 {
		t.Errorf("len(widgets) = %d, want 1", len(comp.widgets))
	}
}

// TestCompositor_StartStop tests starting and stopping the compositor
func TestCompositor_StartStop(t *testing.T) {
	client := testutil.NewTestClient()
	widgets := []widget.Widget{
		newMockWidget("widget1", 0, 0, 64, 40),
	}
	layoutMgr := createLayoutManager(widgets)
	cfg := &config.Config{
		RefreshRateMs: 50, // Fast for testing
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	// Start compositor
	err := comp.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Let it run briefly
	time.Sleep(200 * time.Millisecond)

	// Stop compositor
	comp.Stop()

	// Verify widgets were updated
	mockW := widgets[0].(*mockWidget)
	updateCalls := mockW.GetUpdateCalls()
	if updateCalls < 1 {
		t.Errorf("Widget update calls = %d, want at least 1", updateCalls)
	}

	// Verify frames were sent
	frameCount := client.FrameCount()
	if frameCount < 1 {
		t.Errorf("Frame count = %d, want at least 1", frameCount)
	}
}

// TestCompositor_RenderFrame tests single frame rendering
func TestCompositor_RenderFrame(t *testing.T) {
	client := testutil.NewTestClient()

	// Create widget with known render result
	mockW := newMockWidget("widget1", 0, 0, 128, 40)
	img := image.NewGray(image.Rect(0, 0, 128, 40))
	mockW.renderResult = img

	widgets := []widget.Widget{mockW}
	layoutMgr := createLayoutManager(widgets)
	cfg := &config.Config{
		RefreshRateMs: 100,
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	// Render a frame
	err := comp.renderFrame()
	if err != nil {
		t.Fatalf("renderFrame() error = %v", err)
	}

	// Verify frame was captured
	if client.FrameCount() != 1 {
		t.Errorf("Frame count = %d, want 1", client.FrameCount())
	}

	// Verify bitmap data was sent
	lastFrame := client.LastFrame()
	if lastFrame == nil || len(lastFrame.Data) == 0 {
		t.Error("No bitmap data was sent")
	}

	// Verify widget was rendered
	if mockW.GetRenderCalls() != 1 {
		t.Errorf("Widget render calls = %d, want 1", mockW.GetRenderCalls())
	}
}

// TestCompositor_RenderFrame_SendError tests error handling during send
func TestCompositor_RenderFrame_SendError(t *testing.T) {
	client := testutil.NewTestClient()
	client.SetSendError(errors.New("send error"), 0) // Fail all sends

	mockW := newMockWidget("widget1", 0, 0, 128, 40)
	widgets := []widget.Widget{mockW}
	layoutMgr := createLayoutManager(widgets)
	cfg := &config.Config{
		RefreshRateMs: 100,
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	// Render should return error
	err := comp.renderFrame()
	if err == nil {
		t.Error("renderFrame() should return error when SendScreenData fails")
	}
}

// TestCompositor_MultipleWidgets tests compositor with multiple widgets
func TestCompositor_MultipleWidgets(t *testing.T) {
	client := testutil.NewTestClient()

	widget1 := newMockWidget("widget1", 0, 0, 64, 40)
	widget2 := newMockWidget("widget2", 64, 0, 64, 40)
	widget3 := newMockWidget("widget3", 0, 0, 32, 20)

	widgets := []widget.Widget{widget1, widget2, widget3}
	layoutMgr := createLayoutManager(widgets)

	cfg := &config.Config{
		RefreshRateMs: 50,
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	err := comp.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Let it run
	time.Sleep(200 * time.Millisecond)

	comp.Stop()

	// Verify all widgets were updated
	for i, w := range widgets {
		mockW := w.(*mockWidget)
		updateCalls := mockW.GetUpdateCalls()
		if updateCalls < 1 {
			t.Errorf("Widget %d update calls = %d, want at least 1", i, updateCalls)
		}
	}

	// Verify frames were sent
	if client.FrameCount() < 1 {
		t.Errorf("Frame count = %d, want at least 1", client.FrameCount())
	}
}

// TestCompositor_WidgetUpdateError tests handling of widget update errors
func TestCompositor_WidgetUpdateError(t *testing.T) {
	client := testutil.NewTestClient()

	mockW := newMockWidget("widget1", 0, 0, 128, 40)
	mockW.updateErr = errors.New("update error")
	widgets := []widget.Widget{mockW}
	layoutMgr := createLayoutManager(widgets)
	cfg := &config.Config{
		RefreshRateMs: 50,
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	err := comp.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Let it run - should not crash despite widget errors
	time.Sleep(200 * time.Millisecond)

	comp.Stop()

	// Compositor should continue despite widget errors
	// Verify update was attempted
	if mockW.GetUpdateCalls() < 1 {
		t.Error("Widget update should be attempted despite errors")
	}
}

// TestCompositor_Heartbeat tests heartbeat loop
func TestCompositor_Heartbeat(t *testing.T) {
	client := testutil.NewTestClient()
	mockW := newMockWidget("widget1", 0, 0, 128, 40)
	widgets := []widget.Widget{mockW}
	layoutMgr := createLayoutManager(widgets)

	cfg := &config.Config{
		RefreshRateMs: 100,
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	// Override heartbeat interval for testing by starting manually
	// and checking after a short period
	err := comp.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait less than heartbeat interval (10s), so we may not see heartbeats
	// in normal test run, but the goroutine should be running
	time.Sleep(100 * time.Millisecond)

	comp.Stop()

	// Heartbeat goroutine should have started and stopped cleanly
	// We can't easily test heartbeat calls without mocking time or waiting 10s
}

// TestCompositor_FastRefreshRate tests with very fast refresh rate
func TestCompositor_FastRefreshRate(t *testing.T) {
	client := testutil.NewTestClient()

	mockW := newMockWidget("widget1", 0, 0, 128, 40)
	widgets := []widget.Widget{mockW}
	layoutMgr := createLayoutManager(widgets)
	cfg := &config.Config{
		RefreshRateMs: 10, // Very fast - 10ms
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	err := comp.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Let it run
	time.Sleep(200 * time.Millisecond)

	comp.Stop()

	// With 10ms refresh rate, should have many frames in 200ms
	frameCount := client.FrameCount()
	if frameCount < 5 {
		t.Errorf("Frame count = %d, want at least 5 with fast refresh", frameCount)
	}

	// Verify timing stats
	stats := client.CalculateTimingStats()
	t.Logf("Fast refresh: %d frames, avg interval %v, FPS %.1f",
		stats.FrameCount, stats.AvgInterval, stats.AverageFPS)
}

// TestCompositor_SlowRefreshRate tests with slow refresh rate
func TestCompositor_SlowRefreshRate(t *testing.T) {
	client := testutil.NewTestClient()

	mockW := newMockWidget("widget1", 0, 0, 128, 40)
	widgets := []widget.Widget{mockW}
	layoutMgr := createLayoutManager(widgets)
	cfg := &config.Config{
		RefreshRateMs: 200, // Slow - 200ms
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	err := comp.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Let it run
	time.Sleep(300 * time.Millisecond)

	comp.Stop()

	// With 200ms refresh rate, should have few calls in 300ms
	frameCount := client.FrameCount()
	if frameCount > 3 {
		t.Logf("Frame count = %d (acceptable for slow refresh)", frameCount)
	}

	// Verify timing
	stats := client.CalculateTimingStats()
	t.Logf("Slow refresh: %d frames, avg interval %v", stats.FrameCount, stats.AvgInterval)
}

// TestCompositor_NoWidgets tests compositor with no widgets
func TestCompositor_NoWidgets(t *testing.T) {
	client := testutil.NewTestClient()

	var widgets []widget.Widget // No widgets
	layoutMgr := createLayoutManager(widgets)
	cfg := &config.Config{
		RefreshRateMs: 50,
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	err := comp.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	comp.Stop()

	// Should still send frames (blank frames)
	if client.FrameCount() < 1 {
		t.Errorf("Frame count = %d, want at least 1 even with no widgets", client.FrameCount())
	}

	// Verify frames are blank
	lastFrame := client.LastFrame()
	if lastFrame != nil && !testutil.IsBlankFrame(lastFrame.Data) {
		t.Error("Frame should be blank with no widgets")
	}
}

// TestCompositor_StopWithoutStart tests stopping before starting
func TestCompositor_StopWithoutStart(t *testing.T) {
	client := testutil.NewTestClient()
	mockW := newMockWidget("widget1", 0, 0, 128, 40)
	widgets := []widget.Widget{mockW}
	layoutMgr := createLayoutManager(widgets)

	cfg := &config.Config{
		RefreshRateMs: 100,
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	// Stop without starting - should not panic
	comp.Stop()
}

// TestCompositor_MultipleStartStop tests multiple start/stop cycles
func TestCompositor_MultipleStartStop(t *testing.T) {
	client := testutil.NewTestClient()

	mockW := newMockWidget("widget1", 0, 0, 128, 40)
	widgets := []widget.Widget{mockW}
	layoutMgr := createLayoutManager(widgets)
	cfg := &config.Config{
		RefreshRateMs: 50,
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	// First cycle
	err := comp.Start()
	if err != nil {
		t.Fatalf("First Start() error = %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	comp.Stop()

	// Note: In the actual implementation, Start() can't be called again
	// after Stop() because stopChan is closed. This is the expected behavior.
	// Once a compositor is stopped, a new one must be created.
}

// TestCompositor_WidgetDifferentUpdateIntervals tests widgets with different update rates
func TestCompositor_WidgetDifferentUpdateIntervals(t *testing.T) {
	client := testutil.NewTestClient()

	fastWidget := newMockWidget("fast", 0, 0, 64, 40)
	fastWidget.updateInterval = 50 * time.Millisecond

	slowWidget := newMockWidget("slow", 64, 0, 64, 40)
	slowWidget.updateInterval = 200 * time.Millisecond

	widgets := []widget.Widget{fastWidget, slowWidget}
	layoutMgr := createLayoutManager(widgets)

	cfg := &config.Config{
		RefreshRateMs: 50,
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	err := comp.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	comp.Stop()

	// Fast widget should have more updates than slow widget
	fastCalls := fastWidget.GetUpdateCalls()
	slowCalls := slowWidget.GetUpdateCalls()

	if fastCalls <= slowCalls {
		t.Errorf("Fast widget calls (%d) should be more than slow widget calls (%d)", fastCalls, slowCalls)
	}
}

// TestCompositor_FrameContentVerification tests that frame content can be verified
func TestCompositor_FrameContentVerification(t *testing.T) {
	client := testutil.NewTestClient()

	// Create widget that renders white pixels
	mockW := newMockWidget("widget1", 0, 0, 128, 40)
	img := image.NewGray(image.Rect(0, 0, 128, 40))
	// Fill with white
	for y := 0; y < 40; y++ {
		for x := 0; x < 128; x++ {
			img.Set(x, y, image.White)
		}
	}
	mockW.renderResult = img

	widgets := []widget.Widget{mockW}
	layoutMgr := createLayoutManager(widgets)
	cfg := &config.Config{
		RefreshRateMs: 100,
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	_ = comp.renderFrame()

	// Verify frame is full (all white)
	lastFrame := client.LastFrame()
	if lastFrame == nil {
		t.Fatal("No frame captured")
	}

	if !testutil.IsFullFrame(lastFrame.Data) {
		pixelCount := testutil.CountSetPixels(lastFrame.Data)
		t.Errorf("Expected full white frame, got %d pixels set", pixelCount)
	}
}

// TestCompositor_ErrorRecovery tests that compositor recovers from transient errors
func TestCompositor_ErrorRecovery(t *testing.T) {
	client := testutil.NewTestClient()

	mockW := newMockWidget("widget1", 0, 0, 128, 40)
	widgets := []widget.Widget{mockW}
	layoutMgr := createLayoutManager(widgets)
	cfg := &config.Config{
		RefreshRateMs: 50,
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	// Inject error for first 3 sends, then recover
	client.SetSendError(errors.New("transient error"), 3)

	err := comp.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for recovery
	time.Sleep(400 * time.Millisecond)

	comp.Stop()

	// Should have captured frames after error recovery
	if client.FrameCount() == 0 {
		t.Error("Expected frames after error recovery")
	}

	t.Logf("Captured %d frames after transient errors", client.FrameCount())
}
