package widget

import (
	"image"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewErrorWidget(t *testing.T) {
	widget := NewErrorWidget(128, 40, "TEST")

	if widget == nil {
		t.Fatal("NewErrorWidget() returned nil")
	}

	if widget.Name() != "error_display" {
		t.Errorf("Name() = %s, want error_display", widget.Name())
	}

	if widget.message != "TEST" {
		t.Errorf("message = %s, want TEST", widget.message)
	}

	if !widget.flashState {
		t.Error("flashState should start as true")
	}
}

func TestErrorWidget_Update(t *testing.T) {
	widget := NewErrorWidget(128, 40, "CONFIG")

	// Initial state
	initialState := widget.flashState

	// Update immediately - should not change yet
	_ = widget.Update()
	if widget.flashState != initialState {
		t.Error("flashState should not change immediately")
	}

	// Wait for flash period
	time.Sleep(550 * time.Millisecond)
	_ = widget.Update()

	if widget.flashState == initialState {
		t.Error("flashState should toggle after flash period")
	}
}

func TestErrorWidget_Render_FlashOn(t *testing.T) {
	widget := NewErrorWidget(128, 40, "CONFIG")
	widget.flashState = true

	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	grayImg := img.(*image.Gray)

	// When flash is on, should have visible pixels
	hasPixels := false
	for y := 0; y < 40; y++ {
		for x := 0; x < 128; x++ {
			if grayImg.GrayAt(x, y).Y > 0 {
				hasPixels = true
				break
			}
		}
		if hasPixels {
			break
		}
	}

	if !hasPixels {
		t.Error("Render() with flash on should have visible pixels")
	}
}

func TestErrorWidget_Render_FlashOff(t *testing.T) {
	widget := NewErrorWidget(128, 40, "CONFIG")
	widget.flashState = false

	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	grayImg := img.(*image.Gray)

	// When flash is off, should be all black (background)
	hasNonZeroPixels := false
	for y := 0; y < 40; y++ {
		for x := 0; x < 128; x++ {
			if grayImg.GrayAt(x, y).Y > 0 {
				hasNonZeroPixels = true
				break
			}
		}
		if hasNonZeroPixels {
			break
		}
	}

	if hasNonZeroPixels {
		t.Error("Render() with flash off should have no visible pixels")
	}
}

func TestErrorWidget_Messages(t *testing.T) {
	messages := []string{
		"CONFIG",
		"NO WIDGETS",
		"ERROR",
	}

	for _, msg := range messages {
		t.Run(msg, func(t *testing.T) {
			widget := NewErrorWidget(128, 40, msg)
			widget.flashState = true

			img, err := widget.Render()
			if err != nil {
				t.Fatalf("Render() error = %v for message %s", err, msg)
			}

			if img == nil {
				t.Fatalf("Render() returned nil image for message %s", msg)
			}

			// Verify image has content
			grayImg := img.(*image.Gray)
			hasContent := false
			for y := 0; y < img.Bounds().Dy(); y++ {
				for x := 0; x < img.Bounds().Dx(); x++ {
					if grayImg.GrayAt(x, y).Y > 0 {
						hasContent = true
						break
					}
				}
				if hasContent {
					break
				}
			}

			if !hasContent {
				t.Errorf("Render() for message %s has no visible content", msg)
			}
		})
	}
}

func TestErrorWidget_DifferentSizes(t *testing.T) {
	sizes := []struct {
		width  int
		height int
	}{
		{128, 40},
		{64, 32},
		{256, 64},
	}

	for _, size := range sizes {
		t.Run("size", func(t *testing.T) {
			widget := NewErrorWidget(size.width, size.height, "TEST")

			img, err := widget.Render()
			if err != nil {
				t.Fatalf("Render() error = %v for size %dx%d", err, size.width, size.height)
			}

			if img.Bounds().Dx() != size.width {
				t.Errorf("image width = %d, want %d", img.Bounds().Dx(), size.width)
			}

			if img.Bounds().Dy() != size.height {
				t.Errorf("image height = %d, want %d", img.Bounds().Dy(), size.height)
			}
		})
	}
}

// TestErrorWidget_AllCharacters tests rendering of various characters
func TestErrorWidget_AllCharacters(t *testing.T) {
	messages := []string{
		"ABCDEFGHI",
		"NO WIDGETS",
		"CONFIG FAILED",
		"0123456789",
		"TESTING",
	}

	for _, msg := range messages {
		t.Run(msg, func(t *testing.T) {
			widget := NewErrorWidget(128, 40, msg)
			widget.flashState = true

			img, err := widget.Render()
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			if img == nil {
				t.Fatal("Render() returned nil")
			}
		})
	}
}

// TestErrorWidget_SmallSize tests with very small dimensions
func TestErrorWidget_SmallSize(t *testing.T) {
	widget := NewErrorWidget(10, 10, "X")
	widget.flashState = true

	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil for small size")
	}
}

// TestErrorWidget_EmptyMessage tests with empty message
func TestErrorWidget_EmptyMessage(t *testing.T) {
	widget := NewErrorWidget(128, 40, "")
	widget.flashState = true

	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil for empty message")
	}
}

// TestErrorWidget_LongMessage tests with very long message
func TestErrorWidget_LongMessage(t *testing.T) {
	widget := NewErrorWidget(128, 40, "VERYLONGMESSAGETHATEXCEEDSWIDTH")
	widget.flashState = true

	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil for long message")
	}
}

// TestErrorWidget_UpdateCycle tests multiple update cycles
func TestErrorWidget_UpdateCycle(t *testing.T) {
	widget := NewErrorWidget(128, 40, "TEST")
	initialState := widget.flashState

	// Update multiple times quickly - should not toggle
	for i := 0; i < 5; i++ {
		_ = widget.Update()
	}

	if widget.flashState != initialState {
		t.Error("flashState should not toggle on rapid updates")
	}
}

// TestErrorWidget_GetPosition tests position getter
func TestErrorWidget_GetPosition(t *testing.T) {
	widget := NewErrorWidget(128, 40, "TEST")
	pos := widget.GetPosition()

	if pos.W != 128 {
		t.Errorf("GetPosition().W = %d, want 128", pos.W)
	}

	if pos.H != 40 {
		t.Errorf("GetPosition().H = %d, want 40", pos.H)
	}
}

// TestErrorWidget_GetUpdateInterval tests update interval
func TestErrorWidget_GetUpdateInterval(t *testing.T) {
	widget := NewErrorWidget(128, 40, "TEST")
	interval := widget.GetUpdateInterval()

	// Default update interval from BaseWidget is 1 second
	if interval != 1*time.Second {
		t.Errorf("GetUpdateInterval() = %v, want 1s", interval)
	}
}

// TestErrorWidget_GetStyle tests style getter
func TestErrorWidget_GetStyle(t *testing.T) {
	widget := NewErrorWidget(128, 40, "TEST")
	style := widget.GetStyle()

	if style.Background != 0 {
		t.Errorf("GetStyle().Background = %d, want 0", style.Background)
	}
}

// TestErrorWidget_TinySize tests with dimensions too small for any text layout
// This should trigger the renderIconOnly fallback path
func TestErrorWidget_TinySize(t *testing.T) {
	// Test with dimensions that are too small for any text layout but can fit smallest icon
	sizes := []struct {
		width  int
		height int
	}{
		{20, 15}, // Very small but can fit 12x12 icon
		{15, 15}, // Can fit 12x12 icon
		{12, 12}, // Exactly 12x12 icon size
		{8, 8},   // Too small for any icon
	}

	for _, size := range sizes {
		t.Run("tiny", func(t *testing.T) {
			widget := NewErrorWidget(size.width, size.height, "VERYLONGMESSAGETHATCANTFIT")
			widget.flashState = true

			img, err := widget.Render()
			if err != nil {
				t.Fatalf("Render() error = %v for tiny size %dx%d", err, size.width, size.height)
			}

			if img == nil {
				t.Fatalf("Render() returned nil for tiny size %dx%d", size.width, size.height)
			}

			bounds := img.Bounds()
			if bounds.Dx() != size.width || bounds.Dy() != size.height {
				t.Errorf("image size = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), size.width, size.height)
			}
		})
	}
}

// TestErrorWidget_NewErrorWidgetWithConfig tests widget creation with config
func TestErrorWidget_NewErrorWidgetWithConfig(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "original_widget",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 10, Y: 20, W: 100, H: 50,
		},
	}

	widget := NewErrorWidgetWithConfig(cfg, "CONFIG")

	if widget == nil {
		t.Fatal("NewErrorWidgetWithConfig() returned nil")
	}

	if widget.message != "CONFIG" {
		t.Errorf("message = %s, want CONFIG", widget.message)
	}

	// ID should be original_widget_error
	if widget.Name() != "original_widget_error" {
		t.Errorf("Name() = %s, want original_widget_error", widget.Name())
	}

	// Should inherit position from original config
	pos := widget.GetPosition()
	if pos.X != 10 || pos.Y != 20 {
		t.Errorf("Position = (%d, %d), want (10, 20)", pos.X, pos.Y)
	}
}

// TestErrorWidget_NewErrorWidgetWithConfig_NilStyle tests creation with nil style
func TestErrorWidget_NewErrorWidgetWithConfig_NilStyle(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "no_style_widget",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Style: nil, // Nil style should be handled
	}

	widget := NewErrorWidgetWithConfig(cfg, "ERROR")

	if widget == nil {
		t.Fatal("NewErrorWidgetWithConfig() returned nil for nil style")
	}

	// Should create style with background=0 and border=-1
	style := widget.GetStyle()
	if style.Background != 0 {
		t.Errorf("Style.Background = %d, want 0", style.Background)
	}
}
