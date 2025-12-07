package shared

import (
	"image"
	"image/color"
	"testing"
)

func TestNewStatusRenderer(t *testing.T) {
	tests := []struct {
		fontName string
	}{
		{""},      // default
		{"5x7"},   // explicit
		{"3x5"},   // smaller font
		{"bogus"}, // should fall back to default
	}

	for _, tt := range tests {
		t.Run(tt.fontName, func(t *testing.T) {
			r := NewStatusRenderer(tt.fontName)
			if r == nil {
				t.Fatal("NewStatusRenderer returned nil")
			}
			if r.GlyphSet() == nil {
				t.Error("GlyphSet() returned nil")
			}
		})
	}
}

func TestStatusRenderer_MeasureText(t *testing.T) {
	r := NewStatusRenderer("5x7")

	// Empty string
	w, h := r.MeasureText("")
	if w != 0 || h != 0 {
		t.Errorf("MeasureText(\"\") = %d, %d, want 0, 0", w, h)
	}

	// Single character
	w, h = r.MeasureText("A")
	if w <= 0 {
		t.Errorf("MeasureText(\"A\") width = %d, want > 0", w)
	}
	if h != 7 { // 5x7 font has height 7
		t.Errorf("MeasureText(\"A\") height = %d, want 7", h)
	}

	// Multiple characters
	w1, _ := r.MeasureText("A")
	w2, _ := r.MeasureText("AB")
	if w2 <= w1 {
		t.Errorf("MeasureText(\"AB\") width %d should be > MeasureText(\"A\") width %d", w2, w1)
	}
}

func TestStatusRenderer_DrawAt(t *testing.T) {
	r := NewStatusRenderer("5x7")
	img := image.NewGray(image.Rect(0, 0, 50, 20))

	// Draw text
	r.DrawAt(img, "Hi", 5, 5)

	// Check that some pixels were drawn
	hasPixels := false
	for y := 0; y < 20; y++ {
		for x := 0; x < 50; x++ {
			if img.GrayAt(x, y).Y > 0 {
				hasPixels = true
				break
			}
		}
	}
	if !hasPixels {
		t.Error("DrawAt did not draw any pixels")
	}
}

func TestStatusRenderer_DrawCentered(t *testing.T) {
	r := NewStatusRenderer("5x7")
	img := image.NewGray(image.Rect(0, 0, 100, 40))

	r.DrawCentered(img, "Test", 0, 0, 100, 40)

	// Check that text is roughly centered by looking for pixels in center region
	centerX := 50
	centerY := 20
	foundNearCenter := false

	// Allow some tolerance for centering
	for dy := -10; dy <= 10; dy++ {
		for dx := -20; dx <= 20; dx++ {
			if img.GrayAt(centerX+dx, centerY+dy).Y > 0 {
				foundNearCenter = true
				break
			}
		}
	}

	if !foundNearCenter {
		t.Error("DrawCentered did not draw pixels near center")
	}
}

func TestStatusRenderer_DrawLeftAligned(t *testing.T) {
	r := NewStatusRenderer("5x7")
	img := image.NewGray(image.Rect(0, 0, 100, 40))

	r.DrawLeftAligned(img, "Test", 10, 0, 80, 40)

	// Check that first pixels are near left edge (x=10)
	foundNearLeft := false
	for y := 0; y < 40; y++ {
		for x := 10; x < 25; x++ {
			if img.GrayAt(x, y).Y > 0 {
				foundNearLeft = true
				break
			}
		}
	}

	if !foundNearLeft {
		t.Error("DrawLeftAligned did not draw pixels near left edge")
	}
}

func TestStatusRenderer_DrawRightAligned(t *testing.T) {
	r := NewStatusRenderer("5x7")
	img := image.NewGray(image.Rect(0, 0, 100, 40))

	r.DrawRightAligned(img, "Test", 10, 0, 80, 40)

	// Check that last pixels are near right edge (x=90)
	foundNearRight := false
	for y := 0; y < 40; y++ {
		for x := 75; x < 90; x++ {
			if img.GrayAt(x, y).Y > 0 {
				foundNearRight = true
				break
			}
		}
	}

	if !foundNearRight {
		t.Error("DrawRightAligned did not draw pixels near right edge")
	}
}

func TestStatusRenderer_SetColor(t *testing.T) {
	r := NewStatusRenderer("5x7")

	// Default color is white (255)
	if r.GetColor().Y != 255 {
		t.Errorf("GetColor() = %d, want 255 (default)", r.GetColor().Y)
	}

	// Change color
	r.SetColor(color.Gray{Y: 128})
	if r.GetColor().Y != 128 {
		t.Errorf("GetColor() = %d after SetColor(128), want 128", r.GetColor().Y)
	}

	// Verify color is used when drawing
	img := image.NewGray(image.Rect(0, 0, 50, 20))
	r.DrawAt(img, "X", 5, 5)

	// Check that drawn pixels have the new color
	for y := 0; y < 20; y++ {
		for x := 0; x < 50; x++ {
			pixel := img.GrayAt(x, y).Y
			if pixel > 0 && pixel != 128 {
				t.Errorf("Pixel at (%d, %d) = %d, want 128 or 0", x, y, pixel)
			}
		}
	}
}

func TestStatusRenderer_BoundsChecking(t *testing.T) {
	r := NewStatusRenderer("5x7")
	img := image.NewGray(image.Rect(0, 0, 10, 10))

	// Drawing at negative position should not panic
	r.DrawAt(img, "Test", -5, -5)

	// Drawing outside bounds should not panic
	r.DrawAt(img, "Test", 100, 100)

	// Drawing partially outside should clip correctly
	r.DrawAt(img, "X", 8, 5) // Partially outside right edge
}
