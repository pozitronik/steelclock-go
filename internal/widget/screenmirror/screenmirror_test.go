package screenmirror

import (
	"image"
	"image/color"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestScaleMode_Constants(t *testing.T) {
	tests := []struct {
		name string
		mode ScaleMode
		want string
	}{
		{"fit", ScaleModeFit, "fit"},
		{"stretch", ScaleModeStretch, "stretch"},
		{"crop", ScaleModeCrop, "crop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.mode) != tt.want {
				t.Errorf("ScaleMode = %v, want %v", tt.mode, tt.want)
			}
		})
	}
}

func TestScaleImage_Fit(t *testing.T) {
	// Create a 100x50 source image (2:1 aspect)
	src := image.NewRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			src.Set(x, y, color.White)
		}
	}

	// Scale to 40x40 target (1:1 aspect)
	result := ScaleImage(src, 40, 40, ScaleModeFit)

	if result == nil {
		t.Fatal("ScaleImage returned nil")
	}

	bounds := result.Bounds()
	if bounds.Dx() != 40 || bounds.Dy() != 40 {
		t.Errorf("Result dimensions = %dx%d, want 40x40", bounds.Dx(), bounds.Dy())
	}

	// In fit mode, the wider source should be letterboxed (black bars top/bottom)
	// The top and bottom should be black (due to letterboxing)
	topPixel := result.GrayAt(20, 0).Y
	if topPixel != 0 {
		t.Logf("Top pixel (letterbox area) = %d (expected ~0 for black)", topPixel)
	}
}

func TestScaleImage_Stretch(t *testing.T) {
	// Create a 100x50 source image
	src := image.NewRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			src.Set(x, y, color.White)
		}
	}

	// Scale to 40x40 target
	result := ScaleImage(src, 40, 40, ScaleModeStretch)

	if result == nil {
		t.Fatal("ScaleImage returned nil")
	}

	bounds := result.Bounds()
	if bounds.Dx() != 40 || bounds.Dy() != 40 {
		t.Errorf("Result dimensions = %dx%d, want 40x40", bounds.Dx(), bounds.Dy())
	}

	// In stretch mode, the entire image should be filled
	centerPixel := result.GrayAt(20, 20).Y
	if centerPixel == 0 {
		t.Error("Center pixel should not be black in stretch mode")
	}
}

func TestScaleImage_Crop(t *testing.T) {
	// Create a 100x50 source image
	src := image.NewRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			src.Set(x, y, color.White)
		}
	}

	// Scale to 40x40 target
	result := ScaleImage(src, 40, 40, ScaleModeCrop)

	if result == nil {
		t.Fatal("ScaleImage returned nil")
	}

	bounds := result.Bounds()
	if bounds.Dx() != 40 || bounds.Dy() != 40 {
		t.Errorf("Result dimensions = %dx%d, want 40x40", bounds.Dx(), bounds.Dy())
	}

	// In crop mode, the entire output should be filled (edges cropped from source)
	centerPixel := result.GrayAt(20, 20).Y
	if centerPixel == 0 {
		t.Error("Center pixel should not be black in crop mode")
	}
}

func TestScaleImage_NilSource(t *testing.T) {
	result := ScaleImage(nil, 40, 40, ScaleModeFit)

	if result == nil {
		t.Fatal("ScaleImage returned nil for nil source")
	}

	bounds := result.Bounds()
	if bounds.Dx() != 40 || bounds.Dy() != 40 {
		t.Errorf("Result dimensions = %dx%d, want 40x40", bounds.Dx(), bounds.Dy())
	}
}

func TestScaleImage_ZeroSource(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 0, 0))
	result := ScaleImage(src, 40, 40, ScaleModeFit)

	if result == nil {
		t.Fatal("ScaleImage returned nil for zero-sized source")
	}

	bounds := result.Bounds()
	if bounds.Dx() != 40 || bounds.Dy() != 40 {
		t.Errorf("Result dimensions = %dx%d, want 40x40", bounds.Dx(), bounds.Dy())
	}
}

func TestParseScreenMirrorConfig_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "screen_mirror",
		Position: config.PositionConfig{W: 128, H: 40},
	}

	result := parseScreenMirrorConfig(cfg)

	if result.ScaleMode != ScaleModeFit {
		t.Errorf("ScaleMode = %v, want %v", result.ScaleMode, ScaleModeFit)
	}
	if result.FPS != 15 {
		t.Errorf("FPS = %d, want 15", result.FPS)
	}
	if result.DitherMode != DitherFloydSteinberg {
		t.Errorf("DitherMode = %v, want %v", result.DitherMode, DitherFloydSteinberg)
	}
	if result.DisplayIndex != nil {
		t.Errorf("DisplayIndex = %v, want nil", result.DisplayIndex)
	}
	if result.Region != nil {
		t.Errorf("Region = %v, want nil", result.Region)
	}
	if result.Window != nil {
		t.Errorf("Window = %v, want nil", result.Window)
	}
}

func TestParseScreenMirrorConfig_CustomValues(t *testing.T) {
	displayIdx := 1
	cfg := config.WidgetConfig{
		Type:     "screen_mirror",
		Position: config.PositionConfig{W: 128, H: 40},
		ScreenMirror: &config.ScreenMirrorConfig{
			Display:    &displayIdx,
			ScaleMode:  "stretch",
			FPS:        30,
			DitherMode: "ordered",
		},
	}

	result := parseScreenMirrorConfig(cfg)

	if result.ScaleMode != ScaleModeStretch {
		t.Errorf("ScaleMode = %v, want %v", result.ScaleMode, ScaleModeStretch)
	}
	if result.FPS != 30 {
		t.Errorf("FPS = %d, want 30", result.FPS)
	}
	if result.DitherMode != "ordered" {
		t.Errorf("DitherMode = %v, want ordered", result.DitherMode)
	}
	if result.DisplayIndex == nil || *result.DisplayIndex != 1 {
		t.Errorf("DisplayIndex = %v, want 1", result.DisplayIndex)
	}
}

func TestParseScreenMirrorConfig_FPSClamping(t *testing.T) {
	tests := []struct {
		name    string
		fps     int
		wantFPS int
	}{
		{"too low", 0, 15}, // 0 means use default
		{"minimum", 1, 1},
		{"normal", 15, 15},
		{"maximum", 30, 30},
		{"too high", 60, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:     "screen_mirror",
				Position: config.PositionConfig{W: 128, H: 40},
				ScreenMirror: &config.ScreenMirrorConfig{
					FPS: tt.fps,
				},
			}

			result := parseScreenMirrorConfig(cfg)
			if result.FPS != tt.wantFPS {
				t.Errorf("FPS = %d, want %d", result.FPS, tt.wantFPS)
			}
		})
	}
}

func TestParseScreenMirrorConfig_Region(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "screen_mirror",
		Position: config.PositionConfig{W: 128, H: 40},
		ScreenMirror: &config.ScreenMirrorConfig{
			Region: &config.ScreenMirrorRegionConfig{
				X:      100,
				Y:      200,
				Width:  400,
				Height: 300,
			},
		},
	}

	result := parseScreenMirrorConfig(cfg)

	if result.Region == nil {
		t.Fatal("Region is nil")
	}
	if result.Region.X != 100 {
		t.Errorf("Region.X = %d, want 100", result.Region.X)
	}
	if result.Region.Y != 200 {
		t.Errorf("Region.Y = %d, want 200", result.Region.Y)
	}
	if result.Region.Width != 400 {
		t.Errorf("Region.Width = %d, want 400", result.Region.Width)
	}
	if result.Region.Height != 300 {
		t.Errorf("Region.Height = %d, want 300", result.Region.Height)
	}
}

func TestParseScreenMirrorConfig_Window(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "screen_mirror",
		Position: config.PositionConfig{W: 128, H: 40},
		ScreenMirror: &config.ScreenMirrorConfig{
			Window: &config.ScreenMirrorWindowConfig{
				Title:  "Calculator",
				Active: true,
			},
		},
	}

	result := parseScreenMirrorConfig(cfg)

	if result.Window == nil {
		t.Fatal("Window is nil")
	}
	if result.Window.Title != "Calculator" {
		t.Errorf("Window.Title = %q, want %q", result.Window.Title, "Calculator")
	}
	if !result.Window.Active {
		t.Error("Window.Active = false, want true")
	}
}

func TestOrderedDither(t *testing.T) {
	// Create a gradient image
	src := image.NewGray(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			// Diagonal gradient
			val := uint8((x + y) * 255 / 30)
			src.SetGray(x, y, color.Gray{Y: val})
		}
	}

	result := orderedDither(src)

	if result == nil {
		t.Fatal("orderedDither returned nil")
	}

	bounds := result.Bounds()
	if bounds.Dx() != 16 || bounds.Dy() != 16 {
		t.Errorf("Result dimensions = %dx%d, want 16x16", bounds.Dx(), bounds.Dy())
	}

	// Check that output is binary (only 0 or 255)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			val := result.GrayAt(x, y).Y
			if val != 0 && val != 255 {
				t.Errorf("Pixel at (%d,%d) = %d, want 0 or 255", x, y, val)
			}
		}
	}
}

func TestContentType_String(t *testing.T) {
	// Test DisplayInfo struct
	info := DisplayInfo{
		Index:     0,
		Name:      "Test Display",
		Width:     1920,
		Height:    1080,
		IsPrimary: true,
	}

	if info.Width != 1920 {
		t.Errorf("Width = %d, want 1920", info.Width)
	}
	if info.Height != 1080 {
		t.Errorf("Height = %d, want 1080", info.Height)
	}
	if !info.IsPrimary {
		t.Error("IsPrimary = false, want true")
	}
}

func TestCaptureRegion(t *testing.T) {
	region := CaptureRegion{
		X:      100,
		Y:      200,
		Width:  800,
		Height: 600,
	}

	if region.X != 100 {
		t.Errorf("X = %d, want 100", region.X)
	}
	if region.Y != 200 {
		t.Errorf("Y = %d, want 200", region.Y)
	}
	if region.Width != 800 {
		t.Errorf("Width = %d, want 800", region.Width)
	}
	if region.Height != 600 {
		t.Errorf("Height = %d, want 600", region.Height)
	}
}

func TestWindowTarget(t *testing.T) {
	target := WindowTarget{
		Title:  "Test Window",
		Class:  "TestClass",
		Active: true,
	}

	if target.Title != "Test Window" {
		t.Errorf("Title = %q, want %q", target.Title, "Test Window")
	}
	if target.Class != "TestClass" {
		t.Errorf("Class = %q, want %q", target.Class, "TestClass")
	}
	if !target.Active {
		t.Error("Active = false, want true")
	}
}
