package widget

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestConfigHelper_GetDisplayMode(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		defaultMode string
		want        string
	}{
		{"empty mode uses default", "", "text", "text"},
		{"specified mode overrides default", "bar", "text", "bar"},
		{"gauge mode", "gauge", "text", "gauge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{Mode: tt.mode}
			h := NewConfigHelper(cfg)
			got := h.GetDisplayMode(tt.defaultMode)
			if got != tt.want {
				t.Errorf("GetDisplayMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigHelper_GetTextSettings(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg := config.WidgetConfig{}
		h := NewConfigHelper(cfg)
		settings := h.GetTextSettings()

		if settings.FontSize != 10 {
			t.Errorf("FontSize = %d, want 10", settings.FontSize)
		}
		if settings.FontName != "" {
			t.Errorf("FontName = %s, want empty", settings.FontName)
		}
		if settings.HorizAlign != "center" {
			t.Errorf("HorizAlign = %s, want center", settings.HorizAlign)
		}
		if settings.VertAlign != "center" {
			t.Errorf("VertAlign = %s, want center", settings.VertAlign)
		}
	})

	t.Run("custom values", func(t *testing.T) {
		cfg := config.WidgetConfig{
			Text: &config.TextConfig{
				Size: 14,
				Font: "custom.ttf",
				Align: &config.AlignConfig{
					H: "left",
					V: "top",
				},
			},
		}
		h := NewConfigHelper(cfg)
		settings := h.GetTextSettings()

		if settings.FontSize != 14 {
			t.Errorf("FontSize = %d, want 14", settings.FontSize)
		}
		if settings.FontName != "custom.ttf" {
			t.Errorf("FontName = %s, want custom.ttf", settings.FontName)
		}
		if settings.HorizAlign != "left" {
			t.Errorf("HorizAlign = %s, want left", settings.HorizAlign)
		}
		if settings.VertAlign != "top" {
			t.Errorf("VertAlign = %s, want top", settings.VertAlign)
		}
	})

	t.Run("partial align config", func(t *testing.T) {
		cfg := config.WidgetConfig{
			Text: &config.TextConfig{
				Size: 12,
				Align: &config.AlignConfig{
					H: "right",
					// V not specified, should use default
				},
			},
		}
		h := NewConfigHelper(cfg)
		settings := h.GetTextSettings()

		if settings.HorizAlign != "right" {
			t.Errorf("HorizAlign = %s, want right", settings.HorizAlign)
		}
		if settings.VertAlign != "center" {
			t.Errorf("VertAlign = %s, want center (default)", settings.VertAlign)
		}
	})
}

func TestConfigHelper_GetPadding(t *testing.T) {
	t.Run("no style", func(t *testing.T) {
		cfg := config.WidgetConfig{}
		h := NewConfigHelper(cfg)
		if h.GetPadding() != 0 {
			t.Errorf("GetPadding() = %d, want 0", h.GetPadding())
		}
	})

	t.Run("with padding", func(t *testing.T) {
		cfg := config.WidgetConfig{
			Style: &config.StyleConfig{Padding: 5},
		}
		h := NewConfigHelper(cfg)
		if h.GetPadding() != 5 {
			t.Errorf("GetPadding() = %d, want 5", h.GetPadding())
		}
	})
}

func TestConfigHelper_GetBarSettings(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg := config.WidgetConfig{}
		h := NewConfigHelper(cfg)
		settings := h.GetBarSettings()

		if settings.Direction != "horizontal" {
			t.Errorf("Direction = %s, want horizontal", settings.Direction)
		}
		if settings.Border != false {
			t.Errorf("Border = %v, want false", settings.Border)
		}
		if settings.FillColor != 255 {
			t.Errorf("FillColor = %d, want 255", settings.FillColor)
		}
	})

	t.Run("custom values", func(t *testing.T) {
		fillColor := 128
		cfg := config.WidgetConfig{
			Bar: &config.BarConfig{
				Direction: "vertical",
				Border:    true,
				Colors: &config.ModeColorsConfig{
					Fill: &fillColor,
				},
			},
		}
		h := NewConfigHelper(cfg)
		settings := h.GetBarSettings()

		if settings.Direction != "vertical" {
			t.Errorf("Direction = %s, want vertical", settings.Direction)
		}
		if settings.Border != true {
			t.Errorf("Border = %v, want true", settings.Border)
		}
		if settings.FillColor != 128 {
			t.Errorf("FillColor = %d, want 128", settings.FillColor)
		}
	})
}

func TestConfigHelper_GetGaugeSettings(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg := config.WidgetConfig{}
		h := NewConfigHelper(cfg)
		settings := h.GetGaugeSettings()

		if settings.ArcColor != 200 {
			t.Errorf("ArcColor = %d, want 200", settings.ArcColor)
		}
		if settings.NeedleColor != 255 {
			t.Errorf("NeedleColor = %d, want 255", settings.NeedleColor)
		}
		if settings.ShowTicks != true {
			t.Errorf("ShowTicks = %v, want true", settings.ShowTicks)
		}
		if settings.TicksColor != 150 {
			t.Errorf("TicksColor = %d, want 150", settings.TicksColor)
		}
	})

	t.Run("custom values", func(t *testing.T) {
		arcColor := 100
		needleColor := 200
		ticksColor := 50
		showTicks := false
		cfg := config.WidgetConfig{
			Gauge: &config.GaugeConfig{
				ShowTicks: &showTicks,
				Colors: &config.ModeColorsConfig{
					Arc:    &arcColor,
					Needle: &needleColor,
					Ticks:  &ticksColor,
				},
			},
		}
		h := NewConfigHelper(cfg)
		settings := h.GetGaugeSettings()

		if settings.ArcColor != 100 {
			t.Errorf("ArcColor = %d, want 100", settings.ArcColor)
		}
		if settings.NeedleColor != 200 {
			t.Errorf("NeedleColor = %d, want 200", settings.NeedleColor)
		}
		if settings.ShowTicks != false {
			t.Errorf("ShowTicks = %v, want false", settings.ShowTicks)
		}
		if settings.TicksColor != 50 {
			t.Errorf("TicksColor = %d, want 50", settings.TicksColor)
		}
	})
}

func TestConfigHelper_GetGraphSettings(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg := config.WidgetConfig{}
		h := NewConfigHelper(cfg)
		settings := h.GetGraphSettings()

		if settings.HistoryLen != 30 {
			t.Errorf("HistoryLen = %d, want 30", settings.HistoryLen)
		}
		if settings.Filled != true {
			t.Errorf("Filled = %v, want true", settings.Filled)
		}
		if settings.FillColor != 255 {
			t.Errorf("FillColor = %d, want 255", settings.FillColor)
		}
	})

	t.Run("custom values", func(t *testing.T) {
		fillColor := 180
		filled := false
		cfg := config.WidgetConfig{
			Graph: &config.GraphConfig{
				History: 60,
				Filled:  &filled,
				Colors: &config.ModeColorsConfig{
					Fill: &fillColor,
				},
			},
		}
		h := NewConfigHelper(cfg)
		settings := h.GetGraphSettings()

		if settings.HistoryLen != 60 {
			t.Errorf("HistoryLen = %d, want 60", settings.HistoryLen)
		}
		if settings.Filled != false {
			t.Errorf("Filled = %v, want false", settings.Filled)
		}
		if settings.FillColor != 180 {
			t.Errorf("FillColor = %d, want 180", settings.FillColor)
		}
	})
}

func TestConfigHelper_GetPerCoreSettings(t *testing.T) {
	t.Run("defaults (nil)", func(t *testing.T) {
		cfg := config.WidgetConfig{}
		h := NewConfigHelper(cfg)
		enabled, border, margin := h.GetPerCoreSettings()

		if enabled != false {
			t.Errorf("enabled = %v, want false", enabled)
		}
		if border != false {
			t.Errorf("border = %v, want false", border)
		}
		if margin != 0 {
			t.Errorf("margin = %d, want 0", margin)
		}
	})

	t.Run("custom values", func(t *testing.T) {
		cfg := config.WidgetConfig{
			PerCore: &config.PerCoreConfig{
				Enabled: true,
				Border:  true,
				Margin:  2,
			},
		}
		h := NewConfigHelper(cfg)
		enabled, border, margin := h.GetPerCoreSettings()

		if enabled != true {
			t.Errorf("enabled = %v, want true", enabled)
		}
		if border != true {
			t.Errorf("border = %v, want true", border)
		}
		if margin != 2 {
			t.Errorf("margin = %d, want 2", margin)
		}
	})
}

func TestConfigHelper_GetFillColorForMode(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.WidgetConfig
		mode string
		want int
	}{
		{
			name: "default fill color",
			cfg:  config.WidgetConfig{},
			mode: "text",
			want: 255,
		},
		{
			name: "bar mode with fill color",
			cfg: config.WidgetConfig{
				Bar: &config.BarConfig{
					Colors: &config.ModeColorsConfig{Fill: config.IntPtr(100)},
				},
			},
			mode: "bar",
			want: 100,
		},
		{
			name: "graph mode with fill color",
			cfg: config.WidgetConfig{
				Graph: &config.GraphConfig{
					Colors: &config.ModeColorsConfig{Fill: config.IntPtr(150)},
				},
			},
			mode: "graph",
			want: 150,
		},
		{
			name: "gauge mode with fill color",
			cfg: config.WidgetConfig{
				Gauge: &config.GaugeConfig{
					Colors: &config.ModeColorsConfig{Fill: config.IntPtr(200)},
				},
			},
			mode: "gauge",
			want: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewConfigHelper(tt.cfg)
			got := h.GetFillColorForMode(tt.mode)
			if got != tt.want {
				t.Errorf("GetFillColorForMode(%s) = %d, want %d", tt.mode, got, tt.want)
			}
		})
	}
}

func TestConfigHelper_LoadFontForTextMode(t *testing.T) {
	t.Run("non-text mode returns nil", func(t *testing.T) {
		cfg := config.WidgetConfig{}
		h := NewConfigHelper(cfg)

		face, err := h.LoadFontForTextMode("bar")
		if err != nil {
			t.Errorf("LoadFontForTextMode(bar) returned error: %v", err)
		}
		if face != nil {
			t.Error("LoadFontForTextMode(bar) should return nil face")
		}
	})

	t.Run("text mode loads default font", func(t *testing.T) {
		cfg := config.WidgetConfig{
			Text: &config.TextConfig{
				Size: 10,
			},
		}
		h := NewConfigHelper(cfg)

		face, err := h.LoadFontForTextMode("text")
		if err != nil {
			t.Errorf("LoadFontForTextMode(text) returned error: %v", err)
		}
		if face == nil {
			t.Error("LoadFontForTextMode(text) should return a font face")
		}
	})
}
