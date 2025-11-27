package glyphs

import (
	"image"
	"image/color"
	"testing"
)

func TestDrawGlyph(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		x      int
		y      int
		glyph  *Glyph
		expect bool // expect pixels to be drawn
	}{
		{
			name:   "simple glyph",
			width:  10,
			height: 10,
			x:      2,
			y:      2,
			glyph: &Glyph{
				Width:  3,
				Height: 3,
				Data: [][]bool{
					{true, false, true},
					{false, true, false},
					{true, false, true},
				},
			},
			expect: true,
		},
		{
			name:   "nil glyph",
			width:  10,
			height: 10,
			x:      0,
			y:      0,
			glyph:  nil,
			expect: false,
		},
		{
			name:   "glyph at edge",
			width:  10,
			height: 10,
			x:      8,
			y:      8,
			glyph: &Glyph{
				Width:  5,
				Height: 5,
				Data: [][]bool{
					{true, true, true, true, true},
					{true, true, true, true, true},
					{true, true, true, true, true},
					{true, true, true, true, true},
					{true, true, true, true, true},
				},
			},
			expect: true, // Should clip gracefully
		},
		{
			name:   "glyph outside bounds",
			width:  10,
			height: 10,
			x:      -5,
			y:      -5,
			glyph: &Glyph{
				Width:  3,
				Height: 3,
				Data: [][]bool{
					{true, true, true},
					{true, true, true},
					{true, true, true},
				},
			},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewGray(image.Rect(0, 0, tt.width, tt.height))
			DrawGlyph(img, tt.glyph, tt.x, tt.y, color.Gray{Y: 255})

			// Count filled pixels
			filled := 0
			for y := 0; y < tt.height; y++ {
				for x := 0; x < tt.width; x++ {
					if img.GrayAt(x, y).Y == 255 {
						filled++
					}
				}
			}

			if tt.expect && filled == 0 {
				t.Error("expected pixels to be drawn but none were")
			}

			if !tt.expect && filled > 0 {
				t.Errorf("expected no pixels but %d were drawn", filled)
			}
		})
	}
}

func TestDrawText(t *testing.T) {
	// Create a simple test glyph set
	testGlyphSet := &GlyphSet{
		Name:        "test",
		GlyphWidth:  3,
		GlyphHeight: 5,
		Glyphs: map[rune]*Glyph{
			'A': {
				Width:  3,
				Height: 5,
				Data: [][]bool{
					{false, true, false},
					{true, false, true},
					{true, true, true},
					{true, false, true},
					{true, false, true},
				},
			},
			'B': {
				Width:  3,
				Height: 5,
				Data: [][]bool{
					{true, true, false},
					{true, false, true},
					{true, true, false},
					{true, false, true},
					{true, true, false},
				},
			},
		},
	}

	tests := []struct {
		name      string
		text      string
		glyphSet  *GlyphSet
		expectLen int // expected cursor position (includes trailing spacing)
	}{
		{
			name:      "single character",
			text:      "A",
			glyphSet:  testGlyphSet,
			expectLen: 4, // 3 + 1 spacing
		},
		{
			name:      "two characters",
			text:      "AB",
			glyphSet:  testGlyphSet,
			expectLen: 8, // 3 + 1 spacing + 3 + 1 spacing
		},
		{
			name:      "empty text",
			text:      "",
			glyphSet:  testGlyphSet,
			expectLen: 0,
		},
		{
			name:      "unknown character",
			text:      "X",
			glyphSet:  testGlyphSet,
			expectLen: 0, // Unknown chars are skipped
		},
		{
			name:      "nil glyph set",
			text:      "AB",
			glyphSet:  nil,
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewGray(image.Rect(0, 0, 50, 10))
			width := DrawText(img, tt.text, 0, 0, tt.glyphSet, color.Gray{Y: 255})

			if width != tt.expectLen {
				t.Errorf("DrawText returned width %d, expected %d", width, tt.expectLen)
			}
		})
	}
}

func TestGetGlyph(t *testing.T) {
	glyphSet := &GlyphSet{
		Name:        "test",
		GlyphWidth:  5,
		GlyphHeight: 7,
		Glyphs: map[rune]*Glyph{
			'A': {Width: 5, Height: 7},
			'Z': {Width: 5, Height: 7},
		},
	}

	tests := []struct {
		name      string
		glyphSet  *GlyphSet
		char      rune
		expectNil bool
	}{
		{"existing char A", glyphSet, 'A', false},
		{"existing char Z", glyphSet, 'Z', false},
		{"missing char X", glyphSet, 'X', true},
		{"nil glyph set", nil, 'A', true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGlyph(tt.glyphSet, tt.char)

			if tt.expectNil && result != nil {
				t.Error("expected nil but got a glyph")
			}

			if !tt.expectNil && result == nil {
				t.Error("expected a glyph but got nil")
			}
		})
	}
}

func TestGetIcon(t *testing.T) {
	iconSet := &GlyphSet{
		Name:        "test_icons",
		GlyphWidth:  8,
		GlyphHeight: 8,
		Icons: map[string]*Glyph{
			"warning": {Width: 8, Height: 8},
			"check":   {Width: 8, Height: 8},
		},
	}

	tests := []struct {
		name      string
		glyphSet  *GlyphSet
		iconName  string
		expectNil bool
	}{
		{"existing icon warning", iconSet, "warning", false},
		{"existing icon check", iconSet, "check", false},
		{"missing icon error", iconSet, "error", true},
		{"nil glyph set", nil, "warning", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetIcon(tt.glyphSet, tt.iconName)

			if tt.expectNil && result != nil {
				t.Error("expected nil but got an icon")
			}

			if !tt.expectNil && result == nil {
				t.Error("expected an icon but got nil")
			}
		})
	}
}

func TestMeasureText(t *testing.T) {
	testGlyphSet := &GlyphSet{
		Name:        "test",
		GlyphWidth:  5,
		GlyphHeight: 7,
		Glyphs: map[rune]*Glyph{
			'A': {Width: 5, Height: 7},
			'B': {Width: 5, Height: 7},
			'C': {Width: 5, Height: 7},
		},
	}

	tests := []struct {
		name     string
		text     string
		glyphSet *GlyphSet
		expected int
	}{
		{"empty text", "", testGlyphSet, 0},
		{"single char", "A", testGlyphSet, 5},
		{"two chars", "AB", testGlyphSet, 11},    // 5 + 1 spacing + 5
		{"three chars", "ABC", testGlyphSet, 17}, // 5 + 1 + 5 + 1 + 5
		{"unknown char", "X", testGlyphSet, 0},
		{"nil glyph set", "ABC", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width := MeasureText(tt.text, tt.glyphSet)

			if width != tt.expected {
				t.Errorf("MeasureText returned %d, expected %d", width, tt.expected)
			}
		})
	}
}

func TestFont5x7_Coverage(t *testing.T) {
	if Font5x7 == nil {
		t.Fatal("Font5x7 is nil")
	}

	if Font5x7.Glyphs == nil {
		t.Fatal("Font5x7.Glyphs is nil")
	}

	// Test uppercase letters
	for ch := 'A'; ch <= 'Z'; ch++ {
		if glyph := GetGlyph(Font5x7, ch); glyph == nil {
			t.Errorf("Font5x7 missing uppercase letter: %c", ch)
		}
	}

	// Test digits
	for ch := '0'; ch <= '9'; ch++ {
		if glyph := GetGlyph(Font5x7, ch); glyph == nil {
			t.Errorf("Font5x7 missing digit: %c", ch)
		}
	}

	// Test common punctuation
	commonPunct := []rune{' ', '!', '.', ':', '-', '%', '/', '(', ')'}
	for _, ch := range commonPunct {
		if glyph := GetGlyph(Font5x7, ch); glyph == nil {
			t.Errorf("Font5x7 missing punctuation: %c", ch)
		}
	}
}

func TestFont5x7_GlyphDimensions(t *testing.T) {
	if Font5x7 == nil || Font5x7.Glyphs == nil {
		t.Fatal("Font5x7 not initialized")
	}

	// Check that all glyphs have correct height
	for ch, glyph := range Font5x7.Glyphs {
		if glyph.Height != 7 {
			t.Errorf("Glyph '%c' has height %d, expected 7", ch, glyph.Height)
		}

		if glyph.Width > 5 {
			t.Errorf("Glyph '%c' has width %d, expected <= 5", ch, glyph.Width)
		}

		// Verify data array dimensions match
		if len(glyph.Data) != glyph.Height {
			t.Errorf("Glyph '%c' data rows %d != height %d", ch, len(glyph.Data), glyph.Height)
		}

		for row := 0; row < len(glyph.Data); row++ {
			if len(glyph.Data[row]) != glyph.Width {
				t.Errorf("Glyph '%c' row %d has %d cols, expected %d", ch, row, len(glyph.Data[row]), glyph.Width)
			}
		}
	}
}

func TestFont3x5_Coverage(t *testing.T) {
	if Font3x5 == nil {
		t.Fatal("Font3x5 is nil")
	}

	if Font3x5.Glyphs == nil {
		t.Fatal("Font3x5.Glyphs is nil")
	}

	// Test uppercase letters
	for ch := 'A'; ch <= 'Z'; ch++ {
		if glyph := GetGlyph(Font3x5, ch); glyph == nil {
			t.Errorf("Font3x5 missing uppercase letter: %c", ch)
		}
	}

	// Test digits
	for ch := '0'; ch <= '9'; ch++ {
		if glyph := GetGlyph(Font3x5, ch); glyph == nil {
			t.Errorf("Font3x5 missing digit: %c", ch)
		}
	}
}

func TestFont3x5_GlyphDimensions(t *testing.T) {
	if Font3x5 == nil || Font3x5.Glyphs == nil {
		t.Fatal("Font3x5 not initialized")
	}

	// Check that all glyphs have correct height
	for ch, glyph := range Font3x5.Glyphs {
		if glyph.Height != 5 {
			t.Errorf("Glyph '%c' has height %d, expected 5", ch, glyph.Height)
		}

		if glyph.Width > 3 {
			t.Errorf("Glyph '%c' has width %d, expected <= 3", ch, glyph.Width)
		}

		// Verify data array dimensions match
		if len(glyph.Data) != glyph.Height {
			t.Errorf("Glyph '%c' data rows %d != height %d", ch, len(glyph.Data), glyph.Height)
		}

		for row := 0; row < len(glyph.Data); row++ {
			if len(glyph.Data[row]) != glyph.Width {
				t.Errorf("Glyph '%c' row %d has %d cols, expected %d", ch, row, len(glyph.Data[row]), glyph.Width)
			}
		}
	}
}

func TestKeyboardIcons_Existence(t *testing.T) {
	iconSets := []*GlyphSet{
		KeyboardIcons8x8,
		KeyboardIcons12x12,
		KeyboardIcons16x16,
	}

	expectedIcons := []string{"arrow_up", "arrow_down", "lock_closed", "lock_open"}

	for _, iconSet := range iconSets {
		if iconSet == nil {
			t.Errorf("Icon set is nil")
			continue
		}

		t.Run(iconSet.Name, func(t *testing.T) {
			for _, iconName := range expectedIcons {
				icon := GetIcon(iconSet, iconName)
				if icon == nil {
					t.Errorf("Icon '%s' not found in %s", iconName, iconSet.Name)
				}
			}
		})
	}
}

func TestKeyboardIcons_Dimensions(t *testing.T) {
	tests := []struct {
		iconSet      *GlyphSet
		expectedSize int
	}{
		{KeyboardIcons8x8, 8},
		{KeyboardIcons12x12, 12},
		{KeyboardIcons16x16, 16},
	}

	for _, tt := range tests {
		t.Run(tt.iconSet.Name, func(t *testing.T) {
			if tt.iconSet.GlyphWidth != tt.expectedSize {
				t.Errorf("GlyphWidth is %d, expected %d", tt.iconSet.GlyphWidth, tt.expectedSize)
			}

			if tt.iconSet.GlyphHeight != tt.expectedSize {
				t.Errorf("GlyphHeight is %d, expected %d", tt.iconSet.GlyphHeight, tt.expectedSize)
			}

			// Check each icon's dimensions
			for name, icon := range tt.iconSet.Icons {
				if icon.Width != tt.expectedSize {
					t.Errorf("Icon '%s' width is %d, expected %d", name, icon.Width, tt.expectedSize)
				}

				if icon.Height != tt.expectedSize {
					t.Errorf("Icon '%s' height is %d, expected %d", name, icon.Height, tt.expectedSize)
				}

				// Verify data array dimensions
				if len(icon.Data) != tt.expectedSize {
					t.Errorf("Icon '%s' has %d rows, expected %d", name, len(icon.Data), tt.expectedSize)
				}

				for row := 0; row < len(icon.Data); row++ {
					if len(icon.Data[row]) != tt.expectedSize {
						t.Errorf("Icon '%s' row %d has %d cols, expected %d", name, row, len(icon.Data[row]), tt.expectedSize)
					}
				}
			}
		})
	}
}

func TestCommonIcons_Existence(t *testing.T) {
	iconSets := []*GlyphSet{
		CommonIcons12x12,
		CommonIcons16x16,
		CommonIcons24x24,
	}

	for _, iconSet := range iconSets {
		if iconSet == nil {
			t.Errorf("Icon set is nil")
			continue
		}

		t.Run(iconSet.Name, func(t *testing.T) {
			icon := GetIcon(iconSet, "warning")
			if icon == nil {
				t.Errorf("Warning icon not found in %s", iconSet.Name)
			}
		})
	}
}

func TestCommonIcons_Dimensions(t *testing.T) {
	tests := []struct {
		iconSet      *GlyphSet
		expectedSize int
	}{
		{CommonIcons12x12, 12},
		{CommonIcons16x16, 16},
		{CommonIcons24x24, 24},
	}

	for _, tt := range tests {
		t.Run(tt.iconSet.Name, func(t *testing.T) {
			if tt.iconSet.GlyphWidth != tt.expectedSize {
				t.Errorf("GlyphWidth is %d, expected %d", tt.iconSet.GlyphWidth, tt.expectedSize)
			}

			if tt.iconSet.GlyphHeight != tt.expectedSize {
				t.Errorf("GlyphHeight is %d, expected %d", tt.iconSet.GlyphHeight, tt.expectedSize)
			}

			// Check warning icon dimensions
			warning := GetIcon(tt.iconSet, "warning")
			if warning == nil {
				t.Fatal("Warning icon not found")
			}

			if warning.Width != tt.expectedSize {
				t.Errorf("Warning icon width is %d, expected %d", warning.Width, tt.expectedSize)
			}

			if warning.Height != tt.expectedSize {
				t.Errorf("Warning icon height is %d, expected %d", warning.Height, tt.expectedSize)
			}

			// Verify data array dimensions
			if len(warning.Data) != tt.expectedSize {
				t.Errorf("Warning icon has %d rows, expected %d", len(warning.Data), tt.expectedSize)
			}

			for row := 0; row < len(warning.Data); row++ {
				if len(warning.Data[row]) != tt.expectedSize {
					t.Errorf("Warning icon row %d has %d cols, expected %d", row, len(warning.Data[row]), tt.expectedSize)
				}
			}
		})
	}
}

func TestDrawText_RealFont(t *testing.T) {
	tests := []struct {
		name string
		text string
		font *GlyphSet
	}{
		{"5x7 uppercase", "HELLO", Font5x7},
		{"5x7 digits", "12345", Font5x7},
		{"5x7 mixed", "A1B2", Font5x7},
		{"3x5 uppercase", "DOOM", Font3x5},
		{"3x5 digits", "100", Font3x5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewGray(image.Rect(0, 0, 100, 20))
			width := DrawText(img, tt.text, 0, 0, tt.font, color.Gray{Y: 255})

			if width == 0 {
				t.Error("DrawText returned width 0")
			}

			// Count filled pixels
			filled := 0
			for y := 0; y < 20; y++ {
				for x := 0; x < 100; x++ {
					if img.GrayAt(x, y).Y == 255 {
						filled++
					}
				}
			}

			if filled == 0 {
				t.Error("No pixels drawn")
			}
		})
	}
}

func TestDrawGlyph_Clipping(t *testing.T) {
	// Test that drawing outside bounds doesn't panic
	img := image.NewGray(image.Rect(0, 0, 10, 10))

	glyph := &Glyph{
		Width:  5,
		Height: 5,
		Data: [][]bool{
			{true, true, true, true, true},
			{true, true, true, true, true},
			{true, true, true, true, true},
			{true, true, true, true, true},
			{true, true, true, true, true},
		},
	}

	// These should all clip gracefully without panicking
	DrawGlyph(img, glyph, -10, 0, color.Gray{Y: 255})
	DrawGlyph(img, glyph, 10, 0, color.Gray{Y: 255})
	DrawGlyph(img, glyph, 0, -10, color.Gray{Y: 255})
	DrawGlyph(img, glyph, 0, 10, color.Gray{Y: 255})
	DrawGlyph(img, glyph, 8, 8, color.Gray{Y: 255}) // Partial overlap
}
