package widget

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewMemoryWidget(t *testing.T) {
	tests := []struct {
		name        string
		displayMode string
	}{
		{"Text mode", "text"},
		{"Bar horizontal", "bar_horizontal"},
		{"Bar vertical", "bar_vertical"},
		{"Graph mode", "graph"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "memory",
				ID:      "test_memory",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 64,
					H: 20,
				},
				Style: &config.StyleConfig{
					Background:  0,
					Border:      false,
					BorderColor: 255,
				},
				Mode: tt.displayMode,
				Text: &config.TextConfig{
					Size:  10,
					Align: &config.AlignConfig{H: "center", V: "center"},
				},
				Colors: &config.ColorsConfig{
					Fill: config.IntPtr(255),
				},
				Graph: &config.GraphConfig{
					History: 30,
				},
			}

			widget, err := NewMemoryWidget(cfg)
			if err != nil {
				t.Fatalf("NewMemoryWidget() error = %v", err)
			}

			if widget == nil {
				t.Fatal("NewMemoryWidget() returned nil")
			}
		})
	}
}

func TestMemoryWidgetUpdate(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "memory",
		ID:      "test_memory",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 64,
			H: 20,
		},
		Style: &config.StyleConfig{
			Background:  0,
			Border:      false,
			BorderColor: 255,
		},
		Mode: "text",
		Text: &config.TextConfig{
			Size:  10,
			Align: &config.AlignConfig{H: "center", V: "center"},
		},
		Colors: &config.ColorsConfig{
			Fill: config.IntPtr(255),
		},
	}

	widget, err := NewMemoryWidget(cfg)
	if err != nil {
		t.Fatalf("NewMemoryWidget() error = %v", err)
	}

	// Update should work without error
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Usage should be between 0 and 100
	if widget.currentUsage < 0 || widget.currentUsage > 100 {
		t.Errorf("currentUsage = %.2f, want 0-100", widget.currentUsage)
	}
}

func TestMemoryWidgetRenderAllModes(t *testing.T) {
	modes := []string{"text", "bar_horizontal", "bar_vertical", "graph", "gauge"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "memory",
				ID:      "test_memory",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 64,
					H: 20,
				},
				Style: &config.StyleConfig{
					Background:  0,
					Border:      false,
					BorderColor: 255,
				},
				Mode: mode,
				Text: &config.TextConfig{
					Size:  10,
					Align: &config.AlignConfig{H: "center", V: "center"},
				},
				Colors: &config.ColorsConfig{
					Fill:   config.IntPtr(255),
					Arc:    config.IntPtr(200),
					Needle: config.IntPtr(255),
				},
				Graph: &config.GraphConfig{
					History: 30,
				},
			}

			widget, err := NewMemoryWidget(cfg)
			if err != nil {
				t.Fatalf("NewMemoryWidget() error = %v", err)
			}

			// Update to populate data
			err = widget.Update()
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			// For graph mode, add more history
			if mode == "graph" {
				for i := 0; i < 5; i++ {
					if err := widget.Update(); err != nil {
						t.Fatalf("Update() iteration %d error = %v", i, err)
					}
				}
			}

			// Render should work without error
			img, err := widget.Render()
			if err != nil {
				t.Errorf("Render() error = %v", err)
			}

			if img == nil {
				t.Fatal("Render() returned nil image")
			}
		})
	}
}

func TestMemoryWidget_GaugeDefaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "memory",
		ID:      "test_memory_gauge_defaults",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 64,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background:  0,
			Border:      false,
			BorderColor: 255,
		},
		Mode: "gauge",
		// Don't specify colors to test defaults
	}

	widget, err := NewMemoryWidget(cfg)
	if err != nil {
		t.Fatalf("NewMemoryWidget() error = %v", err)
	}

	// Verify defaults
	if widget.gaugeColor != 200 {
		t.Errorf("default gaugeColor = %d, want 200", widget.gaugeColor)
	}

	if widget.gaugeNeedleColor != 255 {
		t.Errorf("default gaugeNeedleColor = %d, want 255", widget.gaugeNeedleColor)
	}

	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}
