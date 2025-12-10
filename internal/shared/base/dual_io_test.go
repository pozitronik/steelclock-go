package base

import (
	"image"
	"sync"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/shared/util"
)

// mockWidgetBase implements WidgetBase interface for testing
type mockWidgetBase struct {
	position        config.PositionConfig
	style           config.StyleConfig
	backgroundColor uint8
}

func (m *mockWidgetBase) GetPosition() config.PositionConfig {
	return m.position
}

func (m *mockWidgetBase) GetStyle() config.StyleConfig {
	return m.style
}

func (m *mockWidgetBase) GetRenderBackgroundColor() uint8 {
	return m.backgroundColor
}

func (m *mockWidgetBase) CreateCanvas() *image.Gray {
	return bitmap.NewGrayscaleImage(m.position.W, m.position.H, m.backgroundColor)
}

func (m *mockWidgetBase) ApplyBorder(img *image.Gray) {
	if m.style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(m.style.Border))
	}
}

func newMockWidgetBase(w, h int) *mockWidgetBase {
	return &mockWidgetBase{
		position: config.PositionConfig{
			X: 0, Y: 0, W: w, H: h,
		},
		style:           config.StyleConfig{Border: -1, Background: 0},
		backgroundColor: 0,
	}
}

// TestNewBaseDualIOWidget tests widget creation
func TestNewBaseDualIOWidget(t *testing.T) {
	base := newMockWidgetBase(128, 40)
	converter := util.NewByteRateConverter("Mbps")
	renderer := render.NewDualMetricRenderer(
		render.DualBarConfig{Direction: "horizontal", PrimaryColor: 255, SecondaryColor: 200},
		render.DualGraphConfig{HistoryLen: 30, PrimaryFill: 255, SecondaryFill: 200},
		render.DualGaugeConfig{},
		render.TextConfig{HorizAlign: "center", VertAlign: "center"},
	)

	cfg := BaseDualIOConfig{
		Base:          base,
		DisplayMode:   render.DisplayModeText,
		Padding:       2,
		MaxSpeedBps:   1000000,
		Unit:          "Mbps",
		ShowUnit:      true,
		SupportsGauge: false,
		TextConfig: DualIOTextConfig{
			PrimaryPrefix:   "R",
			SecondaryPrefix: "W",
		},
		Converter:  converter,
		Renderer:   renderer,
		HistoryLen: 30,
	}

	widget := NewBaseDualIOWidget(cfg)

	if widget == nil {
		t.Fatal("NewBaseDualIOWidget() returned nil")
	}

	if widget.DisplayMode != render.DisplayModeText {
		t.Errorf("DisplayMode = %s, want text", widget.DisplayMode)
	}

	if widget.Padding != 2 {
		t.Errorf("Padding = %d, want 2", widget.Padding)
	}

	if widget.MaxSpeedBps != 1000000 {
		t.Errorf("MaxSpeedBps = %f, want 1000000", widget.MaxSpeedBps)
	}

	if widget.Unit != "Mbps" {
		t.Errorf("Unit = %s, want Mbps", widget.Unit)
	}

	if !widget.ShowUnit {
		t.Error("ShowUnit should be true")
	}

	if widget.SupportsGauge {
		t.Error("SupportsGauge should be false")
	}

	if widget.TextConfig.PrimaryPrefix != "R" {
		t.Errorf("PrimaryPrefix = %s, want R", widget.TextConfig.PrimaryPrefix)
	}

	if widget.TextConfig.SecondaryPrefix != "W" {
		t.Errorf("SecondaryPrefix = %s, want W", widget.TextConfig.SecondaryPrefix)
	}
}

// TestSetValues tests thread-safe value setting
func TestSetValues(t *testing.T) {
	widget := createTestWidget(render.DisplayModeText)

	widget.SetValues(100.5, 200.5)

	widget.Mu.RLock()
	primary := widget.PrimaryValue
	secondary := widget.SecondaryValue
	widget.Mu.RUnlock()

	if primary != 100.5 {
		t.Errorf("PrimaryValue = %f, want 100.5", primary)
	}

	if secondary != 200.5 {
		t.Errorf("SecondaryValue = %f, want 200.5", secondary)
	}
}

// TestAddToHistory tests history buffer operations
func TestAddToHistory(t *testing.T) {
	widget := createTestWidget(render.DisplayModeGraph)

	widget.AddToHistory(10.0, 20.0)
	widget.AddToHistory(30.0, 40.0)
	widget.AddToHistory(50.0, 60.0)

	widget.Mu.RLock()
	primaryLen := widget.PrimaryHistory.Len()
	secondaryLen := widget.SecondaryHistory.Len()
	widget.Mu.RUnlock()

	if primaryLen != 3 {
		t.Errorf("PrimaryHistory.Len() = %d, want 3", primaryLen)
	}

	if secondaryLen != 3 {
		t.Errorf("SecondaryHistory.Len() = %d, want 3", secondaryLen)
	}
}

// TestSetValuesAndHistory tests combined value and history update
func TestSetValuesAndHistory(t *testing.T) {
	widget := createTestWidget(render.DisplayModeGraph)

	// With history
	widget.SetValuesAndHistory(100.0, 200.0, true)

	widget.Mu.RLock()
	primary := widget.PrimaryValue
	secondary := widget.SecondaryValue
	histLen := widget.PrimaryHistory.Len()
	widget.Mu.RUnlock()

	if primary != 100.0 {
		t.Errorf("PrimaryValue = %f, want 100.0", primary)
	}

	if secondary != 200.0 {
		t.Errorf("SecondaryValue = %f, want 200.0", secondary)
	}

	if histLen != 1 {
		t.Errorf("PrimaryHistory.Len() = %d, want 1", histLen)
	}

	// Without history
	widget.SetValuesAndHistory(300.0, 400.0, false)

	widget.Mu.RLock()
	primary = widget.PrimaryValue
	histLen = widget.PrimaryHistory.Len()
	widget.Mu.RUnlock()

	if primary != 300.0 {
		t.Errorf("PrimaryValue = %f, want 300.0", primary)
	}

	if histLen != 1 {
		t.Errorf("PrimaryHistory.Len() should still be 1, got %d", histLen)
	}
}

// TestIsGraphMode tests display mode detection
func TestIsGraphMode(t *testing.T) {
	tests := []struct {
		mode render.DisplayMode
		want bool
	}{
		{render.DisplayModeText, false},
		{render.DisplayModeBar, false},
		{render.DisplayModeGraph, true},
		{render.DisplayModeGauge, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			widget := createTestWidget(tt.mode)
			got := widget.IsGraphMode()
			if got != tt.want {
				t.Errorf("IsGraphMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRender_TextMode tests rendering in text mode
func TestRender_TextMode(t *testing.T) {
	widget := createTestWidget(render.DisplayModeText)
	widget.SetValues(1000000, 2000000) // 1 MB/s, 2 MB/s

	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Render() image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

// TestRender_BarMode tests rendering in bar mode
func TestRender_BarMode(t *testing.T) {
	widget := createTestWidget(render.DisplayModeBar)
	widget.SetValues(500000, 1000000)

	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestRender_GraphMode tests rendering in graph mode
func TestRender_GraphMode(t *testing.T) {
	widget := createTestWidget(render.DisplayModeGraph)

	// Add enough history for graph rendering
	for i := 0; i < 10; i++ {
		widget.SetValuesAndHistory(float64(i*100000), float64(i*200000), true)
	}

	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestRender_GraphMode_InsufficientHistory tests graph with insufficient data
func TestRender_GraphMode_InsufficientHistory(t *testing.T) {
	widget := createTestWidget(render.DisplayModeGraph)

	// Only one data point (not enough for graph)
	widget.SetValuesAndHistory(100000, 200000, true)

	// Should still render without error
	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestRender_GaugeMode_Supported tests gauge mode when supported
func TestRender_GaugeMode_Supported(t *testing.T) {
	widget := createTestWidgetWithGauge(render.DisplayModeGauge, true)
	widget.SetValues(500000, 1000000)

	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestRender_GaugeMode_NotSupported tests gauge mode when not supported
func TestRender_GaugeMode_NotSupported(t *testing.T) {
	widget := createTestWidgetWithGauge(render.DisplayModeGauge, false)
	widget.SetValues(500000, 1000000)

	// Should render without error (just won't draw gauge)
	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestFormatDualIOValue tests value formatting
func TestFormatDualIOValue(t *testing.T) {
	tests := []struct {
		value float64
		want  string
	}{
		{0.0, "0.00"},
		{1.5, "1.50"},
		{9.99, "9.99"},
		{10.0, "10.0"},
		{50.5, "50.5"},
		{99.9, "99.9"},
		{100.0, "100"},
		{1000.0, "1000"},
		{12345.6, "12346"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatDualIOValue(tt.value)
			if got != tt.want {
				t.Errorf("FormatDualIOValue(%f) = %s, want %s", tt.value, got, tt.want)
			}
		})
	}
}

// TestClampValue tests value clamping
func TestClampValue(t *testing.T) {
	tests := []struct {
		value, min, max, want float64
	}{
		{50, 0, 100, 50},   // Within range
		{-10, 0, 100, 0},   // Below min
		{150, 0, 100, 100}, // Above max
		{0, 0, 100, 0},     // At min
		{100, 0, 100, 100}, // At max
		{0.5, 0, 1, 0.5},   // Float range
	}

	for _, tt := range tests {
		got := ClampValue(tt.value, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("ClampValue(%f, %f, %f) = %f, want %f",
				tt.value, tt.min, tt.max, got, tt.want)
		}
	}
}

// TestConcurrentAccess tests thread safety
func TestConcurrentAccess(t *testing.T) {
	widget := createTestWidget(render.DisplayModeGraph)

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writers
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			widget.SetValues(float64(i), float64(i*2))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			widget.SetValuesAndHistory(float64(i), float64(i*2), true)
		}
	}()

	// Concurrent readers (renderers)
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_, _ = widget.Render()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = widget.IsGraphMode()
		}
	}()

	wg.Wait()
	// Test passes if no race conditions or panics
}

// TestAutoScale tests auto-scaling behavior when MaxSpeedBps is -1
func TestAutoScale(t *testing.T) {
	widget := createTestWidgetAutoScale(render.DisplayModeBar)
	widget.SetValues(500, 1000) // Low values

	// Should render without error using auto-scale
	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() with auto-scale error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// Helper functions

func createTestWidget(mode render.DisplayMode) *BaseDualIOWidget {
	return createTestWidgetWithGauge(mode, false)
}

func createTestWidgetWithGauge(mode render.DisplayMode, supportsGauge bool) *BaseDualIOWidget {
	base := newMockWidgetBase(128, 40)
	converter := util.NewByteRateConverter("MB/s")
	renderer := render.NewDualMetricRenderer(
		render.DualBarConfig{Direction: "horizontal", PrimaryColor: 255, SecondaryColor: 200},
		render.DualGraphConfig{HistoryLen: 30, PrimaryFill: 255, SecondaryFill: 200},
		render.DualGaugeConfig{
			PrimaryArcColor:      200,
			PrimaryNeedleColor:   255,
			SecondaryArcColor:    150,
			SecondaryNeedleColor: 200,
		},
		render.TextConfig{HorizAlign: "center", VertAlign: "center"},
	)

	return NewBaseDualIOWidget(BaseDualIOConfig{
		Base:          base,
		DisplayMode:   mode,
		Padding:       2,
		MaxSpeedBps:   10000000, // 10 MB/s
		Unit:          "MB/s",
		ShowUnit:      false,
		SupportsGauge: supportsGauge,
		TextConfig: DualIOTextConfig{
			PrimaryPrefix:   "R",
			SecondaryPrefix: "W",
		},
		Converter:  converter,
		Renderer:   renderer,
		HistoryLen: 30,
	})
}

func createTestWidgetAutoScale(mode render.DisplayMode) *BaseDualIOWidget {
	base := newMockWidgetBase(128, 40)
	converter := util.NewByteRateConverter("MB/s")
	renderer := render.NewDualMetricRenderer(
		render.DualBarConfig{Direction: "horizontal", PrimaryColor: 255, SecondaryColor: 200},
		render.DualGraphConfig{HistoryLen: 30, PrimaryFill: 255, SecondaryFill: 200},
		render.DualGaugeConfig{},
		render.TextConfig{HorizAlign: "center", VertAlign: "center"},
	)

	return NewBaseDualIOWidget(BaseDualIOConfig{
		Base:          base,
		DisplayMode:   mode,
		Padding:       2,
		MaxSpeedBps:   -1, // Auto-scale
		Unit:          "auto",
		ShowUnit:      true,
		SupportsGauge: false,
		TextConfig: DualIOTextConfig{
			PrimaryPrefix:   "R",
			SecondaryPrefix: "W",
		},
		Converter:  converter,
		Renderer:   renderer,
		HistoryLen: 30,
	})
}
