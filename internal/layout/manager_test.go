package layout

import (
	"fmt"
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
		Style: &config.StyleConfig{
			Background:  0,
			Border:      false,
			BorderColor: 255,
		},
		Text: &config.TextConfig{
			Format: "15:04",
			Size:   12,
			Align:  &config.AlignConfig{H: "center", V: "center"},
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
		Style: &config.StyleConfig{
			Background:  0,
			Border:      false,
			BorderColor: 255,
		},
		Text: &config.TextConfig{
			Format: "15:04",
			Size:   12,
			Align:  &config.AlignConfig{H: "center", V: "center"},
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
		Style: &config.StyleConfig{
			Background:  0,
			Border:      true,
			BorderColor: 255,
		},
		Text: &config.TextConfig{
			Format: "15:04",
			Size:   10,
			Align:  &config.AlignConfig{H: "center", V: "center"},
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
		Style: &config.StyleConfig{
			Background:  0,
			Border:      true,
			BorderColor: 255,
		},
		Text: &config.TextConfig{
			Format: "15:04:05",
			Size:   10,
			Align:  &config.AlignConfig{H: "center", V: "center"},
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
			Z: 0,
		},
		style: config.StyleConfig{
			Background: 0,
			Border:     false,
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

// mockWidgetSimple is a simple mock widget for testing
type mockWidgetSimple struct {
	name     string
	position config.PositionConfig
	style    config.StyleConfig
	img      image.Image
	err      error
}

func newMockWidgetSimple(name string, x, y, w, h, zOrder int) *mockWidgetSimple {
	return &mockWidgetSimple{
		name: name,
		position: config.PositionConfig{
			X: x, Y: y, W: w, H: h,
			Z: zOrder,
		},
		style: config.StyleConfig{
			Background: 0,
			Border:     false,
		},
	}
}

func (m *mockWidgetSimple) Name() string {
	return m.name
}

func (m *mockWidgetSimple) Update() error {
	return nil
}

func (m *mockWidgetSimple) Render() (image.Image, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.img != nil {
		return m.img, nil
	}
	// Default: create simple gray image
	img := image.NewGray(image.Rect(0, 0, m.position.W, m.position.H))
	// Fill with non-zero value
	for y := 0; y < m.position.H; y++ {
		for x := 0; x < m.position.W; x++ {
			img.Set(x, y, color.Gray{Y: 128})
		}
	}
	return img, nil
}

func (m *mockWidgetSimple) GetUpdateInterval() time.Duration {
	return 1 * time.Second
}

func (m *mockWidgetSimple) GetPosition() config.PositionConfig {
	return m.position
}

func (m *mockWidgetSimple) GetStyle() config.StyleConfig {
	return m.style
}

func TestComposite_EmptyWidgetList(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:      128,
		Height:     40,
		Background: 0,
	}

	mgr := NewManager(displayCfg, []widget.Widget{})
	img, err := mgr.Composite()

	if err != nil {
		t.Errorf("Composite() with empty widget list error = %v", err)
	}

	if img == nil {
		t.Fatal("Composite() returned nil image")
	}

	if img.Bounds().Dx() != 128 || img.Bounds().Dy() != 40 {
		t.Errorf("Composite() size = %dx%d, want 128x40", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestComposite_HiddenWidget(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:      128,
		Height:     40,
		Background: 0,
	}

	mockWidget := newMockWidgetSimple("hidden", 0, 0, 128, 40, 0)

	// Use helper widget that returns nil from Render (simulating auto-hide)
	hiddenWidget := &mockWidgetWithNilRender{
		mockWidgetSimple: mockWidget,
	}

	mgr := NewManager(displayCfg, []widget.Widget{hiddenWidget})
	img, err := mgr.Composite()

	if err != nil {
		t.Errorf("Composite() with hidden widget should not error, got: %v", err)
	}

	if img == nil {
		t.Fatal("Composite() returned nil image")
	}
}

// Helper widget that renders nil
type mockWidgetWithNilRender struct {
	*mockWidgetSimple
}

func (m *mockWidgetWithNilRender) Render() (image.Image, error) {
	return nil, nil
}

func TestComposite_WidgetRenderError(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:      128,
		Height:     40,
		Background: 0,
	}

	mockWidget := newMockWidgetSimple("error", 0, 0, 128, 40, 0)
	mockWidget.err = fmt.Errorf("render failed")

	// Override Render to return error
	errorWidget := &mockWidgetWithError{
		mockWidgetSimple: mockWidget,
		renderError:      fmt.Errorf("test render error"),
	}

	mgr := NewManager(displayCfg, []widget.Widget{errorWidget})
	_, err := mgr.Composite()

	if err == nil {
		t.Error("Composite() with widget render error should return error")
	}
}

type mockWidgetWithError struct {
	*mockWidgetSimple
	renderError error
}

func (m *mockWidgetWithError) Render() (image.Image, error) {
	return nil, m.renderError
}

func TestComposite_ZOrderRespected(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:      128,
		Height:     40,
		Background: 0,
	}

	// Create widgets with different z-orders
	// Widget with z-order 0 (bottom)
	widget1 := newMockWidgetSimple("bottom", 0, 0, 64, 40, 0)
	widget1Img := image.NewGray(image.Rect(0, 0, 64, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 64; x++ {
			widget1Img.Set(x, y, color.Gray{Y: 50})
		}
	}
	widget1.img = widget1Img

	// Widget with z-order 1 (middle)
	widget2 := newMockWidgetSimple("middle", 32, 0, 64, 40, 1)
	widget2Img := image.NewGray(image.Rect(0, 0, 64, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 64; x++ {
			widget2Img.Set(x, y, color.Gray{Y: 100})
		}
	}
	widget2.img = widget2Img

	// Widget with z-order 2 (top)
	widget3 := newMockWidgetSimple("top", 64, 0, 64, 40, 2)
	widget3Img := image.NewGray(image.Rect(0, 0, 64, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 64; x++ {
			widget3Img.Set(x, y, color.Gray{Y: 150})
		}
	}
	widget3.img = widget3Img

	// Add widgets in WRONG order (manager should sort by z-order)
	widgets := []widget.Widget{widget3, widget1, widget2}
	mgr := NewManager(displayCfg, widgets)

	img, err := mgr.Composite()
	if err != nil {
		t.Fatalf("Composite() error = %v", err)
	}

	grayImg, ok := img.(*image.Gray)
	if !ok {
		t.Fatal("Composite() did not return *image.Gray")
	}

	// Verify z-ordering: check overlap region (32-64, all y)
	// Widget1 (z=0, gray 50) is at x=0-64
	// Widget2 (z=1, gray 100) is at x=32-96, should overwrite widget1 in overlap
	// Widget3 (z=2, gray 150) is at x=64-128, should overwrite widget2 in overlap

	// Check x=16 (only widget1 visible, z=0, gray=50)
	if grayImg.GrayAt(16, 20).Y != 50 {
		t.Errorf("Pixel at x=16 should be from widget1 (50), got %d", grayImg.GrayAt(16, 20).Y)
	}

	// Check x=48 (widget1 and widget2 overlap, widget2 on top, z=1, gray=100)
	if grayImg.GrayAt(48, 20).Y != 100 {
		t.Errorf("Pixel at x=48 should be from widget2 (100), got %d", grayImg.GrayAt(48, 20).Y)
	}

	// Check x=80 (widget2 and widget3 overlap, widget3 on top, z=2, gray=150)
	if grayImg.GrayAt(80, 20).Y != 150 {
		t.Errorf("Pixel at x=80 should be from widget3 (150), got %d", grayImg.GrayAt(80, 20).Y)
	}
}

func TestComposite_TransparentBackground(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:      128,
		Height:     40,
		Background: 0,
	}

	// Create widget with transparent background
	widget1 := newMockWidgetSimple("transparent", 0, 0, 128, 40, 0)
	widget1.style.Background = -1 // Transparent background

	// Create image with some pixels set to 0 (background) and some to 255 (foreground)
	widget1Img := image.NewGray(image.Rect(0, 0, 128, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 128; x++ {
			if x < 64 {
				widget1Img.Set(x, y, color.Gray{Y: 0}) // Background (transparent)
			} else {
				widget1Img.Set(x, y, color.Gray{Y: 255}) // Foreground
			}
		}
	}
	widget1.img = widget1Img

	mgr := NewManager(displayCfg, []widget.Widget{widget1})
	img, err := mgr.Composite()

	if err != nil {
		t.Fatalf("Composite() error = %v", err)
	}

	grayImg, ok := img.(*image.Gray)
	if !ok {
		t.Fatal("Composite() did not return *image.Gray")
	}

	// Verify transparent pixels (x < 64) remain canvas background (0)
	// and non-transparent pixels (x >= 64) are drawn (255)
	if grayImg.GrayAt(32, 20).Y != 0 {
		t.Errorf("Transparent pixel at x=32 should remain background (0), got %d", grayImg.GrayAt(32, 20).Y)
	}

	if grayImg.GrayAt(96, 20).Y != 255 {
		t.Errorf("Foreground pixel at x=96 should be 255, got %d", grayImg.GrayAt(96, 20).Y)
	}
}

func TestComposite_TransparentLayering(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:      128,
		Height:     40,
		Background: 50, // Gray background
	}

	// Bottom widget: opaque
	widget1 := newMockWidgetSimple("bottom", 0, 0, 128, 40, 0)
	widget1.style.Background = 0
	widget1Img := image.NewGray(image.Rect(0, 0, 128, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 128; x++ {
			widget1Img.Set(x, y, color.Gray{Y: 100})
		}
	}
	widget1.img = widget1Img

	// Top widget: transparent with holes
	widget2 := newMockWidgetSimple("top", 0, 0, 128, 40, 1)
	widget2.style.Background = -1 // Transparent
	widget2Img := image.NewGray(image.Rect(0, 0, 128, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 128; x++ {
			if x%2 == 0 {
				widget2Img.Set(x, y, color.Gray{Y: 0}) // Transparent pixels
			} else {
				widget2Img.Set(x, y, color.Gray{Y: 200}) // Visible pixels
			}
		}
	}
	widget2.img = widget2Img

	mgr := NewManager(displayCfg, []widget.Widget{widget1, widget2})
	img, err := mgr.Composite()

	if err != nil {
		t.Fatalf("Composite() error = %v", err)
	}

	grayImg, ok := img.(*image.Gray)
	if !ok {
		t.Fatal("Composite() did not return *image.Gray")
	}

	// Even x: should show bottom widget (100) through transparent top widget
	if grayImg.GrayAt(10, 20).Y != 100 {
		t.Errorf("Even x pixel should show bottom widget (100), got %d", grayImg.GrayAt(10, 20).Y)
	}

	// Odd x: should show top widget (200)
	if grayImg.GrayAt(11, 20).Y != 200 {
		t.Errorf("Odd x pixel should show top widget (200), got %d", grayImg.GrayAt(11, 20).Y)
	}
}

func TestComposite_PartiallyOffscreen(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:      128,
		Height:     40,
		Background: 0,
	}

	// Widget partially outside display bounds (should be clipped)
	widget1 := newMockWidgetSimple("offscreen", -32, -10, 64, 40, 0)
	widget1Img := image.NewGray(image.Rect(0, 0, 64, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 64; x++ {
			widget1Img.Set(x, y, color.Gray{Y: 200})
		}
	}
	widget1.img = widget1Img

	mgr := NewManager(displayCfg, []widget.Widget{widget1})
	img, err := mgr.Composite()

	if err != nil {
		t.Fatalf("Composite() error = %v", err)
	}

	// Should not crash, should clip widget to visible area
	if img == nil {
		t.Fatal("Composite() returned nil image")
	}
}

func TestCompositeWithTransparency_NonGrayImage(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:      128,
		Height:     40,
		Background: 0,
	}

	// Widget with non-Gray image (should be skipped by compositeWithTransparency)
	widget1 := newMockWidgetSimple("rgba", 0, 0, 64, 40, 0)
	widget1.style.Background = -1 // Transparent

	// Create RGBA image instead of Gray
	rgbaImg := image.NewRGBA(image.Rect(0, 0, 64, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 64; x++ {
			rgbaImg.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	widget1.img = rgbaImg

	mgr := NewManager(displayCfg, []widget.Widget{widget1})
	img, err := mgr.Composite()

	if err != nil {
		t.Fatalf("Composite() error = %v", err)
	}

	// Should not crash, should skip non-Gray image in compositeWithTransparency
	if img == nil {
		t.Fatal("Composite() returned nil image")
	}
}
