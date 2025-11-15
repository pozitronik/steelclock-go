package bitmap

import (
	"image/color"
	"testing"
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
	drawLine(img, 0, 10, 19, 10, color.Gray{Y: 255})

	// Verify horizontal line
	for x := 0; x < 20; x++ {
		if img.GrayAt(x, 10).Y != 255 {
			t.Errorf("horizontal line pixel at x=%d not filled", x)
		}
	}

	// Draw vertical line
	img2 := NewGrayscaleImage(20, 20, 0)
	drawLine(img2, 10, 0, 10, 19, color.Gray{Y: 255})

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
