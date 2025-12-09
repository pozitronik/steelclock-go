package bitmap

import (
	"image"
	"image/color"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
)

func TestGetInternalFontByName(t *testing.T) {
	tests := []struct {
		name     string
		fontName string
		wantNil  bool
	}{
		{"pixel3x5", FontNamePixel3x5, false},
		{"pixel5x7", FontNamePixel5x7, false},
		{"3x5 alias", "3x5", false},
		{"5x7 alias", "5x7", false},
		{"unknown font", "unknown", true},
		{"empty name", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			font := GetInternalFontByName(tc.fontName)
			if tc.wantNil && font != nil {
				t.Errorf("GetInternalFontByName(%q) = non-nil, want nil", tc.fontName)
			}
			if !tc.wantNil && font == nil {
				t.Errorf("GetInternalFontByName(%q) = nil, want non-nil", tc.fontName)
			}
		})
	}
}

func TestMeasureInternalText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		glyphSet *glyphs.GlyphSet
		wantGT   int // expect result greater than this
	}{
		{"empty text", "", glyphs.Font5x7, -1},
		{"single char 5x7", "A", glyphs.Font5x7, 0},
		{"hello 5x7", "Hello", glyphs.Font5x7, 10},
		{"hello 3x5", "Hello", glyphs.Font3x5, 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			width := MeasureInternalText(tc.text, tc.glyphSet)
			if width <= tc.wantGT {
				t.Errorf("MeasureInternalText(%q) = %d, want > %d", tc.text, width, tc.wantGT)
			}
		})
	}
}

func TestDrawAlignedInternalText(t *testing.T) {
	// Test all alignment combinations
	alignments := []struct {
		horiz, vert string
	}{
		{"left", "top"},
		{"left", "center"},
		{"left", "bottom"},
		{"center", "top"},
		{"center", "center"},
		{"center", "bottom"},
		{"right", "top"},
		{"right", "center"},
		{"right", "bottom"},
	}

	for _, align := range alignments {
		t.Run(align.horiz+"_"+align.vert, func(t *testing.T) {
			img := image.NewGray(image.Rect(0, 0, 50, 20))

			// Clear image
			for y := 0; y < 20; y++ {
				for x := 0; x < 50; x++ {
					img.SetGray(x, y, color.Gray{Y: 0})
				}
			}

			DrawAlignedInternalText(img, "Hi", glyphs.Font5x7, align.horiz, align.vert, 2)

			// Check that some pixels are lit
			hasLitPixel := false
			for y := 0; y < 20; y++ {
				for x := 0; x < 50; x++ {
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
				t.Errorf("DrawAlignedInternalText(%s, %s) produced no visible pixels", align.horiz, align.vert)
			}
		})
	}
}

func TestDrawAlignedInternalText_NilGlyphSet(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 50, 20))

	// Clear image
	for y := 0; y < 20; y++ {
		for x := 0; x < 50; x++ {
			img.SetGray(x, y, color.Gray{Y: 0})
		}
	}

	// Should not panic and use default font
	DrawAlignedInternalText(img, "Hi", nil, "center", "center", 2)

	// Check that some pixels are lit (uses default font)
	hasLitPixel := false
	for y := 0; y < 20; y++ {
		for x := 0; x < 50; x++ {
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
		t.Error("DrawAlignedInternalText with nil glyphSet produced no visible pixels")
	}
}

func TestDrawInternalTextInRect(t *testing.T) {
	// Test various alignments within a rectangle
	alignments := []struct {
		horiz, vert string
	}{
		{"left", "top"},
		{"center", "center"},
		{"right", "bottom"},
	}

	for _, align := range alignments {
		t.Run(align.horiz+"_"+align.vert, func(t *testing.T) {
			img := image.NewGray(image.Rect(0, 0, 100, 50))

			// Clear image
			for y := 0; y < 50; y++ {
				for x := 0; x < 100; x++ {
					img.SetGray(x, y, color.Gray{Y: 0})
				}
			}

			DrawInternalTextInRect(img, "Test", glyphs.Font5x7, 10, 10, 80, 30, align.horiz, align.vert, 2)

			// Check that some pixels are lit within the rectangle area
			hasLitPixel := false
			for y := 10; y < 40; y++ {
				for x := 10; x < 90; x++ {
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
				t.Errorf("DrawInternalTextInRect(%s, %s) produced no visible pixels", align.horiz, align.vert)
			}
		})
	}
}

func TestDrawInternalTextInRect_NilGlyphSet(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 100, 50))

	// Clear image
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			img.SetGray(x, y, color.Gray{Y: 0})
		}
	}

	// Should not panic and use default font
	DrawInternalTextInRect(img, "Test", nil, 10, 10, 80, 30, "center", "center", 2)

	// Check that some pixels are lit
	hasLitPixel := false
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
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
		t.Error("DrawInternalTextInRect with nil glyphSet produced no visible pixels")
	}
}

func TestDrawInternalTextClipped(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		x, y         int
		clipX, clipY int
		clipW, clipH int
		expectPixels bool
	}{
		{"text within clip", "Hi", 5, 5, 0, 0, 50, 20, true},
		{"text at clip edge", "Hi", 0, 0, 0, 0, 50, 20, true},
		{"text above clip", "Hi", 5, -20, 0, 0, 50, 20, false},
		{"text below clip", "Hi", 5, 30, 0, 0, 50, 20, false},
		{"text left of clip", "Hi", -50, 5, 0, 0, 50, 20, false},
		{"text right of clip", "Hi", 100, 5, 0, 0, 50, 20, false},
		{"partial clip top", "Hi", 5, -3, 0, 0, 50, 20, true},
		{"partial clip left", "Hi", -5, 5, 0, 0, 50, 20, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			img := image.NewGray(image.Rect(0, 0, 100, 50))

			// Clear image
			for y := 0; y < 50; y++ {
				for x := 0; x < 100; x++ {
					img.SetGray(x, y, color.Gray{Y: 0})
				}
			}

			DrawInternalTextClipped(img, tc.text, glyphs.Font5x7, tc.x, tc.y, tc.clipX, tc.clipY, tc.clipW, tc.clipH, color.Gray{Y: 255})

			// Check for lit pixels within clip area
			hasLitPixel := false
			for y := tc.clipY; y < tc.clipY+tc.clipH; y++ {
				for x := tc.clipX; x < tc.clipX+tc.clipW; x++ {
					if img.GrayAt(x, y).Y > 0 {
						hasLitPixel = true
						break
					}
				}
				if hasLitPixel {
					break
				}
			}

			if tc.expectPixels && !hasLitPixel {
				t.Errorf("DrawInternalTextClipped(%s) expected pixels but found none", tc.name)
			}
			if !tc.expectPixels && hasLitPixel {
				t.Errorf("DrawInternalTextClipped(%s) expected no pixels but found some", tc.name)
			}
		})
	}
}

func TestDrawInternalTextClipped_NilGlyphSet(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 50, 20))

	// Clear image
	for y := 0; y < 20; y++ {
		for x := 0; x < 50; x++ {
			img.SetGray(x, y, color.Gray{Y: 0})
		}
	}

	// Should not panic and use default font
	DrawInternalTextClipped(img, "Hi", nil, 5, 5, 0, 0, 50, 20, color.Gray{Y: 255})

	// Check that some pixels are lit
	hasLitPixel := false
	for y := 0; y < 20; y++ {
		for x := 0; x < 50; x++ {
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
		t.Error("DrawInternalTextClipped with nil glyphSet produced no visible pixels")
	}
}

func TestDrawInternalTextClipped_UnknownCharacter(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 100, 20))

	// Clear image
	for y := 0; y < 20; y++ {
		for x := 0; x < 100; x++ {
			img.SetGray(x, y, color.Gray{Y: 0})
		}
	}

	// Draw text with unknown characters mixed in
	DrawInternalTextClipped(img, "A\u0001B", glyphs.Font5x7, 5, 5, 0, 0, 100, 20, color.Gray{Y: 255})

	// Should still render known characters
	hasLitPixel := false
	for y := 0; y < 20; y++ {
		for x := 0; x < 100; x++ {
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
		t.Error("DrawInternalTextClipped should render known characters even with unknown chars in string")
	}
}

//goland:noinspection GoBoolExpressions
func TestInternalFontConstants(t *testing.T) {
	if FontNamePixel3x5 != "pixel3x5" {
		t.Errorf("FontNamePixel3x5 = %q, want %q", FontNamePixel3x5, "pixel3x5")
	}
	if FontNamePixel5x7 != "pixel5x7" {
		t.Errorf("FontNamePixel5x7 = %q, want %q", FontNamePixel5x7, "pixel5x7")
	}
}
