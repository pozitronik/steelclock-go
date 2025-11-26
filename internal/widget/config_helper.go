package widget

import (
	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// TextSettings holds extracted text configuration with defaults
type TextSettings struct {
	FontSize   int
	FontName   string
	HorizAlign string
	VertAlign  string
}

// BarSettings holds extracted bar configuration with defaults
type BarSettings struct {
	Direction string
	Border    bool
	FillColor int
}

// GaugeSettings holds extracted gauge configuration with defaults
type GaugeSettings struct {
	ArcColor    int
	NeedleColor int
	ShowTicks   bool
	TicksColor  int
}

// GraphSettings holds extracted graph configuration with defaults
type GraphSettings struct {
	HistoryLen int
	Filled     bool
	FillColor  int
}

// TriangleSettings holds extracted triangle configuration with defaults
type TriangleSettings struct {
	FillColor int
	Border    bool
}

// ConfigHelper provides centralized extraction of common widget configuration settings.
// It reduces code duplication across widget constructors by providing typed helper methods
// with consistent defaults.
type ConfigHelper struct {
	cfg config.WidgetConfig
}

// NewConfigHelper creates a new configuration helper for the given widget config
func NewConfigHelper(cfg config.WidgetConfig) *ConfigHelper {
	return &ConfigHelper{cfg: cfg}
}

// GetDisplayMode returns the display mode with a default fallback
func (h *ConfigHelper) GetDisplayMode(defaultMode string) string {
	if h.cfg.Mode == "" {
		return defaultMode
	}
	return h.cfg.Mode
}

// GetTextSettings extracts text configuration with defaults
func (h *ConfigHelper) GetTextSettings() TextSettings {
	settings := TextSettings{
		FontSize:   10,
		FontName:   "",
		HorizAlign: "center",
		VertAlign:  "center",
	}

	if h.cfg.Text != nil {
		if h.cfg.Text.Size > 0 {
			settings.FontSize = h.cfg.Text.Size
		}
		settings.FontName = h.cfg.Text.Font
		if h.cfg.Text.Align != nil {
			if h.cfg.Text.Align.H != "" {
				settings.HorizAlign = h.cfg.Text.Align.H
			}
			if h.cfg.Text.Align.V != "" {
				settings.VertAlign = h.cfg.Text.Align.V
			}
		}
	}

	return settings
}

// GetPadding extracts padding from style configuration
func (h *ConfigHelper) GetPadding() int {
	if h.cfg.Style != nil {
		return h.cfg.Style.Padding
	}
	return 0
}

// GetBarSettings extracts bar configuration with defaults
func (h *ConfigHelper) GetBarSettings() BarSettings {
	settings := BarSettings{
		Direction: "horizontal",
		Border:    false,
		FillColor: 255,
	}

	if h.cfg.Bar != nil {
		if h.cfg.Bar.Direction != "" {
			settings.Direction = h.cfg.Bar.Direction
		}
		settings.Border = h.cfg.Bar.Border
		if h.cfg.Bar.Colors != nil && h.cfg.Bar.Colors.Fill != nil {
			settings.FillColor = *h.cfg.Bar.Colors.Fill
		}
	}

	return settings
}

// GetGaugeSettings extracts gauge configuration with defaults
func (h *ConfigHelper) GetGaugeSettings() GaugeSettings {
	settings := GaugeSettings{
		ArcColor:    200,
		NeedleColor: 255,
		ShowTicks:   true,
		TicksColor:  150,
	}

	if h.cfg.Gauge != nil {
		if h.cfg.Gauge.ShowTicks != nil {
			settings.ShowTicks = *h.cfg.Gauge.ShowTicks
		}
		if h.cfg.Gauge.Colors != nil {
			if h.cfg.Gauge.Colors.Arc != nil {
				settings.ArcColor = *h.cfg.Gauge.Colors.Arc
			}
			if h.cfg.Gauge.Colors.Needle != nil {
				settings.NeedleColor = *h.cfg.Gauge.Colors.Needle
			}
			if h.cfg.Gauge.Colors.Ticks != nil {
				settings.TicksColor = *h.cfg.Gauge.Colors.Ticks
			}
		}
	}

	return settings
}

// GetGraphSettings extracts graph configuration with defaults
func (h *ConfigHelper) GetGraphSettings() GraphSettings {
	settings := GraphSettings{
		HistoryLen: 30,
		Filled:     true,
		FillColor:  255,
	}

	if h.cfg.Graph != nil {
		if h.cfg.Graph.History > 0 {
			settings.HistoryLen = h.cfg.Graph.History
		}
		if h.cfg.Graph.Filled != nil {
			settings.Filled = *h.cfg.Graph.Filled
		}
		if h.cfg.Graph.Colors != nil && h.cfg.Graph.Colors.Fill != nil {
			settings.FillColor = *h.cfg.Graph.Colors.Fill
		}
	}

	return settings
}

// GetTriangleSettings extracts triangle configuration with defaults
func (h *ConfigHelper) GetTriangleSettings() TriangleSettings {
	settings := TriangleSettings{
		FillColor: 255,
		Border:    false,
	}

	if h.cfg.Triangle != nil {
		settings.Border = h.cfg.Triangle.Border
		if h.cfg.Triangle.Colors != nil && h.cfg.Triangle.Colors.Fill != nil {
			settings.FillColor = *h.cfg.Triangle.Colors.Fill
		}
	}

	return settings
}

// GetPerCoreSettings extracts per-core configuration (CPU-specific)
func (h *ConfigHelper) GetPerCoreSettings() (enabled bool, border bool, margin int) {
	if h.cfg.PerCore != nil {
		return h.cfg.PerCore.Enabled, h.cfg.PerCore.Border, h.cfg.PerCore.Margin
	}
	return false, false, 0
}

// GetFillColorForMode returns the fill color based on display mode
// This handles the mode-specific color extraction pattern
func (h *ConfigHelper) GetFillColorForMode(mode string) int {
	fillColor := 255

	switch mode {
	case "bar":
		bar := h.GetBarSettings()
		fillColor = bar.FillColor
	case "graph":
		graph := h.GetGraphSettings()
		fillColor = graph.FillColor
	case "gauge":
		if h.cfg.Gauge != nil && h.cfg.Gauge.Colors != nil && h.cfg.Gauge.Colors.Fill != nil {
			fillColor = *h.cfg.Gauge.Colors.Fill
		}
	}

	return fillColor
}

// LoadFontForTextMode loads font if display mode is "text"
// Returns nil face (not error) if mode is not text
func (h *ConfigHelper) LoadFontForTextMode(mode string) (font.Face, error) {
	if mode != "text" {
		return nil, nil
	}

	text := h.GetTextSettings()
	return bitmap.LoadFont(text.FontName, text.FontSize)
}
