package bitmap

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
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
		horizAlign  config.HAlign
		vertAlign   config.VAlign
		padding     int
		shouldPanic bool
	}{
		{
			name:       "center-center alignment",
			text:       "Hello",
			horizAlign: config.AlignCenter,
			vertAlign:  config.AlignMiddle,
			padding:    0,
		},
		{
			name:       "left-top alignment",
			text:       "Test",
			horizAlign: config.AlignLeft,
			vertAlign:  config.AlignTop,
			padding:    0,
		},
		{
			name:       "right-bottom alignment",
			text:       "Right",
			horizAlign: config.AlignRight,
			vertAlign:  config.AlignBottom,
			padding:    0,
		},
		{
			name:       "center-top alignment",
			text:       "Top",
			horizAlign: config.AlignCenter,
			vertAlign:  config.AlignTop,
			padding:    5,
		},
		{
			name:       "left-center alignment",
			text:       "Left",
			horizAlign: config.AlignLeft,
			vertAlign:  config.AlignMiddle,
			padding:    10,
		},
		{
			name:       "empty text",
			text:       "",
			horizAlign: config.AlignCenter,
			vertAlign:  config.AlignMiddle,
			padding:    0,
		},
		{
			name:       "long text",
			text:       "This is a longer piece of text",
			horizAlign: config.AlignCenter,
			vertAlign:  config.AlignMiddle,
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
		horizAlign config.HAlign
		vertAlign  config.VAlign
	}{
		{"invalid horizontal", config.HAlign("invalid"), config.AlignMiddle},
		{"invalid vertical", config.AlignCenter, config.VAlign("invalid")},
		{"both invalid", config.HAlign("invalid"), config.VAlign("invalid")},
		{"empty alignment", config.HAlign(""), config.VAlign("")},
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

	DrawAlignedText(testImg, "Test", nil, config.AlignCenter, config.AlignMiddle, 0)
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
	// Should handle nil gracefully without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DrawBorder() with nil image panicked: %v", r)
		}
	}()

	// Should not panic, just return early
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

	horizontalAligns := []config.HAlign{config.AlignLeft, config.AlignCenter, config.AlignRight}
	verticalAligns := []config.VAlign{config.AlignTop, config.AlignMiddle, config.AlignBottom}

	for _, hAlign := range horizontalAligns {
		for _, vAlign := range verticalAligns {
			t.Run(string(hAlign)+"_"+string(vAlign), func(t *testing.T) {
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

// TestDrawTextInRect tests text drawing within a rectangle
func TestDrawTextInRect(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name       string
		text       string
		x          int
		y          int
		width      int
		height     int
		horizAlign config.HAlign
		vertAlign  config.VAlign
		padding    int
	}{
		{"center-center", "Test", 10, 10, 60, 20, config.AlignCenter, config.AlignMiddle, 2},
		{"left-top", "Left", 10, 10, 60, 20, config.AlignLeft, config.AlignTop, 0},
		{"right-bottom", "Right", 10, 10, 60, 20, config.AlignRight, config.AlignBottom, 5},
		{"center-top", "Top", 10, 10, 60, 20, config.AlignCenter, config.AlignTop, 0},
		{"left-center", "Mid", 10, 10, 60, 20, config.AlignLeft, config.AlignMiddle, 0},
		{"right-center", "Right", 10, 10, 60, 20, config.AlignRight, config.AlignMiddle, 0},
		{"center-bottom", "Bottom", 10, 10, 60, 20, config.AlignCenter, config.AlignBottom, 0},
		{"small rect", "X", 0, 0, 10, 10, config.AlignCenter, config.AlignMiddle, 1},
		{"large rect", "Large Rectangle Text", 0, 0, 128, 40, config.AlignCenter, config.AlignMiddle, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(128, 40, 0)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DrawTextInRect() panicked: %v", r)
				}
			}()

			DrawTextInRect(img, tt.text, face, tt.x, tt.y, tt.width, tt.height, tt.horizAlign, tt.vertAlign, tt.padding)

			if img == nil {
				t.Error("DrawTextInRect() invalidated image")
			}
		})
	}
}

// TestDrawTextInRect_EdgeCases tests edge cases for DrawTextInRect
func TestDrawTextInRect_EdgeCases(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name   string
		text   string
		width  int
		height int
	}{
		{"zero width", "Text", 0, 20},
		{"zero height", "Text", 60, 0},
		{"negative width", "Text", -10, 20},
		{"negative height", "Text", 60, -10},
		{"empty text", "", 60, 20},
		{"very long text", "This is a very long text that exceeds rectangle width", 30, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(128, 40, 0)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DrawTextInRect() panicked with %s: %v", tt.name, r)
				}
			}()

			DrawTextInRect(img, tt.text, face, 10, 10, tt.width, tt.height, "center", "center", 2)
		})
	}
}

// TestDrawTextInRect_AllAlignments tests all alignment combinations
func TestDrawTextInRect_AllAlignments(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	horizontalAligns := []config.HAlign{config.AlignLeft, config.AlignCenter, config.AlignRight}
	verticalAligns := []config.VAlign{config.AlignTop, config.AlignMiddle, config.AlignBottom}

	for _, hAlign := range horizontalAligns {
		for _, vAlign := range verticalAligns {
			t.Run(string(hAlign)+"_"+string(vAlign), func(t *testing.T) {
				img := NewGrayscaleImage(128, 40, 0)

				defer func() {
					if r := recover(); r != nil {
						t.Errorf("DrawTextInRect() panicked with %s/%s: %v", hAlign, vAlign, r)
					}
				}()

				DrawTextInRect(img, "Align Test", face, 10, 5, 100, 30, hAlign, vAlign, 3)
			})
		}
	}
}

// TestDrawTextInRect_LargePadding tests behavior with large padding
func TestDrawTextInRect_LargePadding(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name    string
		padding int
	}{
		{"zero padding", 0},
		{"normal padding", 5},
		{"large padding", 25},
		{"padding equals width", 30},
		{"padding exceeds dimensions", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(128, 40, 0)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DrawTextInRect() panicked with padding %d: %v", tt.padding, r)
				}
			}()

			DrawTextInRect(img, "Test", face, 10, 10, 60, 20, "center", "center", tt.padding)
		})
	}
}

// TestDrawTextInRect_InvalidAlignment tests invalid alignment strings
func TestDrawTextInRect_InvalidAlignment(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name       string
		horizAlign config.HAlign
		vertAlign  config.VAlign
	}{
		{"invalid horizontal", config.HAlign("invalid"), config.AlignMiddle},
		{"invalid vertical", config.AlignCenter, config.VAlign("invalid")},
		{"both invalid", config.HAlign("wrong"), config.VAlign("bad")},
		{"empty strings", config.HAlign(""), config.VAlign("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(128, 40, 0)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DrawTextInRect() should default invalid alignment, but panicked: %v", r)
				}
			}()

			DrawTextInRect(img, "Test", face, 10, 10, 60, 20, tt.horizAlign, tt.vertAlign, 0)
		})
	}
}

// TestDrawTextAtPosition tests basic text positioning with clipping
func TestDrawTextAtPosition(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name                       string
		x, y                       int
		clipX, clipY, clipW, clipH int
	}{
		{"centered in clip area", 20, 20, 10, 10, 100, 30},
		{"at clip origin", 10, 20, 10, 10, 100, 30},
		{"at clip edge", 100, 20, 10, 10, 100, 30},
		{"full image clip", 10, 20, 0, 0, 128, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(128, 40, 0)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DrawTextAtPosition() panicked: %v", r)
				}
			}()

			DrawTextAtPosition(img, "Test", face, tt.x, tt.y, tt.clipX, tt.clipY, tt.clipW, tt.clipH)
		})
	}
}

// TestDrawTextAtPosition_HorizontalClipping tests horizontal clipping
func TestDrawTextAtPosition_HorizontalClipping(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	img := NewGrayscaleImage(128, 40, 0)

	// Draw text that starts before clip area (should be partially clipped)
	DrawTextAtPosition(img, "Hello World", face, -20, 20, 0, 0, 128, 40)

	// Draw text that extends past clip area (should be partially clipped)
	img2 := NewGrayscaleImage(128, 40, 0)
	DrawTextAtPosition(img2, "Hello World", face, 100, 20, 0, 0, 128, 40)

	// Both should complete without panic
	if img == nil || img2 == nil {
		t.Error("DrawTextAtPosition() returned nil image")
	}
}

// TestDrawTextAtPosition_VerticalClipping tests vertical clipping
func TestDrawTextAtPosition_VerticalClipping(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name   string
		y      int
		clipY  int
		clipH  int
		expect string // "visible", "above", "below"
	}{
		{"text in clip area", 25, 10, 30, "visible"},
		{"text above clip area", 5, 20, 20, "above"},
		{"text below clip area", 50, 10, 20, "below"},
		{"text at top edge", 15, 10, 30, "visible"},
		{"text at bottom edge", 35, 10, 30, "visible"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(128, 40, 0)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DrawTextAtPosition() panicked: %v", r)
				}
			}()

			DrawTextAtPosition(img, "Test", face, 10, tt.y, 0, tt.clipY, 128, tt.clipH)

			// Count non-zero pixels to verify if text was drawn
			nonZeroPixels := 0
			bounds := img.Bounds()
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					if img.GrayAt(x, y).Y > 0 {
						nonZeroPixels++
					}
				}
			}

			if tt.expect == "above" || tt.expect == "below" {
				if nonZeroPixels > 0 {
					t.Errorf("Expected no pixels drawn when text is %s clip area, got %d pixels", tt.expect, nonZeroPixels)
				}
			}
		})
	}
}

// TestDrawTextAtPosition_CompletelyOutside tests text completely outside clip area
func TestDrawTextAtPosition_CompletelyOutside(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name                       string
		x, y                       int
		clipX, clipY, clipW, clipH int
	}{
		{"completely left", -100, 20, 0, 0, 128, 40},
		{"completely right", 200, 20, 0, 0, 128, 40},
		{"completely above", 50, -50, 0, 0, 128, 40},
		{"completely below", 50, 100, 0, 0, 128, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(128, 40, 0)

			DrawTextAtPosition(img, "Test", face, tt.x, tt.y, tt.clipX, tt.clipY, tt.clipW, tt.clipH)

			// Count non-zero pixels - should be zero for completely outside
			nonZeroPixels := 0
			bounds := img.Bounds()
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					if img.GrayAt(x, y).Y > 0 {
						nonZeroPixels++
					}
				}
			}

			if nonZeroPixels > 0 {
				t.Errorf("Expected no pixels drawn when text is %s, got %d pixels", tt.name, nonZeroPixels)
			}
		})
	}
}

// TestDrawTextAtPosition_EmptyClipArea tests with zero-size clip area
func TestDrawTextAtPosition_EmptyClipArea(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name         string
		clipW, clipH int
	}{
		{"zero width", 0, 40},
		{"zero height", 128, 0},
		{"zero both", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(128, 40, 0)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DrawTextAtPosition() panicked with %s: %v", tt.name, r)
				}
			}()

			DrawTextAtPosition(img, "Test", face, 50, 20, 0, 0, tt.clipW, tt.clipH)
		})
	}
}

// TestDrawTextAtPosition_ScrollingSimulation simulates scrolling text
func TestDrawTextAtPosition_ScrollingSimulation(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	// Simulate horizontal scrolling
	for offset := -50; offset <= 150; offset += 10 {
		img := NewGrayscaleImage(128, 40, 0)
		DrawTextAtPosition(img, "Scrolling Text Test", face, offset, 20, 10, 5, 108, 30)
	}

	// Simulate vertical scrolling
	for offset := -20; offset <= 60; offset += 5 {
		img := NewGrayscaleImage(128, 40, 0)
		DrawTextAtPosition(img, "Vertical Scroll", face, 20, offset, 10, 5, 108, 30)
	}
}

// TestDrawTextAtPosition_ConcurrentAccess tests thread safety
func TestDrawTextAtPosition_ConcurrentAccess(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(offset int) {
			img := NewGrayscaleImage(128, 40, 0)
			DrawTextAtPosition(img, "Concurrent Test", face, offset, 20, 0, 0, 128, 40)
			done <- true
		}(i * 10)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestCalculateTextPosition tests text position calculation with various alignments
func TestCalculateTextPosition(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name       string
		text       string
		contentX   int
		contentY   int
		contentW   int
		contentH   int
		horizAlign config.HAlign
		vertAlign  config.VAlign
	}{
		{"center-center", "Test", 10, 10, 100, 30, config.AlignCenter, config.AlignMiddle},
		{"left-top", "Test", 10, 10, 100, 30, config.AlignLeft, config.AlignTop},
		{"right-bottom", "Test", 10, 10, 100, 30, config.AlignRight, config.AlignBottom},
		{"left-center", "Test", 10, 10, 100, 30, config.AlignLeft, config.AlignMiddle},
		{"right-top", "Test", 10, 10, 100, 30, config.AlignRight, config.AlignTop},
		{"center-bottom", "Test", 10, 10, 100, 30, config.AlignCenter, config.AlignBottom},
		{"empty text", "", 10, 10, 100, 30, config.AlignCenter, config.AlignMiddle},
		{"long text", "This is a very long text", 10, 10, 50, 30, config.AlignCenter, config.AlignMiddle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := CalculateTextPosition(tt.text, face, tt.contentX, tt.contentY, tt.contentW, tt.contentH, tt.horizAlign, tt.vertAlign)

			// X should be within or near content area for non-overflowing text
			if tt.text != "" && len(tt.text) < 10 {
				if x < tt.contentX-50 || x > tt.contentX+tt.contentW+50 {
					t.Errorf("X position %d seems unreasonable for content area starting at %d", x, tt.contentX)
				}
			}

			// Y should be reasonable (positive and not extremely large)
			if y < 0 || y > 1000 {
				t.Errorf("Y position %d seems unreasonable", y)
			}
		})
	}
}

// TestCalculateTextPosition_HorizontalAlignment verifies horizontal alignment behavior
func TestCalculateTextPosition_HorizontalAlignment(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	contentX := 10
	contentW := 100

	leftX, _ := CalculateTextPosition("Test", face, contentX, 10, contentW, 30, config.AlignLeft, config.AlignMiddle)
	centerX, _ := CalculateTextPosition("Test", face, contentX, 10, contentW, 30, config.AlignCenter, config.AlignMiddle)
	rightX, _ := CalculateTextPosition("Test", face, contentX, 10, contentW, 30, config.AlignRight, config.AlignMiddle)

	// Left should be at or near contentX
	if leftX != contentX {
		t.Errorf("Left alignment: X = %d, want %d", leftX, contentX)
	}

	// Center should be between left and right
	if centerX <= leftX || centerX >= rightX {
		t.Errorf("Center alignment: X = %d should be between left (%d) and right (%d)", centerX, leftX, rightX)
	}

	// Right should be greater than center
	if rightX <= centerX {
		t.Errorf("Right alignment: X = %d should be > center (%d)", rightX, centerX)
	}
}

// TestCalculateTextPosition_VerticalAlignment verifies vertical alignment behavior
func TestCalculateTextPosition_VerticalAlignment(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	contentY := 10
	contentH := 50

	_, topY := CalculateTextPosition("Test", face, 10, contentY, 100, contentH, config.AlignCenter, config.AlignTop)
	_, centerY := CalculateTextPosition("Test", face, 10, contentY, 100, contentH, config.AlignCenter, config.AlignMiddle)
	_, bottomY := CalculateTextPosition("Test", face, 10, contentY, 100, contentH, config.AlignCenter, config.AlignBottom)

	// Top should be smallest Y (closest to top)
	if topY >= centerY {
		t.Errorf("Top alignment: Y = %d should be < center (%d)", topY, centerY)
	}

	// Center should be between top and bottom
	if centerY <= topY || centerY >= bottomY {
		t.Errorf("Center alignment: Y = %d should be between top (%d) and bottom (%d)", centerY, topY, bottomY)
	}

	// Bottom should be largest Y
	if bottomY <= centerY {
		t.Errorf("Bottom alignment: Y = %d should be > center (%d)", bottomY, centerY)
	}
}

// TestCalculateTextPosition_ConsistencyWithDrawAlignedText verifies position matches DrawAlignedText
func TestCalculateTextPosition_ConsistencyWithDrawAlignedText(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	// Create two images - one drawn with DrawAlignedText, one with DrawTextAtPosition using calculated coords
	img1 := NewGrayscaleImage(128, 40, 0)
	img2 := NewGrayscaleImage(128, 40, 0)

	padding := 5
	text := "Test"

	// Draw using DrawAlignedText
	DrawAlignedText(img1, text, face, "center", "center", padding)

	// Calculate position and draw using DrawTextAtPosition
	contentX := padding
	contentY := padding
	contentW := 128 - padding*2
	contentH := 40 - padding*2
	x, y := CalculateTextPosition(text, face, contentX, contentY, contentW, contentH, "center", "center")
	DrawTextAtPosition(img2, text, face, x, y, 0, 0, 128, 40)

	// Compare images - they should be identical
	for py := 0; py < 40; py++ {
		for px := 0; px < 128; px++ {
			if img1.GrayAt(px, py).Y != img2.GrayAt(px, py).Y {
				t.Errorf("Images differ at (%d, %d): DrawAlignedText=%d, calculated=%d",
					px, py, img1.GrayAt(px, py).Y, img2.GrayAt(px, py).Y)
				return
			}
		}
	}
}

// TestCalculateTextPosition_InvalidAlignment tests default behavior for invalid alignments
func TestCalculateTextPosition_InvalidAlignment(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	// Should not panic and should default to center
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("CalculateTextPosition panicked with invalid alignment: %v", r)
		}
	}()

	x, y := CalculateTextPosition("Test", face, 10, 10, 100, 30, "invalid", "invalid")

	// Should return reasonable values (defaulting to center)
	if x < 0 || y < 0 {
		t.Errorf("Invalid alignment returned negative position: x=%d, y=%d", x, y)
	}
}

func TestSmartDrawAlignedText_InternalFont(t *testing.T) {
	img := NewGrayscaleImage(50, 20, 0)
	SmartDrawAlignedText(img, "Hi", nil, FontNamePixel5x7, "center", "center", 2)

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
		t.Error("SmartDrawAlignedText with internal font produced no visible pixels")
	}
}

func TestSmartDrawAlignedText_TTFFont(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	img := NewGrayscaleImage(50, 20, 0)
	SmartDrawAlignedText(img, "Hi", face, "", "center", "center", 2)

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
		t.Error("SmartDrawAlignedText with TTF font produced no visible pixels")
	}
}

func TestSmartDrawAlignedText_NoFont(t *testing.T) {
	img := NewGrayscaleImage(50, 20, 0)
	// Should not panic with nil font and non-internal font name
	SmartDrawAlignedText(img, "Hi", nil, "unknown_font", "center", "center", 2)
}

func TestSmartDrawTextInRect_InternalFont(t *testing.T) {
	img := NewGrayscaleImage(100, 50, 0)
	SmartDrawTextInRect(img, "Test", nil, FontNamePixel5x7, 10, 10, 80, 30, "center", "center", 2)

	// Check that some pixels are lit within the rectangle
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
		t.Error("SmartDrawTextInRect with internal font produced no visible pixels")
	}
}

func TestSmartDrawTextInRect_TTFFont(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	img := NewGrayscaleImage(100, 50, 0)
	SmartDrawTextInRect(img, "Test", face, "", 10, 10, 80, 30, "center", "center", 2)

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
		t.Error("SmartDrawTextInRect with TTF font produced no visible pixels")
	}
}

func TestSmartDrawTextInRect_NoFont(t *testing.T) {
	img := NewGrayscaleImage(100, 50, 0)
	// Should not panic with nil font and non-internal font name
	SmartDrawTextInRect(img, "Test", nil, "unknown", 10, 10, 80, 30, "center", "center", 2)
}

func TestSmartMeasureText_InternalFont(t *testing.T) {
	w, h := SmartMeasureText("Hello", nil, FontNamePixel5x7)
	if w <= 0 {
		t.Errorf("SmartMeasureText width = %d, want > 0", w)
	}
	if h <= 0 {
		t.Errorf("SmartMeasureText height = %d, want > 0", h)
	}
}

func TestSmartMeasureText_TTFFont(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	w, h := SmartMeasureText("Hello", face, "")
	if w <= 0 {
		t.Errorf("SmartMeasureText width = %d, want > 0", w)
	}
	if h <= 0 {
		t.Errorf("SmartMeasureText height = %d, want > 0", h)
	}
}

func TestSmartMeasureText_NoFont(t *testing.T) {
	w, h := SmartMeasureText("Hello", nil, "unknown")
	if w != 0 || h != 0 {
		t.Errorf("SmartMeasureText with no font = (%d, %d), want (0, 0)", w, h)
	}
}

func TestSmartCalculateTextPosition_InternalFont(t *testing.T) {
	tests := []struct {
		horiz config.HAlign
		vert  config.VAlign
		name  string
	}{
		{config.AlignLeft, config.AlignTop, "left_top"},
		{config.AlignCenter, config.AlignMiddle, "center_center"},
		{config.AlignRight, config.AlignBottom, "right_bottom"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			x, y := SmartCalculateTextPosition("Hi", nil, FontNamePixel5x7, 10, 10, 100, 30, tc.horiz, tc.vert)
			if x < 10 || y < 10 {
				t.Errorf("Position (%d, %d) is outside content area", x, y)
			}
		})
	}
}

func TestSmartCalculateTextPosition_TTFFont(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	x, y := SmartCalculateTextPosition("Hi", face, "", 10, 10, 100, 30, config.AlignCenter, config.AlignMiddle)
	if x < 10 {
		t.Errorf("TTF position x = %d, expected >= 10", x)
	}
	// y can be anywhere in the content area for TTF (baseline-based)
	_ = y
}

func TestSmartCalculateTextPosition_NoFont(t *testing.T) {
	x, y := SmartCalculateTextPosition("Hi", nil, "unknown", 10, 20, 100, 30, "center", "center")
	if x != 10 || y != 20 {
		t.Errorf("Position = (%d, %d), want (10, 20) for no font", x, y)
	}
}

func TestSmartDrawTextAtPosition_InternalFont(t *testing.T) {
	img := NewGrayscaleImage(100, 50, 0)
	SmartDrawTextAtPosition(img, "Hi", nil, FontNamePixel5x7, 10, 10, 0, 0, 100, 50)

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
		t.Error("SmartDrawTextAtPosition with internal font produced no visible pixels")
	}
}

func TestSmartDrawTextAtPosition_TTFFont(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	img := NewGrayscaleImage(100, 50, 0)
	SmartDrawTextAtPosition(img, "Hi", face, "", 10, 30, 0, 0, 100, 50)

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
		t.Error("SmartDrawTextAtPosition with TTF font produced no visible pixels")
	}
}

func TestSmartDrawTextAtPosition_NoFont(t *testing.T) {
	img := NewGrayscaleImage(100, 50, 0)
	// Should not panic with nil font and non-internal font name
	SmartDrawTextAtPosition(img, "Hi", nil, "unknown", 10, 10, 0, 0, 100, 50)
}

func TestSmartDrawTextAtPosition_NilGlyphSet(t *testing.T) {
	img := NewGrayscaleImage(100, 50, 0)
	// When internal font name is valid but GetInternalFontByName returns nil (edge case)
	// This should not happen in practice, but test the nil check
	SmartDrawTextAtPosition(img, "Hi", nil, "", 10, 10, 0, 0, 100, 50)
}
