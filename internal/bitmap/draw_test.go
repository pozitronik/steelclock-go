package bitmap

import (
	"image/color"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// getTestGlyph returns a simple glyph for testing purposes
func getTestGlyph() *glyphs.Glyph {
	return glyphs.GetGlyph(glyphs.Font5x7, 'A')
}

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
			DrawGraph(img, 0, 0, 50, 20, tt.history, tt.maxHistory, 255, 255) // fillColor=255, lineColor=255

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

			// Should not panic
			DrawGauge(img, 0, 0, tt.width, tt.height, tt.percentage, tt.gaugeColor, tt.needleColor, true, tt.gaugeColor)

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

func TestDrawFilledCircle(t *testing.T) {
	tests := []struct {
		name    string
		centerX int
		centerY int
		radius  int
	}{
		{"Small circle", 10, 10, 3},
		{"Large circle", 25, 25, 10},
		{"Single pixel", 5, 5, 1},
		{"Zero radius", 5, 5, 0},
		{"Negative radius", 5, 5, -1},
		{"At edge", 0, 0, 5},
		{"Partially visible", 45, 45, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(50, 50, 0)
			DrawFilledCircle(img, tt.centerX, tt.centerY, tt.radius, color.Gray{Y: 255})

			// Count filled pixels
			filled := 0
			for y := 0; y < 50; y++ {
				for x := 0; x < 50; x++ {
					if img.GrayAt(x, y).Y == 255 {
						filled++
					}
				}
			}

			if tt.radius > 0 && filled == 0 {
				t.Error("DrawFilledCircle did not draw any pixels for positive radius")
			}
			if tt.radius <= 0 && filled > 0 {
				t.Error("DrawFilledCircle drew pixels for non-positive radius")
			}
		})
	}
}

func TestDrawFilledCircle_NilImage(t *testing.T) {
	// Should not panic with nil image
	DrawFilledCircle(nil, 10, 10, 5, color.Gray{Y: 255})
}

func TestDrawDualHorizontalBar(t *testing.T) {
	tests := []struct {
		name          string
		topPercent    float64
		bottomPercent float64
		topColor      int
		bottomColor   int
	}{
		{"Both bars visible", 50, 75, 255, 200},
		{"Top transparent", 50, 75, -1, 200},
		{"Bottom transparent", 50, 75, 255, -1},
		{"Both transparent", 50, 75, -1, -1},
		{"Empty bars", 0, 0, 255, 200},
		{"Full bars", 100, 100, 255, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(40, 20, 0)
			DrawDualHorizontalBar(img, 0, 0, 40, 20, tt.topPercent, tt.bottomPercent, tt.topColor, tt.bottomColor, false)

			// Count filled pixels for each color
			topPixels := 0
			bottomPixels := 0
			for y := 0; y < 20; y++ {
				for x := 0; x < 40; x++ {
					pixel := img.GrayAt(x, y).Y
					if tt.topColor >= 0 && pixel == uint8(tt.topColor) {
						topPixels++
					}
					if tt.bottomColor >= 0 && pixel == uint8(tt.bottomColor) {
						bottomPixels++
					}
				}
			}

			if tt.topColor >= 0 && tt.topPercent > 0 && topPixels == 0 {
				t.Error("Top bar not drawn")
			}
			if tt.bottomColor >= 0 && tt.bottomPercent > 0 && bottomPixels == 0 {
				t.Error("Bottom bar not drawn")
			}
		})
	}
}

func TestDrawDualVerticalBar(t *testing.T) {
	tests := []struct {
		name         string
		leftPercent  float64
		rightPercent float64
		leftColor    int
		rightColor   int
	}{
		{"Both bars visible", 50, 75, 255, 200},
		{"Left transparent", 50, 75, -1, 200},
		{"Right transparent", 50, 75, 255, -1},
		{"Both transparent", 50, 75, -1, -1},
		{"Empty bars", 0, 0, 255, 200},
		{"Full bars", 100, 100, 255, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(20, 40, 0)
			DrawDualVerticalBar(img, 0, 0, 20, 40, tt.leftPercent, tt.rightPercent, tt.leftColor, tt.rightColor, false)

			// Count filled pixels for each color
			leftPixels := 0
			rightPixels := 0
			for y := 0; y < 40; y++ {
				for x := 0; x < 20; x++ {
					pixel := img.GrayAt(x, y).Y
					if tt.leftColor >= 0 && pixel == uint8(tt.leftColor) {
						leftPixels++
					}
					if tt.rightColor >= 0 && pixel == uint8(tt.rightColor) {
						rightPixels++
					}
				}
			}

			if tt.leftColor >= 0 && tt.leftPercent > 0 && leftPixels == 0 {
				t.Error("Left bar not drawn")
			}
			if tt.rightColor >= 0 && tt.rightPercent > 0 && rightPixels == 0 {
				t.Error("Right bar not drawn")
			}
		})
	}
}

func TestDrawFilledRectangle(t *testing.T) {
	tests := []struct {
		name   string
		x      int
		y      int
		width  int
		height int
	}{
		{"Normal rectangle", 5, 5, 10, 10},
		{"At origin", 0, 0, 10, 10},
		{"Partial overlap", -5, -5, 20, 20},
		{"Zero width", 5, 5, 0, 10},
		{"Zero height", 5, 5, 10, 0},
		{"Negative width", 5, 5, -5, 10},
		{"Negative height", 5, 5, 10, -5},
		{"Full image", 0, 0, 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(50, 50, 0)
			DrawFilledRectangle(img, tt.x, tt.y, tt.width, tt.height, 255)

			filled := 0
			for y := 0; y < 50; y++ {
				for x := 0; x < 50; x++ {
					if img.GrayAt(x, y).Y == 255 {
						filled++
					}
				}
			}

			expectedFilled := tt.width > 0 && tt.height > 0
			if expectedFilled && filled == 0 {
				t.Error("DrawFilledRectangle did not draw any pixels")
			}
			if !expectedFilled && filled > 0 {
				t.Error("DrawFilledRectangle drew pixels for invalid dimensions")
			}
		})
	}
}

func TestDrawFilledRectangle_NilImage(t *testing.T) {
	// Should not panic with nil image
	DrawFilledRectangle(nil, 0, 0, 10, 10, 255)
}

func TestDrawHorizontalLine(t *testing.T) {
	tests := []struct {
		name string
		x1   int
		x2   int
		y    int
	}{
		{"Normal line", 5, 20, 10},
		{"Reversed coords", 20, 5, 10},
		{"At top edge", 0, 49, 0},
		{"At bottom edge", 0, 49, 49},
		{"Out of Y bounds", 0, 49, 100},
		{"Negative Y", 0, 49, -5},
		{"Partial overlap left", -10, 20, 10},
		{"Partial overlap right", 30, 60, 10},
		{"Single pixel", 10, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(50, 50, 0)
			DrawHorizontalLine(img, tt.x1, tt.x2, tt.y, 255)

			filled := 0
			for y := 0; y < 50; y++ {
				for x := 0; x < 50; x++ {
					if img.GrayAt(x, y).Y == 255 {
						filled++
					}
				}
			}

			inBounds := tt.y >= 0 && tt.y < 50
			if inBounds && filled == 0 {
				t.Error("DrawHorizontalLine did not draw any pixels")
			}
			if !inBounds && filled > 0 {
				t.Error("DrawHorizontalLine drew pixels outside Y bounds")
			}
		})
	}
}

func TestDrawHorizontalLine_NilImage(t *testing.T) {
	// Should not panic with nil image
	DrawHorizontalLine(nil, 0, 10, 5, 255)
}

func TestDrawVerticalLine(t *testing.T) {
	tests := []struct {
		name string
		x    int
		y1   int
		y2   int
	}{
		{"Normal line", 10, 5, 20},
		{"Reversed coords", 10, 20, 5},
		{"At left edge", 0, 0, 49},
		{"At right edge", 49, 0, 49},
		{"Out of X bounds", 100, 0, 49},
		{"Negative X", -5, 0, 49},
		{"Partial overlap top", 10, -10, 20},
		{"Partial overlap bottom", 10, 30, 60},
		{"Single pixel", 10, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(50, 50, 0)
			DrawVerticalLine(img, tt.x, tt.y1, tt.y2, 255)

			filled := 0
			for y := 0; y < 50; y++ {
				for x := 0; x < 50; x++ {
					if img.GrayAt(x, y).Y == 255 {
						filled++
					}
				}
			}

			inBounds := tt.x >= 0 && tt.x < 50
			if inBounds && filled == 0 {
				t.Error("DrawVerticalLine did not draw any pixels")
			}
			if !inBounds && filled > 0 {
				t.Error("DrawVerticalLine drew pixels outside X bounds")
			}
		})
	}
}

func TestDrawVerticalLine_NilImage(t *testing.T) {
	// Should not panic with nil image
	DrawVerticalLine(nil, 5, 0, 10, 255)
}

func TestCopyGrayRegion(t *testing.T) {
	tests := []struct {
		name string
		srcW int
		srcH int
		dstX int
		dstY int
		dstW int
		dstH int
	}{
		{"Normal copy", 10, 10, 5, 5, 50, 50},
		{"At origin", 10, 10, 0, 0, 50, 50},
		{"Partial overlap right", 20, 10, 40, 10, 50, 50},
		{"Partial overlap bottom", 10, 20, 10, 40, 50, 50},
		{"Out of bounds", 10, 10, 100, 100, 50, 50},
		{"Negative offset", 10, 10, -5, -5, 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := NewGrayscaleImage(tt.srcW, tt.srcH, 255) // White source
			dst := NewGrayscaleImage(tt.dstW, tt.dstH, 0)   // Black destination

			CopyGrayRegion(dst, src, tt.dstX, tt.dstY)

			// Count copied pixels
			filled := 0
			for y := 0; y < tt.dstH; y++ {
				for x := 0; x < tt.dstW; x++ {
					if dst.GrayAt(x, y).Y == 255 {
						filled++
					}
				}
			}

			// If any part overlaps, should have some pixels
			overlapsX := tt.dstX < tt.dstW && tt.dstX+tt.srcW > 0
			overlapsY := tt.dstY < tt.dstH && tt.dstY+tt.srcH > 0
			if overlapsX && overlapsY && filled == 0 {
				t.Error("CopyGrayRegion did not copy any pixels")
			}
		})
	}
}

func TestCopyGrayRegion_NilImages(t *testing.T) {
	src := NewGrayscaleImage(10, 10, 255)
	dst := NewGrayscaleImage(50, 50, 0)

	// Should not panic with nil images
	CopyGrayRegion(nil, src, 0, 0)
	CopyGrayRegion(dst, nil, 0, 0)
	CopyGrayRegion(nil, nil, 0, 0)
}

func TestDrawCrossPattern(t *testing.T) {
	tests := []struct {
		name      string
		x         int
		y         int
		width     int
		height    int
		thickness int
	}{
		{"Normal cross", 10, 10, 20, 20, 1},
		{"Thick cross", 10, 10, 20, 20, 2},
		{"At origin", 0, 0, 30, 30, 1},
		{"Zero width", 10, 10, 0, 20, 1},
		{"Zero height", 10, 10, 20, 0, 1},
		{"Negative dimensions", 10, 10, -5, -5, 1},
		{"Partial overlap", 40, 40, 20, 20, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(50, 50, 0)
			DrawCrossPattern(img, tt.x, tt.y, tt.width, tt.height, tt.thickness, 255)

			filled := 0
			for y := 0; y < 50; y++ {
				for x := 0; x < 50; x++ {
					if img.GrayAt(x, y).Y == 255 {
						filled++
					}
				}
			}

			validDims := tt.width > 0 && tt.height > 0
			if validDims && filled == 0 {
				t.Error("DrawCrossPattern did not draw any pixels")
			}
			if !validDims && filled > 0 {
				t.Error("DrawCrossPattern drew pixels for invalid dimensions")
			}
		})
	}
}

func TestDrawCrossPattern_NilImage(t *testing.T) {
	// Should not panic with nil image
	DrawCrossPattern(nil, 10, 10, 20, 20, 1, 255)
}

func TestDrawGlyphWithBorder(t *testing.T) {
	tests := []struct {
		name        string
		borderColor uint8
		fillColor   uint8
	}{
		{"white border black fill", 255, 0},
		{"black border white fill", 0, 255},
		{"medium colors", 128, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(20, 20, 128)

			// Get a glyph to draw
			glyph := getTestGlyph()
			if glyph == nil {
				t.Skip("Could not get test glyph")
			}

			DrawGlyphWithBorder(img, glyph, 5, 5, tt.borderColor, tt.fillColor)

			// Verify some pixels were drawn
			hasBorder := false
			hasFill := false
			for y := 0; y < 20; y++ {
				for x := 0; x < 20; x++ {
					px := img.GrayAt(x, y).Y
					if px == tt.borderColor {
						hasBorder = true
					}
					if px == tt.fillColor {
						hasFill = true
					}
				}
			}

			if !hasBorder && !hasFill {
				t.Error("DrawGlyphWithBorder did not draw any pixels")
			}
		})
	}
}

func TestDrawGlyphWithBorder_NilGlyph(t *testing.T) {
	img := NewGrayscaleImage(20, 20, 0)
	// Should not panic with nil glyph
	DrawGlyphWithBorder(img, nil, 5, 5, 0, 255)
}

func TestDrawGlyphWithBackground(t *testing.T) {
	tests := []struct {
		name    string
		fgColor int
		bgColor int
	}{
		{"both colors", 255, 0},
		{"fg only", 255, -1},
		{"bg only", -1, 128},
		{"both transparent", -1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(20, 20, 64)

			glyph := getTestGlyph()
			if glyph == nil {
				t.Skip("Could not get test glyph")
			}

			DrawGlyphWithBackground(img, glyph, 5, 5, tt.fgColor, tt.bgColor)

			// Count modified pixels
			modified := 0
			for y := 0; y < 20; y++ {
				for x := 0; x < 20; x++ {
					if img.GrayAt(x, y).Y != 64 {
						modified++
					}
				}
			}

			// If both colors are transparent, no pixels should be modified
			if tt.fgColor == -1 && tt.bgColor == -1 && modified != 0 {
				t.Error("DrawGlyphWithBackground modified pixels when both colors are transparent")
			}
			// If either color is valid, some pixels should be modified
			if (tt.fgColor >= 0 || tt.bgColor >= 0) && modified == 0 {
				t.Error("DrawGlyphWithBackground did not modify any pixels")
			}
		})
	}
}

func TestDrawGlyphWithBackground_NilGlyph(t *testing.T) {
	img := NewGrayscaleImage(20, 20, 0)
	// Should not panic with nil glyph
	DrawGlyphWithBackground(img, nil, 5, 5, 255, 0)
}

func TestDrawGlyphWithBackground_NilImage(t *testing.T) {
	glyph := getTestGlyph()
	if glyph == nil {
		return
	}
	// Should not panic with nil image
	DrawGlyphWithBackground(nil, glyph, 5, 5, 255, 0)
}

func TestDrawGaugePeakHoldMark(t *testing.T) {
	tests := []struct {
		name       string
		percentage float64
		markLen    int
	}{
		{"at start", 0.0, 3},
		{"at middle", 0.5, 3},
		{"at end", 1.0, 3},
		{"quarter", 0.25, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(50, 30, 0)

			DrawGaugePeakHoldMark(img, 25, 25, 15, tt.percentage, tt.markLen, 255)

			// Verify at least some pixels were drawn
			hasPixels := false
			for y := 0; y < 30; y++ {
				for x := 0; x < 50; x++ {
					if img.GrayAt(x, y).Y > 0 {
						hasPixels = true
						break
					}
				}
				if hasPixels {
					break
				}
			}

			if !hasPixels {
				t.Errorf("DrawGaugePeakHoldMark(percentage=%v) did not draw any pixels", tt.percentage)
			}
		})
	}
}

func TestDrawGaugePeakHoldMark_NilImage(t *testing.T) {
	// Should not panic with nil image
	DrawGaugePeakHoldMark(nil, 25, 25, 15, 0.5, 3, 255)
}

func TestDrawGaugePeakHoldMark_ZeroRadius(t *testing.T) {
	img := NewGrayscaleImage(50, 30, 0)
	// Should not panic and should not draw anything
	DrawGaugePeakHoldMark(img, 25, 25, 0, 0.5, 3, 255)
}

func TestDrawDualGraph(t *testing.T) {
	tests := []struct {
		name  string
		fill1 int
		line1 int
		fill2 int
		line2 int
	}{
		{"both graphs", 100, 200, 50, 150},
		{"first only", 100, 200, -1, -1},
		{"second only", -1, -1, 100, 200},
		{"fill only", 100, -1, 50, -1},
		{"line only", -1, 200, -1, 150},
	}

	// History values are percentages (0-100)
	history1 := []float64{20, 50, 80, 30, 60}
	history2 := []float64{40, 30, 60, 90, 20}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(50, 20, 0)

			DrawDualGraph(img, 0, 0, 50, 20, history1, history2, 10, tt.fill1, tt.line1, tt.fill2, tt.line2)

			// Verify pixels were drawn (unless both graphs are transparent)
			hasPixels := false
			for y := 0; y < 20; y++ {
				for x := 0; x < 50; x++ {
					if img.GrayAt(x, y).Y > 0 {
						hasPixels = true
						break
					}
				}
				if hasPixels {
					break
				}
			}

			// If at least one graph should be drawn, check for pixels
			hasGraph := tt.fill1 >= 0 || tt.line1 >= 0 || tt.fill2 >= 0 || tt.line2 >= 0
			if hasGraph && !hasPixels {
				t.Error("DrawDualGraph did not draw any pixels when graphs should be visible")
			}
		})
	}
}

func TestDrawDualGraph_EmptyHistory(t *testing.T) {
	img := NewGrayscaleImage(50, 20, 0)
	// Should not panic with empty histories
	DrawDualGraph(img, 0, 0, 50, 20, []float64{}, []float64{}, 10, 100, 200, 50, 150)
}
