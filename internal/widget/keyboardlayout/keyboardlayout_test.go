//go:build windows

package keyboardlayout

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
		},
		Text: &config.TextConfig{
			Size: 10,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w == nil {
		t.Fatal("New() returned nil")
	}

	if w.Name() != "keyboard_layout" {
		t.Errorf("Name() = %s, want keyboard_layout", w.Name())
	}

	if w.displayFormat != "iso639-1" {
		t.Errorf("displayFormat = %s, want iso639-1", w.displayFormat)
	}

	if w.fontSize != 10 {
		t.Errorf("fontSize = %d, want 10", w.fontSize)
	}
}

func TestNew_WithDisplayFormat(t *testing.T) {
	formats := []string{"iso639-1", "iso639-2", "full"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type: "keyboard_layout",
				ID:   "keyboard_layout",
				Position: config.PositionConfig{
					W: 128,
					H: 40,
				},
				Style: &config.StyleConfig{
					Background: 0,
				},
				Format: format,
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			if w.displayFormat != format {
				t.Errorf("displayFormat = %s, want %s", w.displayFormat, format)
			}
		})
	}
}

func TestNew_InvalidFormat(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
		},
		Format: "invalid",
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("New() expected error for invalid format, got nil")
	}
}

func TestWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = w.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// After update, currentLayout should be set
	if w.currentLayout == "" {
		t.Error("Update() did not set currentLayout")
	}
}

func TestWidget_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Update first to set layout
	_ = w.Update()

	img, err := w.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Render() image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

func TestWidget_FormatLayout_ISO639_1(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Format: "iso639-1",
	}

	w, _ := New(cfg)

	tests := []struct {
		lcid uint16
		want string
	}{
		{0x0409, "EN"},     // English (US)
		{0x0419, "RU"},     // Russian
		{0x0407, "DE"},     // German
		{0x040C, "FR"},     // French
		{0x040A, "ES"},     // Spanish
		{0x9999, "0x9999"}, // Unknown
	}

	for _, tt := range tests {
		got := w.formatLayout(tt.lcid)
		if got != tt.want {
			t.Errorf("formatLayout(0x%04X) = %s, want %s", tt.lcid, got, tt.want)
		}
	}
}

func TestWidget_FormatLayout_ISO639_2(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Format: "iso639-2",
	}

	w, _ := New(cfg)

	tests := []struct {
		lcid uint16
		want string
	}{
		{0x0409, "ENG"},
		{0x0419, "RUS"},
		{0x0407, "DEU"},
		{0x040C, "FRA"},
		{0x040A, "SPA"},
	}

	for _, tt := range tests {
		got := w.formatLayout(tt.lcid)
		if got != tt.want {
			t.Errorf("formatLayout(0x%04X) = %s, want %s", tt.lcid, got, tt.want)
		}
	}
}

func TestWidget_FormatLayout_Full(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Format: "full",
	}

	w, _ := New(cfg)

	tests := []struct {
		lcid uint16
		want string
	}{
		{0x0409, "English"},
		{0x0419, "Русский"},
		{0x0407, "Deutsch"},
		{0x040C, "Français"},
		{0x040A, "Español"},
	}

	for _, tt := range tests {
		got := w.formatLayout(tt.lcid)
		if got != tt.want {
			t.Errorf("formatLayout(0x%04X) = %s, want %s", tt.lcid, got, tt.want)
		}
	}
}

func TestWidget_GetUpdateInterval(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
	}

	w, _ := New(cfg)

	interval := w.GetUpdateInterval()
	if interval.Seconds() != 1.0 {
		t.Errorf("GetUpdateInterval() = %v, want 1s", interval)
	}
}

func TestWidget_GetPosition(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			X: 10,
			Y: 20,
			W: 128,
			H: 40,
		},
	}

	w, _ := New(cfg)

	pos := w.GetPosition()
	if pos.X != 10 || pos.Y != 20 || pos.W != 128 || pos.H != 40 {
		t.Errorf("GetPosition() = %+v, want X:10 Y:20 W:128 H:40", pos)
	}
}
