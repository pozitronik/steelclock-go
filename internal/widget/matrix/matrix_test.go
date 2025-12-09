package matrix

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "matrix",
		ID:      "test_matrix",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if widget == nil {
		t.Fatal("New() returned nil")
	}

	if widget.Name() != "test_matrix" {
		t.Errorf("Name() = %s, want test_matrix", widget.Name())
	}
}

func TestWidget_WithConfig(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "matrix",
		ID:      "test_matrix_config",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Matrix: &config.MatrixConfig{
			Charset:        "binary",
			Density:        0.6,
			MinSpeed:       1.0,
			MaxSpeed:       3.0,
			MinLength:      3,
			MaxLength:      10,
			HeadColor:      200,
			TrailFade:      0.9,
			CharChangeRate: 0.05,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if widget.charsetName != "binary" {
		t.Errorf("charsetName = %s, want binary", widget.charsetName)
	}

	if widget.density != 0.6 {
		t.Errorf("density = %f, want 0.6", widget.density)
	}

	if widget.headColor != 200 {
		t.Errorf("headColor = %d, want 200", widget.headColor)
	}
}

func TestWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "matrix",
		ID:      "test_matrix_update",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Update should not error
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Multiple updates should work
	for i := 0; i < 10; i++ {
		err = widget.Update()
		if err != nil {
			t.Errorf("Update() iteration %d error = %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestWidget_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "matrix",
		ID:      "test_matrix_render",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Render should return valid image
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("image dimensions = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

func TestWidget_Charsets(t *testing.T) {
	charsets := []string{"ascii", "katakana", "binary", "digits", "hex"}

	for _, charset := range charsets {
		t.Run(charset, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "matrix",
				ID:      "test_matrix_" + charset,
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 128,
					H: 40,
				},
				Matrix: &config.MatrixConfig{
					Charset: charset,
				},
			}

			widget, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			if widget.charsetName != charset {
				t.Errorf("charsetName = %s, want %s", widget.charsetName, charset)
			}

			// Should render without error
			_, err = widget.Render()
			if err != nil {
				t.Errorf("Render() error = %v", err)
			}
		})
	}
}

func TestWidget_SmallDisplay(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "matrix",
		ID:      "test_matrix_small",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 64,
			H: 20,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Should use smaller font for small display
	if widget.charHeight >= 8 {
		t.Log("Small display should use 3x5 font (smaller char height)")
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 20 {
		t.Errorf("image dimensions = %dx%d, want 64x20", bounds.Dx(), bounds.Dy())
	}
}

func TestWidget_ColumnsInitialized(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "matrix",
		ID:      "test_matrix_columns",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if len(widget.columns) == 0 {
		t.Error("columns should be initialized")
	}

	if widget.numColumns == 0 {
		t.Error("numColumns should be > 0")
	}

	// Each column should have characters initialized
	for i, col := range widget.columns {
		if len(col.chars) == 0 {
			t.Errorf("column %d has no characters", i)
		}
		if len(col.brightness) == 0 {
			t.Errorf("column %d has no brightness values", i)
		}
	}
}
