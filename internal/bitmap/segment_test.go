package bitmap

import (
	"image"
	"image/color"
	"testing"
)

func TestDrawSegmentDigit_AllDigits(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 20, 30))

	// Test all valid digits with all styles
	styles := []SegmentStyle{SegmentStyleRectangle, SegmentStyleHexagon, SegmentStyleRounded}

	for _, style := range styles {
		t.Run(string(style), func(t *testing.T) {
			for digit := 0; digit <= 9; digit++ {
				// Clear image
				for y := 0; y < 30; y++ {
					for x := 0; x < 20; x++ {
						img.SetGray(x, y, color.Gray{Y: 255})
					}
				}

				DrawSegmentDigit(img, 2, 2, 16, 26, digit, style, 2, 255, 0)

				// Verify that at least some pixels are lit (non-zero)
				hasLitPixel := false
				for y := 0; y < 30; y++ {
					for x := 0; x < 20; x++ {
						if img.GrayAt(x, y).Y > 0 {
							hasLitPixel = true
							break
						}
					}
					if hasLitPixel {
						break
					}
				}

				if !hasLitPixel {
					t.Errorf("DrawSegmentDigit(%d, %s) produced no visible pixels", digit, style)
				}
			}
		})
	}
}

func TestDrawSegmentDigit_InvalidDigit(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 20, 30))

	// Fill with white
	for y := 0; y < 30; y++ {
		for x := 0; x < 20; x++ {
			img.SetGray(x, y, color.Gray{Y: 255})
		}
	}

	// Draw invalid digit - should not change anything
	DrawSegmentDigit(img, 2, 2, 16, 26, -1, SegmentStyleRectangle, 2, 255, 0)
	DrawSegmentDigit(img, 2, 2, 16, 26, 10, SegmentStyleRectangle, 2, 255, 0)
	DrawSegmentDigit(img, 2, 2, 16, 26, 100, SegmentStyleRectangle, 2, 255, 0)

	// Image should still be all white
	for y := 0; y < 30; y++ {
		for x := 0; x < 20; x++ {
			if img.GrayAt(x, y).Y != 255 {
				t.Errorf("DrawSegmentDigit(invalid) modified image at (%d, %d)", x, y)
				return
			}
		}
	}
}

func TestDrawSegmentDigitAnimated(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 20, 30))

	// Test with different animation progress values
	testCases := []struct {
		name     string
		progress float64
	}{
		{"zero", 0.0},
		{"half", 0.5},
		{"full", 1.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear image
			for y := 0; y < 30; y++ {
				for x := 0; x < 20; x++ {
					img.SetGray(x, y, color.Gray{Y: 0})
				}
			}

			DrawSegmentDigitAnimated(img, 2, 2, 16, 26, 8, SegmentStyleRectangle, 2, 255, 0, tc.progress)

			// Check that animation progress affects pixel brightness
			maxBrightness := uint8(0)
			for y := 0; y < 30; y++ {
				for x := 0; x < 20; x++ {
					if img.GrayAt(x, y).Y > maxBrightness {
						maxBrightness = img.GrayAt(x, y).Y
					}
				}
			}

			expectedMax := uint8(float64(255) * tc.progress)
			if maxBrightness != expectedMax {
				t.Errorf("DrawSegmentDigitAnimated(progress=%v): max brightness = %d, want %d", tc.progress, maxBrightness, expectedMax)
			}
		})
	}
}

func TestDrawSegmentColon_Dots(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 10, 30))

	// Clear image
	for y := 0; y < 30; y++ {
		for x := 0; x < 10; x++ {
			img.SetGray(x, y, color.Gray{Y: 0})
		}
	}

	DrawSegmentColon(img, 0, 0, 10, 30, ColonStyleDots, 2, 255, true)

	// Check that some pixels are lit
	hasLitPixel := false
	for y := 0; y < 30; y++ {
		for x := 0; x < 10; x++ {
			if img.GrayAt(x, y).Y > 0 {
				hasLitPixel = true
				break
			}
		}
		if hasLitPixel {
			break
		}
	}

	if !hasLitPixel {
		t.Error("DrawSegmentColon(dots, visible=true) produced no visible pixels")
	}
}

func TestDrawSegmentColon_Bar(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 10, 30))

	// Clear image
	for y := 0; y < 30; y++ {
		for x := 0; x < 10; x++ {
			img.SetGray(x, y, color.Gray{Y: 0})
		}
	}

	DrawSegmentColon(img, 0, 0, 10, 30, ColonStyleBar, 2, 255, true)

	// Check that some pixels are lit
	hasLitPixel := false
	for y := 0; y < 30; y++ {
		for x := 0; x < 10; x++ {
			if img.GrayAt(x, y).Y > 0 {
				hasLitPixel = true
				break
			}
		}
		if hasLitPixel {
			break
		}
	}

	if !hasLitPixel {
		t.Error("DrawSegmentColon(bar, visible=true) produced no visible pixels")
	}
}

func TestDrawSegmentColon_None(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 10, 30))

	// Fill with black
	for y := 0; y < 30; y++ {
		for x := 0; x < 10; x++ {
			img.SetGray(x, y, color.Gray{Y: 0})
		}
	}

	DrawSegmentColon(img, 0, 0, 10, 30, ColonStyleNone, 2, 255, true)

	// Image should still be all black
	for y := 0; y < 30; y++ {
		for x := 0; x < 10; x++ {
			if img.GrayAt(x, y).Y != 0 {
				t.Errorf("DrawSegmentColon(none) modified image at (%d, %d)", x, y)
				return
			}
		}
	}
}

func TestDrawSegmentColon_NotVisible(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 10, 30))

	// Fill with black
	for y := 0; y < 30; y++ {
		for x := 0; x < 10; x++ {
			img.SetGray(x, y, color.Gray{Y: 0})
		}
	}

	DrawSegmentColon(img, 0, 0, 10, 30, ColonStyleDots, 2, 255, false)

	// Image should still be all black when not visible
	for y := 0; y < 30; y++ {
		for x := 0; x < 10; x++ {
			if img.GrayAt(x, y).Y != 0 {
				t.Errorf("DrawSegmentColon(visible=false) modified image at (%d, %d)", x, y)
				return
			}
		}
	}
}

func TestSegmentPatterns(t *testing.T) {
	// Verify segment patterns are correctly defined
	// Each pattern should be non-zero (all digits show at least one segment)
	for digit, pattern := range SegmentPatterns {
		if pattern == 0 {
			t.Errorf("SegmentPatterns[%d] = 0, want non-zero", digit)
		}
	}

	// Digit 8 should have all segments on
	if SegmentPatterns[8] != 0b1111111 {
		t.Errorf("SegmentPatterns[8] = %b, want 1111111", SegmentPatterns[8])
	}

	// Digit 1 should only have segments b and c
	if SegmentPatterns[1] != 0b0000110 {
		t.Errorf("SegmentPatterns[1] = %b, want 0000110", SegmentPatterns[1])
	}
}

func TestSegmentStyles(t *testing.T) {
	// Test that style constants have expected values
	tests := []struct {
		style SegmentStyle
		want  string
	}{
		{SegmentStyleRectangle, "rectangle"},
		{SegmentStyleHexagon, "hexagon"},
		{SegmentStyleRounded, "rounded"},
	}

	for _, tc := range tests {
		if string(tc.style) != tc.want {
			t.Errorf("SegmentStyle constant = %q, want %q", string(tc.style), tc.want)
		}
	}
}

func TestColonStyles(t *testing.T) {
	// Test that colon style constants have expected values
	tests := []struct {
		style ColonStyle
		want  string
	}{
		{ColonStyleDots, "dots"},
		{ColonStyleBar, "bar"},
		{ColonStyleNone, "none"},
	}

	for _, tc := range tests {
		if string(tc.style) != tc.want {
			t.Errorf("ColonStyle constant = %q, want %q", string(tc.style), tc.want)
		}
	}
}
