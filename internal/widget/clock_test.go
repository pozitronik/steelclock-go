package widget

import (
	"image"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewClockWidget(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_clock",
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

	widget, err := NewClockWidget(cfg)
	if err != nil {
		t.Fatalf("NewClockWidget() error = %v", err)
	}

	if widget == nil {
		t.Fatal("NewClockWidget() returned nil")
	}

	if widget.Name() != "test_clock" {
		t.Errorf("Name() = %s, want test_clock", widget.Name())
	}
}

func TestClockWidgetUpdate(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_clock",
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
			Format:          "15:04:05",
			FontSize:        12,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
		},
	}

	widget, err := NewClockWidget(cfg)
	if err != nil {
		t.Fatalf("NewClockWidget() error = %v", err)
	}

	// Update should populate currentTime
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	if widget.currentTime == "" {
		t.Error("Update() did not set currentTime")
	}
}

func TestClockWidgetRender(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_clock",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          true,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			Format:          "15:04",
			FontSize:        12,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
		},
	}

	widget, err := NewClockWidget(cfg)
	if err != nil {
		t.Fatalf("NewClockWidget() error = %v", err)
	}

	// Update before render
	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Render
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	if img.Bounds().Dx() != 128 {
		t.Errorf("image width = %d, want 128", img.Bounds().Dx())
	}

	if img.Bounds().Dy() != 40 {
		t.Errorf("image height = %d, want 40", img.Bounds().Dy())
	}
}

func TestClockWidgetRender_ClockFace(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_clock_face",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 80,
			H: 80,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          false,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "clock_face",
		},
	}

	widget, err := NewClockWidget(cfg)
	if err != nil {
		t.Fatalf("NewClockWidget() error = %v", err)
	}

	if widget == nil {
		t.Fatal("NewClockWidget() returned nil")
	}

	if widget.displayMode != "clock_face" {
		t.Errorf("displayMode = %s, want clock_face", widget.displayMode)
	}

	// Update before render
	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Render
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	if img.Bounds().Dx() != 80 {
		t.Errorf("image width = %d, want 80", img.Bounds().Dx())
	}

	if img.Bounds().Dy() != 80 {
		t.Errorf("image height = %d, want 80", img.Bounds().Dy())
	}

	// Check that some pixels are drawn (clock face should have content)
	grayImg := img.(*image.Gray)
	hasNonZeroPixels := false
	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			if grayImg.GrayAt(x, y).Y > 0 {
				hasNonZeroPixels = true
				break
			}
		}
		if hasNonZeroPixels {
			break
		}
	}

	if !hasNonZeroPixels {
		t.Error("Clock face rendered but has no visible pixels")
	}
}

func TestClockWidget_DefaultDisplayMode(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_default_mode",
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
			Format:   "15:04",
			FontSize: 12,
		},
	}

	widget, err := NewClockWidget(cfg)
	if err != nil {
		t.Fatalf("NewClockWidget() error = %v", err)
	}

	// Should default to "text" mode
	if widget.displayMode != "text" {
		t.Errorf("displayMode = %s, want text (default)", widget.displayMode)
	}
}

func TestClockWidgetRender_ClockFaceAlignment(t *testing.T) {
	tests := []struct {
		name          string
		horizAlign    string
		vertAlign     string
		width         int
		height        int
		padding       int
		checkPixelsFn func(*image.Gray, int, int) bool // Function to verify alignment
	}{
		{
			name:       "left-top alignment",
			horizAlign: "left",
			vertAlign:  "top",
			width:      100,
			height:     80,
			padding:    5,
			checkPixelsFn: func(img *image.Gray, w, h int) bool {
				// Clock should be in top-left quadrant
				// Check for pixels in left side
				hasLeft := false
				for y := 0; y < h/2; y++ {
					for x := 0; x < w/4; x++ {
						if img.GrayAt(x, y).Y > 0 {
							hasLeft = true
							break
						}
					}
				}
				return hasLeft
			},
		},
		{
			name:       "right-bottom alignment",
			horizAlign: "right",
			vertAlign:  "bottom",
			width:      100,
			height:     80,
			padding:    5,
			checkPixelsFn: func(img *image.Gray, w, h int) bool {
				// Clock should be in bottom-right quadrant
				// Check for pixels in right side
				hasRight := false
				for y := h / 2; y < h; y++ {
					for x := 3 * w / 4; x < w; x++ {
						if img.GrayAt(x, y).Y > 0 {
							hasRight = true
							break
						}
					}
				}
				return hasRight
			},
		},
		{
			name:       "center alignment",
			horizAlign: "center",
			vertAlign:  "center",
			width:      80,
			height:     80,
			padding:    0,
			checkPixelsFn: func(img *image.Gray, w, h int) bool {
				// Clock should be centered - check for pixels near center
				centerX, centerY := w/2, h/2
				hasCenter := false
				for dy := -5; dy <= 5; dy++ {
					for dx := -5; dx <= 5; dx++ {
						x, y := centerX+dx, centerY+dy
						if x >= 0 && x < w && y >= 0 && y < h {
							if img.GrayAt(x, y).Y > 0 {
								hasCenter = true
								break
							}
						}
					}
				}
				return hasCenter
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "clock",
				ID:      "test_clock_alignment",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: tt.width,
					H: tt.height,
				},
				Style: config.StyleConfig{
					BackgroundColor: 0,
					Border:          false,
					BorderColor:     255,
				},
				Properties: config.WidgetProperties{
					DisplayMode:     "clock_face",
					HorizontalAlign: tt.horizAlign,
					VerticalAlign:   tt.vertAlign,
					Padding:         tt.padding,
				},
			}

			widget, err := NewClockWidget(cfg)
			if err != nil {
				t.Fatalf("NewClockWidget() error = %v", err)
			}

			// Render
			img, err := widget.Render()
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			grayImg := img.(*image.Gray)

			// Verify alignment using the test-specific check function
			if !tt.checkPixelsFn(grayImg, tt.width, tt.height) {
				t.Errorf("Clock face alignment check failed for %s/%s", tt.horizAlign, tt.vertAlign)
			}

			// Verify widget has correct alignment properties
			if widget.horizAlign != tt.horizAlign {
				t.Errorf("horizAlign = %s, want %s", widget.horizAlign, tt.horizAlign)
			}
			if widget.vertAlign != tt.vertAlign {
				t.Errorf("vertAlign = %s, want %s", widget.vertAlign, tt.vertAlign)
			}
		})
	}
}

// TestClockWidget_ConcurrentAccess tests that concurrent calls to Update() and Render()
// do not cause data races on the currentTime string field.
// This test should be run with -race flag to detect concurrent access violations.
func TestClockWidget_ConcurrentAccess(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_clock_concurrent",
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
			Format:          "15:04:05",
			FontSize:        12,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
		},
	}

	widget, err := NewClockWidget(cfg)
	if err != nil {
		t.Fatalf("NewClockWidget() error = %v", err)
	}

	// Number of concurrent goroutines
	const numUpdaters = 20
	const numRenderers = 20
	const numIterations = 50

	done := make(chan bool, numUpdaters+numRenderers)
	errors := make(chan error, (numUpdaters+numRenderers)*numIterations)

	// Launch updater goroutines
	for i := 0; i < numUpdaters; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < numIterations; j++ {
				if err := widget.Update(); err != nil {
					errors <- err
				}
			}
		}(i)
	}

	// Launch renderer goroutines
	for i := 0; i < numRenderers; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < numIterations; j++ {
				_, err := widget.Render()
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numUpdaters+numRenderers; i++ {
		<-done
	}
	close(errors)

	// Check for any errors during execution
	var errCount int
	for err := range errors {
		t.Errorf("Error during concurrent access: %v", err)
		errCount++
		if errCount > 5 {
			t.Log("(truncating error list...)")
			break
		}
	}

	// Note: The race detector will catch concurrent string access
	// even if no errors are returned. Run with: go test -race
	t.Log("Concurrent access test completed. Run with -race flag to detect data races.")
}
