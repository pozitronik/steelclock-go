package shared

import (
	"fmt"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"golang.org/x/image/font"
)

// TextSettings holds extracted text configuration with defaults
type TextSettings struct {
	FontSize   int
	FontName   string
	HorizAlign config.HAlign
	VertAlign  config.VAlign
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
	FillColor  int // -1 = disabled, 0-255 = fill color
	LineColor  int // 0-255 = line color
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
		HorizAlign: config.AlignCenter,
		VertAlign:  config.AlignMiddle,
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
		Direction: config.DirectionHorizontal,
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
		FillColor:  255, // Default: filled with white. Use -1 to disable fill.
		LineColor:  255, // Default: white line
	}

	if h.cfg.Graph != nil {
		if h.cfg.Graph.History > 0 {
			settings.HistoryLen = h.cfg.Graph.History
		}
		if h.cfg.Graph.Colors != nil {
			if h.cfg.Graph.Colors.Fill != nil {
				settings.FillColor = *h.cfg.Graph.Colors.Fill
			}
			if h.cfg.Graph.Colors.Line != nil {
				settings.LineColor = *h.cfg.Graph.Colors.Line
			}
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

// MetricRendererResult holds all outputs from BuildMetricRenderer needed by widget constructors
type MetricRendererResult struct {
	Renderer    *render.MetricRenderer
	Strategy    render.MetricDisplayStrategy
	DisplayMode render.DisplayMode
	FontFace    font.Face
	FontName    string
	Padding     int
	FillColor   int // from graph settings (-1 = no fill, 0-255)
	HistoryLen  int // from graph settings
}

// BuildMetricRenderer extracts common widget settings and builds a MetricRenderer.
// This eliminates duplicated initialization code across single-value metric widgets
// (CPU, Memory, and any future similar widgets).
func (h *ConfigHelper) BuildMetricRenderer() (*MetricRendererResult, error) {
	displayMode := render.DisplayMode(h.GetDisplayMode(config.ModeText))
	textSettings := h.GetTextSettings()
	padding := h.GetPadding()
	barSettings := h.GetBarSettings()
	graphSettings := h.GetGraphSettings()
	gaugeSettings := h.GetGaugeSettings()

	// Load font for text mode
	fontFace, err := bitmap.LoadFontForTextMode(string(displayMode), textSettings.FontName, textSettings.FontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	// Determine bar color
	barColor := uint8(255)
	if graphSettings.FillColor >= 0 && graphSettings.FillColor <= 255 {
		barColor = uint8(graphSettings.FillColor)
	}

	// Create metric renderer
	renderer := render.NewMetricRenderer(
		render.BarConfig{
			Direction: barSettings.Direction,
			Border:    barSettings.Border,
			Color:     barColor,
		},
		render.GraphConfig{
			FillColor:  graphSettings.FillColor,
			LineColor:  graphSettings.LineColor,
			HistoryLen: graphSettings.HistoryLen,
		},
		render.GaugeConfig{
			ArcColor:    uint8(gaugeSettings.ArcColor),
			NeedleColor: uint8(gaugeSettings.NeedleColor),
			ShowTicks:   gaugeSettings.ShowTicks,
			TicksColor:  uint8(gaugeSettings.TicksColor),
		},
		render.TextConfig{
			FontFace:   fontFace,
			FontName:   textSettings.FontName,
			HorizAlign: textSettings.HorizAlign,
			VertAlign:  textSettings.VertAlign,
			Padding:    padding,
		},
	)

	return &MetricRendererResult{
		Renderer:    renderer,
		Strategy:    render.GetMetricStrategy(displayMode),
		DisplayMode: displayMode,
		FontFace:    fontFace,
		FontName:    textSettings.FontName,
		Padding:     padding,
		FillColor:   graphSettings.FillColor,
		HistoryLen:  graphSettings.HistoryLen,
	}, nil
}

// GetFillColorForMode returns the fill color based on display mode
// This handles the mode-specific color extraction pattern
func (h *ConfigHelper) GetFillColorForMode(mode string) int {
	fillColor := 255

	switch mode {
	case config.ModeBar:
		bar := h.GetBarSettings()
		fillColor = bar.FillColor
	case config.ModeGraph:
		graph := h.GetGraphSettings()
		fillColor = graph.FillColor
	case config.ModeGauge:
		if h.cfg.Gauge != nil && h.cfg.Gauge.Colors != nil && h.cfg.Gauge.Colors.Fill != nil {
			fillColor = *h.cfg.Gauge.Colors.Fill
		}
	}

	return fillColor
}
