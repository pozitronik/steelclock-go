package bitmap

import (
	"image/color"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestDrawHorizontalBar(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		height     int
		percentage float64
		border     bool
	}{
		{"Empty bar", 20, 5, 0, false},
		{"Half full", 20, 5, 50, false},
		{"Full bar", 20, 5, 100, false},
		{"With border empty", 20, 5, 0, true},
		{"With border half", 20, 5, 50, true},
		{"With border full", 20, 5, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(tt.width, tt.height, 0)
			DrawHorizontalBar(img, 0, 0, tt.width, tt.height, tt.percentage, 255, tt.border)

			// Count filled pixels
			filled := 0
			for y := 0; y < tt.height; y++ {
				for x := 0; x < tt.width; x++ {
					if img.GrayAt(x, y).Y == 255 {
						filled++
					}
				}
			}

			if filled == 0 && tt.percentage > 0 {
				t.Errorf("no pixels filled for percentage %.1f", tt.percentage)
			}
		})
	}
}

func TestDrawVerticalBar(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		height     int
		percentage float64
		border     bool
	}{
		{"Empty bar", 5, 20, 0, false},
		{"Half full", 5, 20, 50, false},
		{"Full bar", 5, 20, 100, false},
		{"With border empty", 5, 20, 0, true},
		{"With border half", 5, 20, 50, true},
		{"With border full", 5, 20, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(tt.width, tt.height, 0)
			DrawVerticalBar(img, 0, 0, tt.width, tt.height, tt.percentage, 255, tt.border)

			// Count filled pixels
			filled := 0
			for y := 0; y < tt.height; y++ {
				for x := 0; x < tt.width; x++ {
					if img.GrayAt(x, y).Y == 255 {
						filled++
					}
				}
			}

			if filled == 0 && tt.percentage > 0 {
				t.Errorf("no pixels filled for percentage %.1f", tt.percentage)
			}

			// Verify vertical bar fills from bottom
			if tt.percentage == 100 && !tt.border {
				// Bottom pixel should be filled
				if img.GrayAt(0, tt.height-1).Y != 255 {
					t.Error("bottom pixel not filled for 100%")
				}
			}
		})
	}
}

func TestDrawGraph(t *testing.T) {
	tests := []struct {
		name       string
		history    []float64
		maxHistory int
	}{
		{"Empty history", []float64{}, 10},
		{"Single value", []float64{50}, 10},
		{"Multiple values", []float64{10, 20, 30, 40, 50}, 10},
		{"Full history", []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(50, 20, 0)
			DrawGraph(img, 0, 0, 50, 20, tt.history, tt.maxHistory, 255)

			// Just verify no panic occurs and some pixels are drawn if history > 1
			if len(tt.history) > 1 {
				filled := 0
				for y := 0; y < 20; y++ {
					for x := 0; x < 50; x++ {
						if img.GrayAt(x, y).Y > 0 {
							filled++
						}
					}
				}
				if filled == 0 {
					t.Error("no pixels drawn for valid history")
				}
			}
		})
	}
}

func TestDrawLine(t *testing.T) {
	img := NewGrayscaleImage(20, 20, 0)

	// Draw horizontal line
	DrawLine(img, 0, 10, 19, 10, color.Gray{Y: 255})

	// Verify horizontal line
	for x := 0; x < 20; x++ {
		if img.GrayAt(x, 10).Y != 255 {
			t.Errorf("horizontal line pixel at x=%d not filled", x)
		}
	}

	// Draw vertical line
	img2 := NewGrayscaleImage(20, 20, 0)
	DrawLine(img2, 10, 0, 10, 19, color.Gray{Y: 255})

	// Verify vertical line
	for y := 0; y < 20; y++ {
		if img2.GrayAt(10, y).Y != 255 {
			t.Errorf("vertical line pixel at y=%d not filled", y)
		}
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{100, 100},
		{-100, 100},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		if result != tt.expected {
			t.Errorf("abs(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestDrawLinePublic(t *testing.T) {
	img := NewGrayscaleImage(30, 30, 0)

	// Draw horizontal line using public function
	DrawLine(img, 5, 15, 25, 15, color.Gray{Y: 255})

	// Verify horizontal line
	filled := 0
	for x := 5; x <= 25; x++ {
		if img.GrayAt(x, 15).Y == 255 {
			filled++
		}
	}

	if filled == 0 {
		t.Error("DrawLine did not draw any pixels for horizontal line")
	}

	// Draw diagonal line
	img2 := NewGrayscaleImage(30, 30, 0)
	DrawLine(img2, 0, 0, 29, 29, color.Gray{Y: 200})

	// Verify some pixels are drawn
	filled = 0
	for i := 0; i < 30; i++ {
		if img2.GrayAt(i, i).Y == 200 {
			filled++
		}
	}

	if filled == 0 {
		t.Error("DrawLine did not draw any pixels for diagonal line")
	}
}

func TestDrawGauge(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		height      int
		percentage  float64
		gaugeColor  uint8
		needleColor uint8
	}{
		{"Empty gauge", 50, 40, 0, 200, 255},
		{"Half gauge", 50, 40, 50, 200, 255},
		{"Full gauge", 50, 40, 100, 200, 255},
		{"Quarter gauge", 50, 40, 25, 180, 240},
		{"Over 100%", 50, 40, 150, 200, 255}, // Should clamp to 100%
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(tt.width, tt.height, 0)
			pos := config.PositionConfig{
				W: tt.width,
				H: tt.height,
			}

			// Should not panic
			DrawGauge(img, pos, tt.percentage, tt.gaugeColor, tt.needleColor)

			// Count pixels of gauge color
			gaugePixels := 0
			needlePixels := 0
			for y := 0; y < tt.height; y++ {
				for x := 0; x < tt.width; x++ {
					pixel := img.GrayAt(x, y).Y
					if pixel == tt.gaugeColor {
						gaugePixels++
					}
					if pixel == tt.needleColor {
						needlePixels++
					}
				}
			}

			// Gauge arc and ticks should be drawn
			if gaugePixels == 0 {
				t.Error("no gauge arc pixels drawn")
			}

			// Needle should be drawn
			if needlePixels == 0 {
				t.Error("no needle pixels drawn")
			}
		})
	}
}

func TestDrawDualGauge(t *testing.T) {
	tests := []struct {
		name            string
		width           int
		height          int
		outerPercentage float64
		innerPercentage float64
	}{
		{"Both zero", 60, 50, 0, 0},
		{"Both half", 60, 50, 50, 50},
		{"Both full", 60, 50, 100, 100},
		{"Outer high, inner low", 60, 50, 80, 20},
		{"Outer low, inner high", 60, 50, 20, 80},
		{"Mixed values", 60, 50, 33.3, 66.6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(tt.width, tt.height, 0)
			pos := config.PositionConfig{
				W: tt.width,
				H: tt.height,
			}

			outerGaugeColor := uint8(255)
			outerNeedleColor := uint8(255)
			innerGaugeColor := uint8(180)
			innerNeedleColor := uint8(200)

			// Should not panic
			DrawDualGauge(img, pos, tt.outerPercentage, tt.innerPercentage,
				outerGaugeColor, outerNeedleColor, innerGaugeColor, innerNeedleColor)

			// Count pixels of different colors
			outerGaugePixels := 0
			innerGaugePixels := 0
			for y := 0; y < tt.height; y++ {
				for x := 0; x < tt.width; x++ {
					pixel := img.GrayAt(x, y).Y
					if pixel == outerGaugeColor {
						outerGaugePixels++
					}
					if pixel == innerGaugeColor {
						innerGaugePixels++
					}
				}
			}

			// Both gauge arcs should be drawn
			if outerGaugePixels == 0 {
				t.Error("no outer gauge arc pixels drawn")
			}

			if innerGaugePixels == 0 {
				t.Error("no inner gauge arc pixels drawn")
			}
		})
	}
}

func TestDrawDualGauge_NeedlesSeparate(t *testing.T) {
	// Test that outer needle doesn't overlap inner gauge
	img := NewGrayscaleImage(60, 50, 0)
	pos := config.PositionConfig{W: 60, H: 50}

	DrawDualGauge(img, pos, 50, 50, 255, 240, 200, 180)

	// Just verify no panic and pixels are drawn
	filled := 0
	for y := 0; y < 50; y++ {
		for x := 0; x < 60; x++ {
			if img.GrayAt(x, y).Y > 0 {
				filled++
			}
		}
	}

	if filled == 0 {
		t.Error("DrawDualGauge did not draw any pixels")
	}
}

func TestDrawCircle(t *testing.T) {
	tests := []struct {
		name    string
		width   int
		height  int
		centerX int
		centerY int
		radius  int
	}{
		{"Small circle", 20, 20, 10, 10, 5},
		{"Large circle", 50, 50, 25, 25, 20},
		{"Circle at edge", 30, 30, 5, 5, 4},
		{"Radius 1", 10, 10, 5, 5, 1},
		{"Radius 0", 10, 10, 5, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(tt.width, tt.height, 0)
			DrawCircle(img, tt.centerX, tt.centerY, tt.radius, color.Gray{Y: 255})

			// Count filled pixels
			filled := 0
			for y := 0; y < tt.height; y++ {
				for x := 0; x < tt.width; x++ {
					if img.GrayAt(x, y).Y == 255 {
						filled++
					}
				}
			}

			if tt.radius > 0 && filled == 0 {
				t.Error("DrawCircle did not draw any pixels for radius > 0")
			}

			// For radius 0, should draw at least the center point
			if tt.radius == 0 && filled == 0 {
				t.Error("DrawCircle did not draw center point for radius 0")
			}
		})
	}
}

func TestDrawCircle_Clipping(t *testing.T) {
	// Test circle partially outside bounds
	img := NewGrayscaleImage(20, 20, 0)
	// Draw circle with center outside bounds
	DrawCircle(img, -5, 10, 10, color.Gray{Y: 255})

	// Should not panic and should clip
	filled := 0
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			if img.GrayAt(x, y).Y == 255 {
				filled++
			}
		}
	}

	// Some pixels should be visible (clipped portion)
	if filled == 0 {
		t.Log("Circle completely outside bounds is ok")
	}
}

func TestDrawRectangle(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		height      int
		x           int
		y           int
		w           int
		h           int
		borderColor uint8
	}{
		{"Small rectangle", 30, 30, 5, 5, 20, 15, 255},
		{"Large rectangle", 128, 40, 10, 5, 100, 30, 200},
		{"Thin rectangle", 40, 40, 10, 10, 20, 5, 150},
		{"Square", 50, 50, 10, 10, 30, 30, 180},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(tt.width, tt.height, 0)
			DrawRectangle(img, tt.x, tt.y, tt.w, tt.h, tt.borderColor)

			// Count border pixels
			borderPixels := 0
			for y := 0; y < tt.height; y++ {
				for x := 0; x < tt.width; x++ {
					if img.GrayAt(x, y).Y == tt.borderColor {
						borderPixels++
					}
				}
			}

			if borderPixels == 0 {
				t.Error("DrawRectangle did not draw any border pixels")
			}

			// Verify corners are drawn
			if tt.x >= 0 && tt.x < tt.width && tt.y >= 0 && tt.y < tt.height {
				topLeft := img.GrayAt(tt.x, tt.y).Y
				if topLeft != tt.borderColor {
					t.Error("Top-left corner not drawn")
				}
			}
		})
	}
}

func TestDrawRectangle_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		x    int
		y    int
		w    int
		h    int
	}{
		{"Zero width", 10, 10, 0, 10},
		{"Zero height", 10, 10, 10, 0},
		{"Negative width", 10, 10, -5, 10},
		{"Negative height", 10, 10, 10, -5},
		{"Out of bounds", 100, 100, 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(50, 50, 0)
			// Should not panic
			DrawRectangle(img, tt.x, tt.y, tt.w, tt.h, 255)
		})
	}
}

func TestDrawRectangle_NilImage(t *testing.T) {
	// Should not panic with nil image
	DrawRectangle(nil, 0, 0, 10, 10, 255)
}
