package clock

import (
	"image"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNew(t *testing.T) {
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
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Text: &config.TextConfig{
			Format: "15:04",
			Size:   12,
			Align:  &config.AlignConfig{H: "center", V: "center"},
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if widget == nil {
		t.Fatal("New() returned nil")
	}

	if widget.Name() != "test_clock" {
		t.Errorf("Name() = %s, want test_clock", widget.Name())
	}
}

func TestWidget_Update(t *testing.T) {
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
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Text: &config.TextConfig{
			Format: "15:04:05",
			Size:   12,
			Align:  &config.AlignConfig{H: "center", V: "center"},
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Update should populate currentTime
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	if widget.currentTime.IsZero() {
		t.Error("Update() did not set currentTime")
	}
}

func TestWidget_Render(t *testing.T) {
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
		Style: &config.StyleConfig{
			Background: 0,
			Border:     255,
		},
		Text: &config.TextConfig{
			Format: "15:04",
			Size:   12,
			Align:  &config.AlignConfig{H: "center", V: "center"},
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
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

func TestWidget_Render_ClockFace(t *testing.T) {
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
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Mode: "clock_face",
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if widget == nil {
		t.Fatal("New() returned nil")
	}

	// "clock_face" is mapped to "analog" internally
	if widget.displayMode != "analog" {
		t.Errorf("displayMode = %s, want analog (mapped from clock_face)", widget.displayMode)
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

func TestWidget_DefaultDisplayMode(t *testing.T) {
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
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Text: &config.TextConfig{
			Format: "15:04",
			Size:   12,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Should default to "text" mode
	if widget.displayMode != "text" {
		t.Errorf("displayMode = %s, want text (default)", widget.displayMode)
	}
}

func TestWidget_Render_ClockFaceAlignment(t *testing.T) {
	tests := []struct {
		name          string
		horizAlign    config.HAlign
		vertAlign     config.VAlign
		width         int
		height        int
		padding       int
		checkPixelsFn func(*image.Gray, int, int) bool // Function to verify alignment
	}{
		{
			name:       "left-top alignment",
			horizAlign: config.AlignLeft,
			vertAlign:  config.AlignTop,
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
			horizAlign: config.AlignRight,
			vertAlign:  config.AlignBottom,
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
			horizAlign: config.AlignCenter,
			vertAlign:  config.AlignMiddle,
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
				Style: &config.StyleConfig{
					Background: 0,
					Border:     -1,
					Padding:    tt.padding,
				},
				Mode: "clock_face",
				Text: &config.TextConfig{
					Align: &config.AlignConfig{H: tt.horizAlign, V: tt.vertAlign},
				},
			}

			widget, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
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
		})
	}
}

func TestWidget_BinaryMode(t *testing.T) {
	tests := []struct {
		name         string
		binaryConfig *config.BinaryClockConfig
	}{
		{
			name:         "default binary config",
			binaryConfig: nil,
		},
		{
			name: "custom binary format bcd",
			binaryConfig: &config.BinaryClockConfig{
				Format: "bcd",
				Style:  "dots",
				Layout: "horizontal",
			},
		},
		{
			name: "custom binary format true",
			binaryConfig: &config.BinaryClockConfig{
				Format:     "true",
				Style:      "bars",
				Layout:     "vertical",
				ShowLabels: true,
				ShowHint:   true,
				DotSize:    4,
				DotSpacing: 2,
				DotStyle:   "square",
				OnColor:    intPtr(255),
				OffColor:   intPtr(50),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "clock",
				ID:      "test_binary_clock",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Style: &config.StyleConfig{
					Background: 0,
					Border:     -1,
				},
				Mode:   "binary",
				Binary: tt.binaryConfig,
			}

			widget, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			if widget.displayMode != "binary" {
				t.Errorf("displayMode = %s, want binary", widget.displayMode)
			}

			// Render should succeed
			img, err := widget.Render()
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			if img == nil {
				t.Fatal("Render() returned nil image")
			}

			bounds := img.Bounds()
			if bounds.Dx() != 128 || bounds.Dy() != 40 {
				t.Errorf("image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
			}
		})
	}
}

func TestWidget_SegmentMode(t *testing.T) {
	tests := []struct {
		name          string
		segmentConfig *config.SegmentClockConfig
	}{
		{
			name:          "default segment config",
			segmentConfig: nil,
		},
		{
			name: "custom segment format",
			segmentConfig: &config.SegmentClockConfig{
				Format: "15:04",
			},
		},
		{
			name: "full custom segment config",
			segmentConfig: &config.SegmentClockConfig{
				Format:           "15:04:05",
				DigitHeight:      20,
				SegmentThickness: 3,
				SegmentStyle:     "rounded",
				DigitSpacing:     2,
				ColonStyle:       "dots",
				ColonBlink:       config.BoolPtr(true),
				OnColor:          intPtr(255),
				OffColor:         intPtr(30),
				Flip: &config.FlipEffectConfig{
					Style: "slide",
					Speed: 100,
				},
			},
		},
		{
			name: "segment with no flip config",
			segmentConfig: &config.SegmentClockConfig{
				Format:       "15:04",
				DigitHeight:  16,
				DigitSpacing: 1,
				ColonBlink:   config.BoolPtr(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "clock",
				ID:      "test_segment_clock",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Style: &config.StyleConfig{
					Background: 0,
					Border:     -1,
				},
				Mode:    "segment",
				Segment: tt.segmentConfig,
			}

			widget, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			if widget.displayMode != "segment" {
				t.Errorf("displayMode = %s, want segment", widget.displayMode)
			}

			// Render should succeed
			img, err := widget.Render()
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			if img == nil {
				t.Fatal("Render() returned nil image")
			}

			bounds := img.Bounds()
			if bounds.Dx() != 128 || bounds.Dy() != 40 {
				t.Errorf("image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
			}
		})
	}
}

func TestWidget_NeedsUpdate(t *testing.T) {
	// Text mode clock should return false for NeedsUpdate
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_clock_needs_update",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
		Text: &config.TextConfig{
			Format: "15:04:05",
			Size:   12,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// NeedsUpdate should return a boolean without error
	needsUpdate := widget.NeedsUpdate()
	// Text renderer typically returns false
	t.Logf("Text mode NeedsUpdate() = %v", needsUpdate)

	// Test with segment mode (has animations)
	segmentCfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_segment_needs_update",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "segment",
		Segment: &config.SegmentClockConfig{
			ColonBlink: config.BoolPtr(true),
		},
	}

	segmentWidget, err := New(segmentCfg)
	if err != nil {
		t.Fatalf("New(segment) error = %v", err)
	}

	segmentNeedsUpdate := segmentWidget.NeedsUpdate()
	t.Logf("Segment mode NeedsUpdate() = %v", segmentNeedsUpdate)
}

// TestWidget_ConcurrentAccess tests that concurrent calls to Update() and Render()
// do not cause data races on the currentTime string field.
// This test should be run with -race flag to detect concurrent access violations.
func TestWidget_ConcurrentAccess(t *testing.T) {
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
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Text: &config.TextConfig{
			Format: "15:04:05",
			Size:   12,
			Align:  &config.AlignConfig{H: "center", V: "center"},
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
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

// Helper function for tests
func intPtr(i int) *int {
	return &i
}

func TestConvert24to12(t *testing.T) {
	tests := []struct {
		hour24   int
		wantHour int
		wantIsPM bool
	}{
		{0, 12, false},  // midnight
		{1, 1, false},   // 1 AM
		{11, 11, false}, // 11 AM
		{12, 12, true},  // noon
		{13, 1, true},   // 1 PM
		{23, 11, true},  // 11 PM
		{6, 6, false},   // 6 AM
		{18, 6, true},   // 6 PM
	}

	for _, tt := range tests {
		t.Run("hour_"+string(rune('0'+tt.hour24/10))+string(rune('0'+tt.hour24%10)), func(t *testing.T) {
			gotHour, gotIsPM := convert24to12(tt.hour24)
			if gotHour != tt.wantHour || gotIsPM != tt.wantIsPM {
				t.Errorf("convert24to12(%d) = (%d, %v), want (%d, %v)",
					tt.hour24, gotHour, gotIsPM, tt.wantHour, tt.wantIsPM)
			}
		})
	}
}

func TestWidget_12HourMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		use12h   bool
		showAmPm bool
	}{
		{"text 24h", "text", false, false},
		{"text 12h no ampm", "text", true, false},
		{"text 12h with ampm", "text", true, true},
		{"segment 24h", "segment", false, false},
		{"segment 12h no ampm", "segment", true, false},
		{"segment 12h with ampm dot", "segment", true, true},
		{"binary 24h", "binary", false, false},
		{"binary 12h no ampm", "binary", true, false},
		{"binary 12h with ampm", "binary", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "clock",
				ID:      "test_12h_clock",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Style: &config.StyleConfig{
					Background: 0,
					Border:     -1,
				},
				Mode: tt.mode,
			}

			// Set mode-specific config
			switch tt.mode {
			case "text":
				cfg.Text = &config.TextConfig{
					Format:   "15:04:05",
					Size:     12,
					Use12h:   tt.use12h,
					ShowAmPm: tt.showAmPm,
				}
			case "segment":
				cfg.Segment = &config.SegmentClockConfig{
					Format:   "%H:%M",
					Use12h:   tt.use12h,
					ShowAmPm: tt.showAmPm,
				}
			case "binary":
				cfg.Binary = &config.BinaryClockConfig{
					Format:   "%H:%M:%S",
					Use12h:   tt.use12h,
					ShowAmPm: tt.showAmPm,
				}
			}

			widget, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			// Render should succeed
			img, err := widget.Render()
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			if img == nil {
				t.Fatal("Render() returned nil image")
			}

			bounds := img.Bounds()
			if bounds.Dx() != 128 || bounds.Dy() != 40 {
				t.Errorf("image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
			}
		})
	}
}

func TestWidget_SegmentAmPmStyles(t *testing.T) {
	tests := []struct {
		name      string
		ampmStyle string
	}{
		{"dot style", "dot"},
		{"text style", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "clock",
				ID:      "test_segment_ampm_style",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Style: &config.StyleConfig{
					Background: 0,
					Border:     -1,
				},
				Mode: "segment",
				Segment: &config.SegmentClockConfig{
					Format:    "%H:%M",
					Use12h:    true,
					ShowAmPm:  true,
					AmPmStyle: tt.ampmStyle,
				},
			}

			widget, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			img, err := widget.Render()
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			if img == nil {
				t.Fatal("Render() returned nil image")
			}
		})
	}
}

func TestConvertStrftimeToGo(t *testing.T) {
	tests := []struct {
		strftime string
		want     string
	}{
		// Exact matches
		{"%H:%M:%S", "15:04:05"},
		{"%H:%M", "15:04"},
		{"%I:%M:%S", "3:04:05"},
		{"%I:%M", "3:04"},
		{"%Y-%m-%d", "2006-01-02"},
		{"%d.%m.%Y", "02.01.2006"},
		// Token replacements
		{"%H", "15"},
		{"%I", "3"},
		{"%M", "04"},
		{"%S", "05"},
		{"%p", "PM"},
		// Mixed formats
		{"%I:%M %p", "3:04 PM"},
		{"%H:%M:%S on %Y-%m-%d", "15:04:05 on 2006-01-02"},
		// Go format passthrough
		{"15:04:05", "15:04:05"},
		{"3:04 PM", "3:04 PM"},
	}

	for _, tt := range tests {
		t.Run(tt.strftime, func(t *testing.T) {
			got := convertStrftimeToGo(tt.strftime)
			if got != tt.want {
				t.Errorf("convertStrftimeToGo(%q) = %q, want %q", tt.strftime, got, tt.want)
			}
		})
	}
}
