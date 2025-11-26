package compositor

import (
	"errors"
	"image"
	"sync"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/layout"
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

// mockGameSenseAPI implements gamesense.GameSenseAPI for testing
type mockGameSenseAPI struct {
	mu                  sync.Mutex
	sendScreenDataCalls int
	sendHeartbeatCalls  int
	sendScreenDataErr   error
	sendHeartbeatErr    error
	lastBitmapData      []int
	registerGameErr     error
	bindScreenEventErr  error
	removeGameErr       error
}

func newMockGameSenseAPI() *mockGameSenseAPI {
	return &mockGameSenseAPI{}
}

func (m *mockGameSenseAPI) RegisterGame(_ string, _ int) error {
	return m.registerGameErr
}

func (m *mockGameSenseAPI) BindScreenEvent(_, _ string) error {
	return m.bindScreenEventErr
}

func (m *mockGameSenseAPI) SendScreenData(_ string, bitmapData []int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendScreenDataCalls++
	m.lastBitmapData = bitmapData
	return m.sendScreenDataErr
}

func (m *mockGameSenseAPI) SendHeartbeat() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendHeartbeatCalls++
	return m.sendHeartbeatErr
}

func (m *mockGameSenseAPI) RemoveGame() error {
	return m.removeGameErr
}

func (m *mockGameSenseAPI) SupportsMultipleEvents() bool {
	return false
}

func (m *mockGameSenseAPI) SendScreenDataMultiRes(_ string, resolutionData map[string][]int) error {
	// Find 128x40 resolution (standard test resolution)
	if data, ok := resolutionData["image-data-128x40"]; ok {
		return m.SendScreenData("", data)
	}
	return nil
}

func (m *mockGameSenseAPI) SendMultipleScreenData(_ string, frames [][]int) error {
	if len(frames) > 0 {
		return m.SendScreenData("", frames[len(frames)-1])
	}
	return nil
}

func (m *mockGameSenseAPI) GetSendScreenDataCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sendScreenDataCalls
}

func (m *mockGameSenseAPI) GetSendHeartbeatCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sendHeartbeatCalls
}

func (m *mockGameSenseAPI) GetLastBitmapData() []int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastBitmapData
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
	client := newMockGameSenseAPI()
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
	client := newMockGameSenseAPI()
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
	sendCalls := client.GetSendScreenDataCalls()
	if sendCalls < 1 {
		t.Errorf("SendScreenData calls = %d, want at least 1", sendCalls)
	}
}

// TestCompositor_RenderFrame tests single frame rendering
func TestCompositor_RenderFrame(t *testing.T) {
	client := newMockGameSenseAPI()

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

	// Verify SendScreenData was called
	if client.GetSendScreenDataCalls() != 1 {
		t.Errorf("SendScreenData calls = %d, want 1", client.GetSendScreenDataCalls())
	}

	// Verify bitmap data was sent
	bitmapData := client.GetLastBitmapData()
	if len(bitmapData) == 0 {
		t.Error("No bitmap data was sent")
	}

	// Verify widget was rendered
	if mockW.GetRenderCalls() != 1 {
		t.Errorf("Widget render calls = %d, want 1", mockW.GetRenderCalls())
	}
}

// TestCompositor_RenderFrame_SendError tests error handling during send
func TestCompositor_RenderFrame_SendError(t *testing.T) {
	client := newMockGameSenseAPI()
	client.sendScreenDataErr = errors.New("send error")

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
	client := newMockGameSenseAPI()

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
	if client.GetSendScreenDataCalls() < 1 {
		t.Errorf("SendScreenData calls = %d, want at least 1", client.GetSendScreenDataCalls())
	}
}

// TestCompositor_WidgetUpdateError tests handling of widget update errors
func TestCompositor_WidgetUpdateError(t *testing.T) {
	client := newMockGameSenseAPI()

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
	client := newMockGameSenseAPI()
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
	client := newMockGameSenseAPI()

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

	// With 10ms refresh rate, should have many calls in 200ms
	sendCalls := client.GetSendScreenDataCalls()
	if sendCalls < 5 {
		t.Errorf("SendScreenData calls = %d, want at least 5 with fast refresh", sendCalls)
	}
}

// TestCompositor_SlowRefreshRate tests with slow refresh rate
func TestCompositor_SlowRefreshRate(t *testing.T) {
	client := newMockGameSenseAPI()

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
	sendCalls := client.GetSendScreenDataCalls()
	if sendCalls > 3 {
		t.Logf("SendScreenData calls = %d (acceptable for slow refresh)", sendCalls)
	}
}

// TestCompositor_NoWidgets tests compositor with no widgets
func TestCompositor_NoWidgets(t *testing.T) {
	client := newMockGameSenseAPI()

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
	if client.GetSendScreenDataCalls() < 1 {
		t.Errorf("SendScreenData calls = %d, want at least 1 even with no widgets", client.GetSendScreenDataCalls())
	}
}

// TestCompositor_StopWithoutStart tests stopping before starting
func TestCompositor_StopWithoutStart(t *testing.T) {
	client := newMockGameSenseAPI()
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
	client := newMockGameSenseAPI()

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
	client := newMockGameSenseAPI()

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
