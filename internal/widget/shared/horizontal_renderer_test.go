package shared

import (
	"image"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
)

func createTestHorizontalRenderer(t *testing.T) *HorizontalTextRenderer {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	cfg := HorizontalTextRendererConfig{
		FontFace:      fontFace,
		FontName:      "",
		HorizAlign:    config.AlignLeft,
		VertAlign:     config.AlignTop,
		ScrollMode:    ScrollContinuous,
		ScrollGap:     10,
		ScrollEnabled: true,
	}

	return NewHorizontalTextRenderer(cfg)
}

func TestNewHorizontalTextRenderer(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	t.Run("creates with configuration", func(t *testing.T) {
		cfg := HorizontalTextRendererConfig{
			FontFace:      fontFace,
			FontName:      "test",
			HorizAlign:    config.AlignCenter,
			VertAlign:     config.AlignMiddle,
			ScrollMode:    ScrollBounce,
			ScrollGap:     20,
			ScrollEnabled: true,
		}

		r := NewHorizontalTextRenderer(cfg)
		if r == nil {
			t.Fatal("NewHorizontalTextRenderer returned nil")
		}
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
		if !r.scrollEnabled {
			t.Error("Expected scrollEnabled true")
		}
	})

	t.Run("creates with scrolling disabled", func(t *testing.T) {
		cfg := HorizontalTextRendererConfig{
			FontFace:      fontFace,
			ScrollEnabled: false,
		}

		r := NewHorizontalTextRenderer(cfg)
		if r.scrollEnabled {
			t.Error("Expected scrollEnabled false")
		}
	})
}

func TestHorizontalTextRenderer_MeasureTextWidth(t *testing.T) {
	r := createTestHorizontalRenderer(t)

	t.Run("measures non-empty text", func(t *testing.T) {
		width := r.MeasureTextWidth("Hello World")
		if width <= 0 {
			t.Errorf("Expected positive width, got %d", width)
		}
	})

	t.Run("measures empty text", func(t *testing.T) {
		width := r.MeasureTextWidth("")
		if width != 0 {
			t.Errorf("Expected 0 width for empty text, got %d", width)
		}
	})
}

func TestHorizontalTextRenderer_Render(t *testing.T) {
	r := createTestHorizontalRenderer(t)

	img := image.NewGray(image.Rect(0, 0, 128, 40))
	bounds := image.Rect(0, 0, 100, 20)

	t.Run("renders empty text without error", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("Render panicked on empty text: %v", rec)
			}
		}()
		r.Render(img, "", 0, bounds)
	})

	t.Run("renders short text without error", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("Render panicked on short text: %v", rec)
			}
		}()
		r.Render(img, "Hi", 0, bounds)
	})

	t.Run("renders long text with scrolling", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("Render panicked on long text: %v", rec)
			}
		}()
		r.Render(img, "This is a very long text that should scroll horizontally", 50.0, bounds)
	})
}

func TestHorizontalTextRenderer_RenderScrollModes(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	img := image.NewGray(image.Rect(0, 0, 128, 40))
	bounds := image.Rect(0, 0, 50, 20) // Narrow width to force scrolling

	longText := "This is a long text that needs to scroll"

	scrollModes := []ScrollMode{
		ScrollContinuous,
		ScrollBounce,
		ScrollPauseEnds,
		"unknown",
	}

	for _, mode := range scrollModes {
		t.Run("scroll mode "+string(mode), func(t *testing.T) {
			cfg := HorizontalTextRendererConfig{
				FontFace:      fontFace,
				FontName:      "",
				ScrollMode:    mode,
				ScrollGap:     10,
				ScrollEnabled: true,
			}

			r := NewHorizontalTextRenderer(cfg)

			defer func() {
				if rec := recover(); rec != nil {
					t.Errorf("Render panicked with scroll mode %q: %v", mode, rec)
				}
			}()

			// Test at various scroll offsets
			for _, offset := range []float64{0, 10, 50, 100, 200, 500} {
				r.Render(img, longText, offset, bounds)
			}
		})
	}
}

func TestHorizontalTextRenderer_RenderWithoutScrolling(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	img := image.NewGray(image.Rect(0, 0, 128, 40))
	bounds := image.Rect(0, 0, 100, 20)

	cfg := HorizontalTextRendererConfig{
		FontFace:      fontFace,
		ScrollEnabled: false,
	}

	r := NewHorizontalTextRenderer(cfg)

	t.Run("short text renders without scrolling", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("Render panicked: %v", rec)
			}
		}()
		r.Render(img, "Short", 0, bounds)
	})

	t.Run("long text does not scroll when disabled", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("Render panicked: %v", rec)
			}
		}()
		r.Render(img, "This is a long text that would scroll but scrolling is disabled", 50.0, bounds)
	})
}

func TestHorizontalTextRenderer_BounceEdgeCases(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	cfg := HorizontalTextRendererConfig{
		FontFace:      fontFace,
		ScrollMode:    ScrollBounce,
		ScrollEnabled: true,
	}

	img := image.NewGray(image.Rect(0, 0, 128, 40))

	t.Run("text fits without bouncing", func(t *testing.T) {
		r := NewHorizontalTextRenderer(cfg)
		bounds := image.Rect(0, 0, 200, 20) // Wide bounds

		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("Render panicked: %v", rec)
			}
		}()
		r.Render(img, "Short", 100.0, bounds)
	})
}

func TestHorizontalTextRenderer_PauseEndsEdgeCases(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	cfg := HorizontalTextRendererConfig{
		FontFace:      fontFace,
		ScrollMode:    ScrollPauseEnds,
		ScrollEnabled: true,
	}

	img := image.NewGray(image.Rect(0, 0, 128, 40))

	t.Run("text fits without scrolling", func(t *testing.T) {
		r := NewHorizontalTextRenderer(cfg)
		bounds := image.Rect(0, 0, 200, 20)

		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("Render panicked: %v", rec)
			}
		}()
		r.Render(img, "Short", 100.0, bounds)
	})
}
