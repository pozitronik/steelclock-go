package shared

import (
	"image"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
)

func createTestRenderer(t *testing.T, maxWidth int) *MultiLineRenderer {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	cfg := MultiLineRendererConfig{
		FontFace:      fontFace,
		FontName:      "",
		HorizAlign:    config.AlignLeft,
		VertAlign:     config.AlignTop,
		ScrollMode:    ScrollContinuous,
		ScrollGap:     10,
		ScrollEnabled: true,
		WordBreak:     "normal",
	}

	return NewMultiLineRenderer(cfg, maxWidth)
}

func TestNewMultiLineRenderer(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	t.Run("creates with default word break mode", func(t *testing.T) {
		cfg := MultiLineRendererConfig{
			FontFace:      fontFace,
			FontName:      "",
			ScrollMode:    ScrollContinuous,
			ScrollEnabled: true,
			WordBreak:     "",
		}

		r := NewMultiLineRenderer(cfg, 100)
		if r == nil {
			t.Fatal("NewMultiLineRenderer returned nil")
		}
		if r.wrapper == nil {
			t.Error("wrapper should be initialized")
		}
	})

	t.Run("creates with break-all mode", func(t *testing.T) {
		cfg := MultiLineRendererConfig{
			FontFace:      fontFace,
			FontName:      "",
			ScrollEnabled: true,
			WordBreak:     "break-all",
		}

		r := NewMultiLineRenderer(cfg, 100)
		if r == nil {
			t.Fatal("NewMultiLineRenderer returned nil")
		}
	})

	t.Run("sets configuration correctly", func(t *testing.T) {
		cfg := MultiLineRendererConfig{
			FontFace:      fontFace,
			FontName:      "test",
			HorizAlign:    config.AlignCenter,
			VertAlign:     config.AlignMiddle,
			ScrollMode:    ScrollBounce,
			ScrollGap:     20,
			ScrollEnabled: true,
		}

		r := NewMultiLineRenderer(cfg, 150)
		if r.fontName != "test" {
			t.Errorf("Expected fontName 'test', got %q", r.fontName)
		}
		if r.horizAlign != config.AlignCenter {
			t.Errorf("Expected horizAlign center, got %v", r.horizAlign)
		}
		if r.vertAlign != config.AlignMiddle {
			t.Errorf("Expected vertAlign middle, got %v", r.vertAlign)
		}
		if r.scrollMode != ScrollBounce {
			t.Errorf("Expected scrollMode bounce, got %q", r.scrollMode)
		}
		if r.scrollGap != 20 {
			t.Errorf("Expected scrollGap 20, got %d", r.scrollGap)
		}
	})
}

func TestMultiLineRenderer_SetMaxWidth(t *testing.T) {
	r := createTestRenderer(t, 100)

	r.SetMaxWidth(200)
	// Verify by wrapping text
	lines := r.WrapText("Short")
	if len(lines) != 1 {
		t.Errorf("Expected 1 line for short text, got %d", len(lines))
	}
}

func TestMultiLineRenderer_MeasureLineHeight(t *testing.T) {
	r := createTestRenderer(t, 100)

	height := r.MeasureLineHeight()
	if height <= 0 {
		t.Errorf("Expected positive line height, got %d", height)
	}
}

func TestMultiLineRenderer_MeasureTotalHeight(t *testing.T) {
	r := createTestRenderer(t, 100)

	t.Run("single line", func(t *testing.T) {
		height := r.MeasureTotalHeight("Hello", 200)
		lineHeight := r.MeasureLineHeight()
		if height != lineHeight {
			t.Errorf("Expected %d for single line, got %d", lineHeight, height)
		}
	})

	t.Run("multiple lines", func(t *testing.T) {
		height := r.MeasureTotalHeight("Line1\nLine2\nLine3", 200)
		lineHeight := r.MeasureLineHeight()
		expected := lineHeight * 3
		if height != expected {
			t.Errorf("Expected %d for 3 lines, got %d", expected, height)
		}
	})
}

func TestMultiLineRenderer_WrapText(t *testing.T) {
	r := createTestRenderer(t, 50)

	t.Run("empty text", func(t *testing.T) {
		lines := r.WrapText("")
		if lines != nil {
			t.Errorf("Expected nil for empty text, got %v", lines)
		}
	})

	t.Run("text with newline", func(t *testing.T) {
		lines := r.WrapText("A\nB")
		if len(lines) != 2 {
			t.Errorf("Expected 2 lines, got %d: %v", len(lines), lines)
		}
	})
}

func TestMultiLineRenderer_Render(t *testing.T) {
	r := createTestRenderer(t, 100)

	// Create test image
	img := image.NewGray(image.Rect(0, 0, 128, 40))
	bounds := image.Rect(0, 0, 100, 30)

	t.Run("empty text renders without error", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Render panicked on empty text: %v", r)
			}
		}()
		r.Render(img, "", 0, bounds)
	})

	t.Run("simple text renders without error", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Render panicked on simple text: %v", r)
			}
		}()
		r.Render(img, "Hello World", 0, bounds)
	})

	t.Run("multiline text renders without error", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Render panicked on multiline text: %v", r)
			}
		}()
		r.Render(img, "Line 1\nLine 2\nLine 3", 0, bounds)
	})

	t.Run("scrolling text renders without error", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Render panicked with scroll offset: %v", r)
			}
		}()
		// Long text that would require scrolling
		r.Render(img, "Line 1\nLine 2\nLine 3\nLine 4\nLine 5", 50.0, bounds)
	})
}

func TestMultiLineRenderer_RenderScrollModes(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	img := image.NewGray(image.Rect(0, 0, 128, 40))
	bounds := image.Rect(0, 0, 100, 20) // Small height to force scrolling

	longText := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"

	scrollModes := []ScrollMode{
		ScrollContinuous,
		ScrollBounce,
		ScrollPauseEnds,
		"unknown", // Test fallback
	}

	for _, mode := range scrollModes {
		t.Run("scroll mode "+string(mode), func(t *testing.T) {
			cfg := MultiLineRendererConfig{
				FontFace:      fontFace,
				FontName:      "",
				ScrollMode:    mode,
				ScrollGap:     10,
				ScrollEnabled: true,
			}

			r := NewMultiLineRenderer(cfg, 100)

			defer func() {
				if rec := recover(); rec != nil {
					t.Errorf("Render panicked with scroll mode %q: %v", mode, rec)
				}
			}()

			// Test at various scroll offsets
			for _, offset := range []float64{0, 10, 50, 100, 200} {
				r.Render(img, longText, offset, bounds)
			}
		})
	}
}

func TestMultiLineRenderer_RenderWithoutScrolling(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	img := image.NewGray(image.Rect(0, 0, 128, 40))
	bounds := image.Rect(0, 0, 100, 30)

	cfg := MultiLineRendererConfig{
		FontFace:      fontFace,
		FontName:      "",
		ScrollEnabled: false, // Scrolling disabled
	}

	r := NewMultiLineRenderer(cfg, 100)

	t.Run("short text renders without scrolling", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("Render panicked: %v", rec)
			}
		}()
		r.Render(img, "Hello", 0, bounds)
	})

	t.Run("long text truncates without scrolling", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("Render panicked: %v", rec)
			}
		}()
		r.Render(img, "Line 1\nLine 2\nLine 3\nLine 4\nLine 5", 0, bounds)
	})
}

func TestMultiLineRenderer_RenderLines(t *testing.T) {
	r := createTestRenderer(t, 100)

	img := image.NewGray(image.Rect(0, 0, 128, 40))
	bounds := image.Rect(0, 0, 100, 30)

	t.Run("empty lines renders without error", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("RenderLines panicked on empty lines: %v", rec)
			}
		}()
		r.RenderLines(img, nil, bounds)
		r.RenderLines(img, []string{}, bounds)
	})

	t.Run("renders provided lines", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("RenderLines panicked: %v", rec)
			}
		}()
		lines := []string{"Line 1", "Line 2", "Line 3"}
		r.RenderLines(img, lines, bounds)
	})
}

func TestMultiLineRenderer_BounceScrollEdgeCases(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	cfg := MultiLineRendererConfig{
		FontFace:      fontFace,
		FontName:      "",
		ScrollMode:    ScrollBounce,
		ScrollEnabled: true,
	}

	img := image.NewGray(image.Rect(0, 0, 128, 40))

	t.Run("text fits without bouncing", func(t *testing.T) {
		r := NewMultiLineRenderer(cfg, 200)
		bounds := image.Rect(0, 0, 200, 100) // Large bounds

		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("Render panicked: %v", rec)
			}
		}()
		r.Render(img, "Short", 0, bounds)
	})
}

func TestMultiLineRenderer_PauseEndsScrollEdgeCases(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	cfg := MultiLineRendererConfig{
		FontFace:      fontFace,
		FontName:      "",
		ScrollMode:    ScrollPauseEnds,
		ScrollEnabled: true,
	}

	img := image.NewGray(image.Rect(0, 0, 128, 40))

	t.Run("text fits without scrolling", func(t *testing.T) {
		r := NewMultiLineRenderer(cfg, 200)
		bounds := image.Rect(0, 0, 200, 100)

		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("Render panicked: %v", rec)
			}
		}()
		r.Render(img, "Short", 0, bounds)
	})
}
