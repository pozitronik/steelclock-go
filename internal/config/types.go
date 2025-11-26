package config

// Config represents the complete SteelClock configuration (v2 schema)
type Config struct {
	SchemaVersion        int                 `json:"schema_version,omitempty"`
	GameName             string              `json:"game_name"`
	GameDisplayName      string              `json:"game_display_name"`
	RefreshRateMs        int                 `json:"refresh_rate_ms"`
	UnregisterOnExit     bool                `json:"unregister_on_exit,omitempty"`
	DeinitializeTimerMs  int                 `json:"deinitialize_timer_ms,omitempty"`
	EventBatchingEnabled bool                `json:"event_batching_enabled,omitempty"`
	EventBatchSize       int                 `json:"event_batch_size,omitempty"`
	SupportedResolutions []ResolutionConfig  `json:"supported_resolutions,omitempty"`
	BundledFontURL       *string             `json:"bundled_font_url,omitempty"`
	Backend              string              `json:"backend,omitempty"`
	DirectDriver         *DirectDriverConfig `json:"direct_driver,omitempty"`
	Display              DisplayConfig       `json:"display"`
	Defaults             *DefaultsConfig     `json:"defaults,omitempty"`
	Layout               *LayoutConfig       `json:"layout,omitempty"`
	Widgets              []WidgetConfig      `json:"widgets"`
}

// DirectDriverConfig represents settings for direct USB HID driver
type DirectDriverConfig struct {
	VID       string `json:"vid,omitempty"`
	PID       string `json:"pid,omitempty"`
	Interface string `json:"interface,omitempty"`
}

// ResolutionConfig represents an additional display resolution
type ResolutionConfig struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DisplayConfig represents display settings
type DisplayConfig struct {
	Width      int `json:"width"`
	Height     int `json:"height"`
	Background int `json:"background"`
}

// DefaultsConfig represents global defaults inherited by widgets
type DefaultsConfig struct {
	Colors         map[string]int `json:"colors,omitempty"`
	Text           *TextConfig    `json:"text,omitempty"`
	UpdateInterval float64        `json:"update_interval,omitempty"`
}

// LayoutConfig represents virtual canvas layout settings
type LayoutConfig struct {
	Type          string `json:"type"`
	VirtualWidth  int    `json:"virtual_width,omitempty"`
	VirtualHeight int    `json:"virtual_height,omitempty"`
}

// WidgetConfig represents a widget configuration (v2 schema)
type WidgetConfig struct {
	Type     string         `json:"type"`
	ID       string         `json:"-"` // Auto-generated, not from JSON
	Enabled  *bool          `json:"enabled,omitempty"`
	Position PositionConfig `json:"position"`
	Style    *StyleConfig   `json:"style,omitempty"`

	// Mode selection (replaces display_mode)
	Mode string `json:"mode,omitempty"`

	// Mode-specific configurations
	Bar          *BarConfig          `json:"bar,omitempty"`
	Graph        *GraphConfig        `json:"graph,omitempty"`
	Gauge        *GaugeConfig        `json:"gauge,omitempty"`
	Triangle     *TriangleConfig     `json:"triangle,omitempty"`
	Analog       *AnalogConfig       `json:"analog,omitempty"`
	Spectrum     *SpectrumConfig     `json:"spectrum,omitempty"`
	Oscilloscope *OscilloscopeConfig `json:"oscilloscope,omitempty"`

	// Common widget configurations
	Text           *TextConfig     `json:"text,omitempty"`
	Colors         *ColorsConfig   `json:"colors,omitempty"`
	AutoHide       *AutoHideConfig `json:"auto_hide,omitempty"`
	UpdateInterval float64         `json:"update_interval,omitempty"`
	PollInterval   float64         `json:"poll_interval,omitempty"` // Internal polling rate for volume/volume_meter (seconds)

	// Widget-specific configurations
	PerCore    *PerCoreConfig    `json:"per_core,omitempty"`   // CPU widget
	Stereo     *StereoConfig     `json:"stereo,omitempty"`     // Volume meter
	Metering   *MeteringConfig   `json:"metering,omitempty"`   // Volume meter
	Peak       *PeakConfig       `json:"peak,omitempty"`       // Volume meter
	Clipping   *ClippingConfig   `json:"clipping,omitempty"`   // Volume meter
	Indicators *IndicatorsConfig `json:"indicators,omitempty"` // Keyboard
	Layout     *KeyboardLayout   `json:"layout,omitempty"`     // Keyboard

	// Simple widget-specific properties
	Interface     *string `json:"interface,omitempty"`       // Network
	MaxSpeedMbps  float64 `json:"max_speed_mbps,omitempty"`  // Network, Disk
	Disk          *string `json:"disk,omitempty"`            // Disk
	Format        string  `json:"format,omitempty"`          // Keyboard layout
	Channel       string  `json:"channel,omitempty"`         // Audio visualizer
	Wad           string  `json:"wad,omitempty"`             // DOOM
	BundledWadURL *string `json:"bundled_wad_url,omitempty"` // DOOM - custom WAD download URL
}

// IsEnabled returns true if the widget is enabled (defaults to true if not specified)
func (w *WidgetConfig) IsEnabled() bool {
	if w.Enabled == nil {
		return true
	}
	return *w.Enabled
}

// PositionConfig represents widget position and size
type PositionConfig struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
	Z int `json:"z"`
}

// StyleConfig represents widget styling
type StyleConfig struct {
	Background int `json:"background"`
	Border     int `json:"border"` // -1=disabled, 0-255=border color
	Padding    int `json:"padding,omitempty"`
}

// TextConfig represents text rendering properties
type TextConfig struct {
	Format string       `json:"format,omitempty"`
	Font   string       `json:"font,omitempty"`
	Size   int          `json:"size,omitempty"`
	Align  *AlignConfig `json:"align,omitempty"`
	Unit   string       `json:"unit,omitempty"`
}

// AlignConfig represents text alignment
type AlignConfig struct {
	H string `json:"h,omitempty"` // "left", "center", "right"
	V string `json:"v,omitempty"` // "top", "center", "bottom"
}

// ColorsConfig represents widget colors (keys vary by widget type)
type ColorsConfig struct {
	// Common
	Fill *int `json:"fill,omitempty"`

	// Gauge
	Arc    *int `json:"arc,omitempty"`
	Needle *int `json:"needle,omitempty"`
	Ticks  *int `json:"ticks,omitempty"`

	// Clock analog
	Face   *int `json:"face,omitempty"`
	Hour   *int `json:"hour,omitempty"`
	Minute *int `json:"minute,omitempty"`
	Second *int `json:"second,omitempty"`

	// Network
	Rx       *int `json:"rx,omitempty"`
	Tx       *int `json:"tx,omitempty"`
	RxNeedle *int `json:"rx_needle,omitempty"`
	TxNeedle *int `json:"tx_needle,omitempty"`

	// Disk
	Read  *int `json:"read,omitempty"`
	Write *int `json:"write,omitempty"`

	// Keyboard
	On  *int `json:"on,omitempty"`
	Off *int `json:"off,omitempty"`

	// Audio visualizer
	Left  *int `json:"left,omitempty"`
	Right *int `json:"right,omitempty"`
}

// AutoHideConfig represents auto-hide behavior
type AutoHideConfig struct {
	Enabled     bool    `json:"enabled,omitempty"`
	Timeout     float64 `json:"timeout,omitempty"`
	OnSilence   bool    `json:"on_silence,omitempty"`
	SilenceTime float64 `json:"silence_time,omitempty"`
}

// ModeColorsConfig represents colors for mode-specific rendering
type ModeColorsConfig struct {
	// Common fill color (bar, graph, gauge, triangle)
	Fill *int `json:"fill,omitempty"`
	Line *int `json:"line,omitempty"` // Optional separate line color for graph

	// Gauge specific
	Arc    *int `json:"arc,omitempty"`
	Needle *int `json:"needle,omitempty"`
	Ticks  *int `json:"ticks,omitempty"`

	// Analog clock specific
	Face   *int `json:"face,omitempty"`
	Hour   *int `json:"hour,omitempty"`
	Minute *int `json:"minute,omitempty"`
	Second *int `json:"second,omitempty"`

	// Dual-value widgets (Network)
	Rx       *int `json:"rx,omitempty"`
	Tx       *int `json:"tx,omitempty"`
	RxNeedle *int `json:"rx_needle,omitempty"`
	TxNeedle *int `json:"tx_needle,omitempty"`

	// Dual-value widgets (Disk)
	Read  *int `json:"read,omitempty"`
	Write *int `json:"write,omitempty"`

	// Audio visualizer stereo channels (separated mode)
	Left  *int `json:"left,omitempty"`
	Right *int `json:"right,omitempty"`

	// Volume meter clipping indicator
	Clipping *int `json:"clipping,omitempty"`

	// Volume meter peak hold indicator
	Peak *int `json:"peak,omitempty"`
}

// BarConfig represents bar mode settings
type BarConfig struct {
	Direction string            `json:"direction,omitempty"` // "horizontal", "vertical"
	Border    bool              `json:"border,omitempty"`
	Colors    *ModeColorsConfig `json:"colors,omitempty"`
}

// GraphConfig represents graph mode settings
type GraphConfig struct {
	History int               `json:"history,omitempty"`
	Filled  *bool             `json:"filled,omitempty"`
	Colors  *ModeColorsConfig `json:"colors,omitempty"`
}

// GaugeConfig represents gauge mode settings
type GaugeConfig struct {
	ShowTicks *bool             `json:"show_ticks,omitempty"`
	Colors    *ModeColorsConfig `json:"colors,omitempty"`
}

// TriangleConfig represents triangle mode settings (volume widget)
type TriangleConfig struct {
	Border bool              `json:"border,omitempty"`
	Colors *ModeColorsConfig `json:"colors,omitempty"`
}

// AnalogConfig represents analog clock mode settings
type AnalogConfig struct {
	ShowSeconds bool              `json:"show_seconds,omitempty"`
	ShowTicks   bool              `json:"show_ticks,omitempty"`
	Colors      *ModeColorsConfig `json:"colors,omitempty"`
}

// SpectrumConfig represents spectrum analyzer settings
type SpectrumConfig struct {
	Bars                  int                   `json:"bars,omitempty"`
	Scale                 string                `json:"scale,omitempty"` // "logarithmic", "linear"
	Style                 string                `json:"style,omitempty"` // "bars", "line"
	Smoothing             float64               `json:"smoothing,omitempty"`
	FrequencyCompensation bool                  `json:"frequency_compensation,omitempty"`
	DynamicScaling        *DynamicScalingConfig `json:"dynamic_scaling,omitempty"`
	Peak                  *PeakConfig           `json:"peak,omitempty"`
	Colors                *ModeColorsConfig     `json:"colors,omitempty"`
}

// DynamicScalingConfig represents dynamic scaling settings
type DynamicScalingConfig struct {
	Strength float64 `json:"strength,omitempty"`
	Window   float64 `json:"window,omitempty"`
}

// OscilloscopeConfig represents oscilloscope settings
type OscilloscopeConfig struct {
	Style   string            `json:"style,omitempty"` // "line", "filled"
	Samples int               `json:"samples,omitempty"`
	Colors  *ModeColorsConfig `json:"colors,omitempty"`
}

// PerCoreConfig represents per-core CPU settings
type PerCoreConfig struct {
	Enabled bool `json:"enabled,omitempty"`
	Margin  int  `json:"margin,omitempty"`
	Border  bool `json:"border,omitempty"`
}

// StereoConfig represents stereo settings for volume meter
type StereoConfig struct {
	Enabled bool `json:"enabled,omitempty"`
	Divider *int `json:"divider,omitempty"` // Divider color between left/right channels (0-255, -1=disabled)
}

// MeteringConfig represents VU meter metering settings
type MeteringConfig struct {
	DBScale          bool    `json:"db_scale,omitempty"`
	DecayRate        float64 `json:"decay_rate,omitempty"`
	SilenceThreshold float64 `json:"silence_threshold,omitempty"`
}

// PeakConfig represents peak hold settings
type PeakConfig struct {
	Enabled  bool    `json:"enabled,omitempty"`
	HoldTime float64 `json:"hold_time,omitempty"`
}

// ClippingConfig represents clipping detection settings
type ClippingConfig struct {
	Enabled   bool    `json:"enabled,omitempty"`
	Threshold float64 `json:"threshold,omitempty"`
}

// IndicatorsConfig represents keyboard indicator settings
type IndicatorsConfig struct {
	Caps   *IndicatorConfig `json:"caps,omitempty"`
	Num    *IndicatorConfig `json:"num,omitempty"`
	Scroll *IndicatorConfig `json:"scroll,omitempty"`
}

// IndicatorConfig represents a single keyboard indicator
type IndicatorConfig struct {
	On  *string `json:"on"`  // nil = use embedded glyph, string = use text
	Off *string `json:"off"` // nil = use embedded glyph, string = use text
}

// KeyboardLayout represents keyboard widget layout settings
type KeyboardLayout struct {
	Spacing   int    `json:"spacing,omitempty"`
	Separator string `json:"separator,omitempty"`
}
