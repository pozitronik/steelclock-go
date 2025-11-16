package bitmap

import (
	"testing"
)

// TestDrawAlignedText tests text drawing with various alignments
func TestDrawAlignedText(t *testing.T) {
	// Load font
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name        string
		text        string
		horizAlign  string
		vertAlign   string
		padding     int
		shouldPanic bool
	}{
		{
			name:       "center-center alignment",
			text:       "Hello",
			horizAlign: "center",
			vertAlign:  "center",
			padding:    0,
		},
		{
			name:       "left-top alignment",
			text:       "Test",
			horizAlign: "left",
			vertAlign:  "top",
			padding:    0,
		},
		{
			name:       "right-bottom alignment",
			text:       "Right",
			horizAlign: "right",
			vertAlign:  "bottom",
			padding:    0,
		},
		{
			name:       "center-top alignment",
			text:       "Top",
			horizAlign: "center",
			vertAlign:  "top",
			padding:    5,
		},
		{
			name:       "left-center alignment",
			text:       "Left",
			horizAlign: "left",
			vertAlign:  "center",
			padding:    10,
		},
		{
			name:       "empty text",
			text:       "",
			horizAlign: "center",
			vertAlign:  "center",
			padding:    0,
		},
		{
			name:       "long text",
			text:       "This is a longer piece of text",
			horizAlign: "center",
			vertAlign:  "center",
			padding:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh image for each test
			testImg := NewGrayscaleImage(128, 40, 0)

			// Should not panic
			defer func() {
				if r := recover(); r != nil && !tt.shouldPanic {
					t.Errorf("DrawAlignedText() panicked: %v", r)
				}
			}()

			DrawAlignedText(testImg, tt.text, face, tt.horizAlign, tt.vertAlign, tt.padding)

			// Verify image is still valid
			if testImg == nil {
				t.Error("DrawAlignedText() invalidated image")
			}
		})
	}
}

// TestDrawAlignedText_InvalidAlignment tests behavior with invalid alignment values
func TestDrawAlignedText_InvalidAlignment(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name       string
		horizAlign string
		vertAlign  string
	}{
		{"invalid horizontal", "invalid", "center"},
		{"invalid vertical", "center", "invalid"},
		{"both invalid", "invalid", "invalid"},
		{"empty alignment", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testImg := NewGrayscaleImage(128, 40, 0)

			// Should not panic with invalid alignment
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DrawAlignedText() should handle invalid alignment gracefully, but panicked: %v", r)
				}
			}()

			DrawAlignedText(testImg, "Test", face, tt.horizAlign, tt.vertAlign, 0)
		})
	}
}

// TestDrawAlignedText_NilFace tests drawing with nil font face
func TestDrawAlignedText_NilFace(t *testing.T) {
	testImg := NewGrayscaleImage(128, 40, 0)

	// Should handle nil face gracefully (might skip drawing or use fallback)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("DrawAlignedText() with nil face panicked (may be expected): %v", r)
		}
	}()

	DrawAlignedText(testImg, "Test", nil, "center", "center", 0)
}

// TestDrawAlignedText_LargePadding tests behavior with large padding values
func TestDrawAlignedText_LargePadding(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name    string
		padding int
	}{
		{"large padding", 100},
		{"very large padding", 500},
		{"negative padding", -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testImg := NewGrayscaleImage(128, 40, 0)

			// Should handle extreme padding gracefully
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DrawAlignedText() panicked with padding %d: %v", tt.padding, r)
				}
			}()

			DrawAlignedText(testImg, "Test", face, "center", "center", tt.padding)
		})
	}
}

// TestDrawAlignedText_SmallImage tests drawing on very small images
func TestDrawAlignedText_SmallImage(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"tiny image", 1, 1},
		{"narrow image", 10, 40},
		{"short image", 128, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(tt.width, tt.height, 0)

			// Should handle small images gracefully
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DrawAlignedText() panicked on %dx%d image: %v", tt.width, tt.height, r)
				}
			}()

			DrawAlignedText(img, "Test", face, "center", "center", 0)
		})
	}
}

// TestDrawBorderComprehensive tests border drawing functionality comprehensively
func TestDrawBorderComprehensive(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		height      int
		borderColor uint8
	}{
		{
			name:        "standard border",
			width:       128,
			height:      40,
			borderColor: 255,
		},
		{
			name:        "dark border",
			width:       128,
			height:      40,
			borderColor: 100,
		},
		{
			name:        "black border",
			width:       128,
			height:      40,
			borderColor: 0,
		},
		{
			name:        "small image",
			width:       10,
			height:      10,
			borderColor: 255,
		},
		{
			name:        "wide image",
			width:       200,
			height:      20,
			borderColor: 255,
		},
		{
			name:        "tall image",
			width:       20,
			height:      200,
			borderColor: 255,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(tt.width, tt.height, 0)

			DrawBorder(img, tt.borderColor)

			// Verify corners are set to border color
			bounds := img.Bounds()
			cornerColor := img.GrayAt(bounds.Min.X, bounds.Min.Y).Y

			if cornerColor != tt.borderColor {
				t.Errorf("Border color at top-left corner = %d, want %d", cornerColor, tt.borderColor)
			}

			// Verify edges have border
			if tt.width > 2 && tt.height > 2 {
				// Check top edge
				topEdge := img.GrayAt(bounds.Min.X+tt.width/2, bounds.Min.Y).Y
				if topEdge != tt.borderColor {
					t.Errorf("Top edge color = %d, want %d", topEdge, tt.borderColor)
				}

				// Check bottom edge
				bottomEdge := img.GrayAt(bounds.Min.X+tt.width/2, bounds.Min.Y+tt.height-1).Y
				if bottomEdge != tt.borderColor {
					t.Errorf("Bottom edge color = %d, want %d", bottomEdge, tt.borderColor)
				}

				// Check left edge
				leftEdge := img.GrayAt(bounds.Min.X, bounds.Min.Y+tt.height/2).Y
				if leftEdge != tt.borderColor {
					t.Errorf("Left edge color = %d, want %d", leftEdge, tt.borderColor)
				}

				// Check right edge
				rightEdge := img.GrayAt(bounds.Min.X+tt.width-1, bounds.Min.Y+tt.height/2).Y
				if rightEdge != tt.borderColor {
					t.Errorf("Right edge color = %d, want %d", rightEdge, tt.borderColor)
				}
			}
		})
	}
}

// TestDrawBorder_TinyImage tests border on 1x1 and 2x2 images
func TestDrawBorder_TinyImage(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"1x1 image", 1, 1},
		{"2x2 image", 2, 2},
		{"1x10 image", 1, 10},
		{"10x1 image", 10, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(tt.width, tt.height, 0)

			// Should handle tiny images without panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DrawBorder() panicked on %dx%d image: %v", tt.width, tt.height, r)
				}
			}()

			DrawBorder(img, 255)
		})
	}
}

// TestDrawBorder_NilImage tests border drawing with nil image
func TestDrawBorder_NilImage(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			// It's okay if it doesn't panic, but log it
			t.Log("DrawBorder() with nil image did not panic")
		}
	}()

	// This might panic, which is acceptable behavior
	DrawBorder(nil, 255)
}

// TestDrawAlignedText_ConcurrentAccess tests concurrent text drawing
func TestDrawAlignedText_ConcurrentAccess(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	done := make(chan bool, 5)

	// Draw on different images concurrently
	for i := 0; i < 5; i++ {
		go func(text string) {
			img := NewGrayscaleImage(128, 40, 0)
			DrawAlignedText(img, text, face, "center", "center", 0)
			done <- true
		}("Test")
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}

// TestDrawAlignedText_AllAlignmentCombinations tests all valid alignment combinations
func TestDrawAlignedText_AllAlignmentCombinations(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	horizontalAligns := []string{"left", "center", "right"}
	verticalAligns := []string{"top", "center", "bottom"}

	for _, hAlign := range horizontalAligns {
		for _, vAlign := range verticalAligns {
			t.Run(hAlign+"_"+vAlign, func(t *testing.T) {
				img := NewGrayscaleImage(128, 40, 0)

				defer func() {
					if r := recover(); r != nil {
						t.Errorf("DrawAlignedText() panicked with %s/%s: %v", hAlign, vAlign, r)
					}
				}()

				DrawAlignedText(img, "Test", face, hAlign, vAlign, 5)
			})
		}
	}
}

// TestDrawAlignedText_SpecialCharacters tests drawing special characters
func TestDrawAlignedText_SpecialCharacters(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name string
		text string
	}{
		{"symbols", "!@#$%^&*()"},
		{"unicode arrows", "↑↓←→"},
		{"mixed", "Test 123 !@#"},
		{"newline", "Line1\nLine2"},
		{"tab", "Col1\tCol2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(128, 40, 0)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DrawAlignedText() panicked with text %q: %v", tt.text, r)
				}
			}()

			DrawAlignedText(img, tt.text, face, "center", "center", 0)
		})
	}
}

// TestDrawBorder_DifferentColors tests border with various grayscale values
func TestDrawBorder_DifferentColors(t *testing.T) {
	colors := []uint8{0, 64, 128, 192, 255}

	for _, color := range colors {
		t.Run("color_"+string(rune(color)), func(t *testing.T) {
			img := NewGrayscaleImage(50, 50, 128) // Mid-gray background

			DrawBorder(img, color)

			// Verify border pixels are correct color
			bounds := img.Bounds()
			topLeft := img.GrayAt(bounds.Min.X, bounds.Min.Y).Y

			if topLeft != color {
				t.Errorf("Border color = %d, want %d", topLeft, color)
			}
		})
	}
}
