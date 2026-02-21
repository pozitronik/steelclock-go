package render

import (
	"image"
	"testing"
)

func TestDrawBatteryShape(t *testing.T) {
	tests := []struct {
		name   string
		w, h   int
		config BatteryShapeConfig
	}{
		{
			name: "horizontal 0%",
			w:    64, h: 20,
			config: BatteryShapeConfig{Orientation: "horizontal", Percentage: 0, FillColor: 255, BorderColor: 255, Padding: 1},
		},
		{
			name: "horizontal 50%",
			w:    64, h: 20,
			config: BatteryShapeConfig{Orientation: "horizontal", Percentage: 50, FillColor: 200, BorderColor: 255, Padding: 1},
		},
		{
			name: "horizontal 100%",
			w:    64, h: 20,
			config: BatteryShapeConfig{Orientation: "horizontal", Percentage: 100, FillColor: 255, BorderColor: 255, Padding: 0},
		},
		{
			name: "vertical 0%",
			w:    20, h: 64,
			config: BatteryShapeConfig{Orientation: "vertical", Percentage: 0, FillColor: 255, BorderColor: 255, Padding: 1},
		},
		{
			name: "vertical 50%",
			w:    20, h: 64,
			config: BatteryShapeConfig{Orientation: "vertical", Percentage: 50, FillColor: 200, BorderColor: 255, Padding: 1},
		},
		{
			name: "vertical 100%",
			w:    20, h: 64,
			config: BatteryShapeConfig{Orientation: "vertical", Percentage: 100, FillColor: 255, BorderColor: 255, Padding: 0},
		},
		{
			name: "minimal horizontal dimensions",
			w:    10, h: 8,
			config: BatteryShapeConfig{Orientation: "horizontal", Percentage: 75, FillColor: 255, BorderColor: 255, Padding: 0},
		},
		{
			name: "minimal vertical dimensions",
			w:    8, h: 10,
			config: BatteryShapeConfig{Orientation: "vertical", Percentage: 75, FillColor: 255, BorderColor: 255, Padding: 0},
		},
		{
			name: "percentage clamped below 0",
			w:    64, h: 20,
			config: BatteryShapeConfig{Orientation: "horizontal", Percentage: -10, FillColor: 255, BorderColor: 255, Padding: 0},
		},
		{
			name: "percentage clamped above 100",
			w:    64, h: 20,
			config: BatteryShapeConfig{Orientation: "horizontal", Percentage: 150, FillColor: 255, BorderColor: 255, Padding: 0},
		},
		{
			name: "large padding",
			w:    64, h: 40,
			config: BatteryShapeConfig{Orientation: "horizontal", Percentage: 50, FillColor: 200, BorderColor: 255, Padding: 10},
		},
		{
			name: "zero fill color",
			w:    64, h: 20,
			config: BatteryShapeConfig{Orientation: "horizontal", Percentage: 50, FillColor: 0, BorderColor: 255, Padding: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewGray(image.Rect(0, 0, tt.w, tt.h))

			// Must not panic
			DrawBatteryShape(img, 0, 0, tt.w, tt.h, tt.config)

			if img.Bounds().Dx() != tt.w || img.Bounds().Dy() != tt.h {
				t.Errorf("image dimensions changed: got %dx%d, want %dx%d",
					img.Bounds().Dx(), img.Bounds().Dy(), tt.w, tt.h)
			}
		})
	}
}

func TestDrawBatteryShape_Offset(t *testing.T) {
	// Drawing at a non-zero offset should not panic
	img := image.NewGray(image.Rect(0, 0, 128, 40))
	DrawBatteryShape(img, 30, 5, 40, 20, BatteryShapeConfig{
		Orientation: "horizontal",
		Percentage:  80,
		FillColor:   200,
		BorderColor: 255,
		Padding:     1,
	})
}

func TestDrawBatteryShape_FillProportional(t *testing.T) {
	// At 0% there should be no fill pixels; at 100% there should be fill pixels.
	// We check by counting non-zero pixels in the interior region.
	makeImg := func(pct int) *image.Gray {
		img := image.NewGray(image.Rect(0, 0, 64, 20))
		DrawBatteryShape(img, 0, 0, 64, 20, BatteryShapeConfig{
			Orientation: "horizontal",
			Percentage:  pct,
			FillColor:   200,
			BorderColor: 255,
			Padding:     0,
		})
		return img
	}

	img0 := makeImg(0)
	img100 := makeImg(100)

	// Count pixels with the fill color value (200) in the interior
	count := func(img *image.Gray) int {
		n := 0
		for y := 3; y < 17; y++ {
			for x := 3; x < 50; x++ {
				if img.GrayAt(x, y).Y == 200 {
					n++
				}
			}
		}
		return n
	}

	fill0 := count(img0)
	fill100 := count(img100)

	if fill0 != 0 {
		t.Errorf("at 0%% expected 0 fill pixels in interior, got %d", fill0)
	}
	if fill100 == 0 {
		t.Error("at 100% expected some fill pixels in interior, got 0")
	}
}
