//go:build windows

package widget

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewKeyboardLayoutWidget(t *testing.T) {
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

	widget, err := NewKeyboardLayoutWidget(cfg)
	if err != nil {
		t.Fatalf("NewKeyboardLayoutWidget() error = %v", err)
	}

	if widget == nil {
		t.Fatal("NewKeyboardLayoutWidget() returned nil")
	}

	if widget.Name() != "keyboard_layout" {
		t.Errorf("Name() = %s, want keyboard_layout", widget.Name())
	}

	if widget.displayFormat != "iso639-1" {
		t.Errorf("displayFormat = %s, want iso639-1", widget.displayFormat)
	}

	if widget.fontSize != 10 {
		t.Errorf("fontSize = %d, want 10", widget.fontSize)
	}
}

func TestNewKeyboardLayoutWidget_WithDisplayFormat(t *testing.T) {
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

			widget, err := NewKeyboardLayoutWidget(cfg)
			if err != nil {
				t.Fatalf("NewKeyboardLayoutWidget() error = %v", err)
			}

			if widget.displayFormat != format {
				t.Errorf("displayFormat = %s, want %s", widget.displayFormat, format)
			}
		})
	}
}

func TestNewKeyboardLayoutWidget_InvalidFormat(t *testing.T) {
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

	_, err := NewKeyboardLayoutWidget(cfg)
	if err == nil {
		t.Error("NewKeyboardLayoutWidget() expected error for invalid format, got nil")
	}
}

func TestKeyboardLayoutWidget_Update(t *testing.T) {
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

	widget, err := NewKeyboardLayoutWidget(cfg)
	if err != nil {
		t.Fatalf("NewKeyboardLayoutWidget() error = %v", err)
	}

	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// After update, currentLayout should be set
	if widget.currentLayout == "" {
		t.Error("Update() did not set currentLayout")
	}
}

func TestKeyboardLayoutWidget_Render(t *testing.T) {
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

	widget, err := NewKeyboardLayoutWidget(cfg)
	if err != nil {
		t.Fatalf("NewKeyboardLayoutWidget() error = %v", err)
	}

	// Update first to set layout
	_ = widget.Update()

	img, err := widget.Render()
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

func TestKeyboardLayoutWidget_FormatLayout_ISO639_1(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Format: "iso639-1",
	}

	widget, _ := NewKeyboardLayoutWidget(cfg)

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
		got := widget.formatLayout(tt.lcid)
		if got != tt.want {
			t.Errorf("formatLayout(0x%04X) = %s, want %s", tt.lcid, got, tt.want)
		}
	}
}

func TestKeyboardLayoutWidget_FormatLayout_ISO639_2(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Format: "iso639-2",
	}

	widget, _ := NewKeyboardLayoutWidget(cfg)

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
		got := widget.formatLayout(tt.lcid)
		if got != tt.want {
			t.Errorf("formatLayout(0x%04X) = %s, want %s", tt.lcid, got, tt.want)
		}
	}
}

func TestKeyboardLayoutWidget_FormatLayout_Full(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Format: "full",
	}

	widget, _ := NewKeyboardLayoutWidget(cfg)

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
		got := widget.formatLayout(tt.lcid)
		if got != tt.want {
			t.Errorf("formatLayout(0x%04X) = %s, want %s", tt.lcid, got, tt.want)
		}
	}
}

func TestKeyboardLayoutWidget_GetUpdateInterval(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "keyboard_layout",
		ID:   "keyboard_layout",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
	}

	widget, _ := NewKeyboardLayoutWidget(cfg)

	interval := widget.GetUpdateInterval()
	if interval.Seconds() != 1.0 {
		t.Errorf("GetUpdateInterval() = %v, want 1s", interval)
	}
}

func TestKeyboardLayoutWidget_GetPosition(t *testing.T) {
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

	widget, _ := NewKeyboardLayoutWidget(cfg)

	pos := widget.GetPosition()
	if pos.X != 10 || pos.Y != 20 || pos.W != 128 || pos.H != 40 {
		t.Errorf("GetPosition() = %+v, want X:10 Y:20 W:128 H:40", pos)
	}
}
