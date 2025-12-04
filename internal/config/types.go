package config

import "encoding/json"

// StringOrSlice is a type that can unmarshal from either a string or an array of strings.
// When unmarshaling a single string, it becomes a slice with one element.
type StringOrSlice []string

// UnmarshalJSON implements json.Unmarshaler for StringOrSlice
func (s *StringOrSlice) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = []string{str}
		return nil
	}

	// Try to unmarshal as an array of strings
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*s = arr
	return nil
}

// MarshalJSON implements json.Marshaler for StringOrSlice
func (s *StringOrSlice) MarshalJSON() ([]byte, error) {
	if len(*s) == 1 {
		return json.Marshal((*s)[0])
	}
	return json.Marshal([]string(*s))
}

// Config represents the complete SteelClock configuration (v2 schema)
type Config struct {
	SchemaVersion        int                 `json:"schema_version,omitempty"`
	ConfigName           string              `json:"config_name,omitempty"` // Display name for profile selection menu
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
	Interface      *string `json:"interface,omitempty"`       // Network
	MaxSpeedMbps   float64 `json:"max_speed_mbps,omitempty"`  // Network, Disk
	Disk           *string `json:"disk,omitempty"`            // Disk
	Unit           string  `json:"unit,omitempty"`            // Disk: "auto", "B/s", "KB/s", "MB/s", "GB/s", "KiB/s", "MiB/s", "GiB/s"
	Format         string  `json:"format,omitempty"`          // Keyboard layout
	Channel        string  `json:"channel,omitempty"`         // Audio visualizer
	ErrorThreshold int     `json:"error_threshold,omitempty"` // Audio visualizer: consecutive errors before failure (default: 30)
	Wad            string  `json:"wad,omitempty"`             // DOOM
	BundledWadURL  *string `json:"bundled_wad_url,omitempty"` // DOOM - custom WAD download URL

	// Winamp widget
	Winamp   *WinampConfig         `json:"winamp,omitempty"`    // Winamp settings (placeholder)
	Scroll   *ScrollConfig         `json:"scroll,omitempty"`    // Text scrolling settings
	AutoShow *WinampAutoShowConfig `json:"auto_show,omitempty"` // Auto-show events (Winamp)

	// Matrix widget
	Matrix *MatrixConfig `json:"matrix,omitempty"` // Matrix "digital rain" settings

	// Weather widget
	Weather *WeatherConfig `json:"weather,omitempty"` // Weather widget settings
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
	Format   string       `json:"format,omitempty"`
	Font     string       `json:"font,omitempty"` // Font name: TTF font name/path or built-in: "pixel3x5", "pixel5x7"
	Size     int          `json:"size,omitempty"`
	Align    *AlignConfig `json:"align,omitempty"`
	Unit     string       `json:"unit,omitempty"`
	ShowUnit *bool        `json:"show_unit,omitempty"` // Show unit suffix in text mode (disk widget)
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

// WinampConfig represents Winamp widget settings
// Note: format string is configured via text.format with placeholders:
// {title}, {filename}, {filepath}, {position}, {duration}, {position_ms},
// {duration_s}, {bitrate}, {samplerate}, {channels}, {status}
type WinampConfig struct {
	// Placeholder configuration when Winamp is not playing
	Placeholder *WinampPlaceholderConfig `json:"placeholder,omitempty"`
}

// WinampAutoShowConfig represents events that trigger the widget to show
type WinampAutoShowConfig struct {
	// OnTrackChange - show widget when track changes (default: true)
	OnTrackChange *bool `json:"on_track_change,omitempty"`
	// OnPlay - show widget when playback starts
	OnPlay bool `json:"on_play,omitempty"`
	// OnPause - show widget when playback is paused
	OnPause bool `json:"on_pause,omitempty"`
	// OnStop - show widget when playback stops
	OnStop bool `json:"on_stop,omitempty"`
	// OnSeek - show widget when user seeks to different position
	OnSeek bool `json:"on_seek,omitempty"`
}

// WinampPlaceholderConfig represents what to show when Winamp is not playing
type WinampPlaceholderConfig struct {
	// Mode: "text" for custom text, "icon" for Winamp icon
	Mode string `json:"mode,omitempty"`
	// Text to display when mode is "text"
	Text string `json:"text,omitempty"`
}

// ScrollConfig represents text scrolling settings
type ScrollConfig struct {
	// Enabled explicitly enables/disables scrolling
	Enabled bool `json:"enabled,omitempty"`
	// Direction: "left", "right", "up", "down"
	Direction string `json:"direction,omitempty"`
	// Speed in pixels per second
	Speed float64 `json:"speed,omitempty"`
	// Mode: "continuous" (loop), "bounce" (reverse at edges), "pause_ends" (pause at start/end)
	Mode string `json:"mode,omitempty"`
	// PauseMs - pause duration in milliseconds at ends (for bounce/pause_ends modes)
	PauseMs int `json:"pause_ms,omitempty"`
	// Gap - pixels between end and start of text in continuous mode
	Gap int `json:"gap,omitempty"`
}

// MatrixConfig represents Matrix "digital rain" widget settings
type MatrixConfig struct {
	// Charset: "ascii", "katakana", "binary", "digits", "hex"
	Charset string `json:"charset,omitempty"`
	// Density: probability of column being active (0.0-1.0, default: 0.4)
	Density float64 `json:"density,omitempty"`
	// MinSpeed: minimum fall speed in pixels per frame (default: 0.5)
	MinSpeed float64 `json:"min_speed,omitempty"`
	// MaxSpeed: maximum fall speed in pixels per frame (default: 2.0)
	MaxSpeed float64 `json:"max_speed,omitempty"`
	// MinLength: minimum trail length in characters (default: 4)
	MinLength int `json:"min_length,omitempty"`
	// MaxLength: maximum trail length in characters (default: 15)
	MaxLength int `json:"max_length,omitempty"`
	// HeadColor: brightness of leading character (0-255, default: 255)
	HeadColor int `json:"head_color,omitempty"`
	// TrailFade: how quickly trail fades (0.0-1.0, default: 0.85)
	TrailFade float64 `json:"trail_fade,omitempty"`
	// CharChangeRate: probability of character changing per frame (default: 0.02)
	CharChangeRate float64 `json:"char_change_rate,omitempty"`
	// FontSize: "small" (3x5), "large" (5x7), or "auto" (based on display height, default)
	FontSize string `json:"font_size,omitempty"`
}

// WeatherConfig represents Weather widget settings
type WeatherConfig struct {
	// Provider: "openweathermap" or "open-meteo" (default: "open-meteo")
	Provider string `json:"provider,omitempty"`
	// ApiKey: API key for OpenWeatherMap (required for openweathermap provider)
	ApiKey string `json:"api_key,omitempty"`
	// Location configuration
	Location *WeatherLocationConfig `json:"location,omitempty"`
	// Units: "metric" (Celsius, m/s) or "imperial" (Fahrenheit, mph) (default: "metric")
	Units string `json:"units,omitempty"`
	// IconSize: size of weather icons in pixels (default: 16)
	IconSize int `json:"icon_size,omitempty"`
	// Format: display format string(s) with tokens like {icon}, {temp}, {aqi}, etc.
	// Can be a single string or an array of strings (for cycling between formats).
	// Supports newlines (\n) for multi-line layouts.
	// Default: "{icon} {temp}"
	Format StringOrSlice `json:"format,omitempty"`
	// Cycle: format cycling and transition settings
	Cycle *WeatherCycleConfig `json:"cycle,omitempty"`
	// Forecast: forecast display settings (hours, days, scroll_speed)
	Forecast *WeatherForecastConfig `json:"forecast,omitempty"`

	// Deprecated fields for backward compatibility
	// ShowIcon: deprecated, use Format instead
	ShowIcon *bool `json:"show_icon,omitempty"`
	// ForecastHours: deprecated, use Forecast.Hours instead
	ForecastHours int `json:"forecast_hours,omitempty"`
	// ForecastDays: deprecated, use Forecast.Days instead
	ForecastDays int `json:"forecast_days,omitempty"`
	// ScrollSpeed: deprecated, use Forecast.ScrollSpeed instead
	ScrollSpeed float64 `json:"scroll_speed,omitempty"`
}

// WeatherLocationConfig represents weather location settings
type WeatherLocationConfig struct {
	// City: city name (e.g., "London" or "New York,US")
	City string `json:"city,omitempty"`
	// Lat: latitude for coordinate-based location
	Lat float64 `json:"lat,omitempty"`
	// Lon: longitude for coordinate-based location
	Lon float64 `json:"lon,omitempty"`
}

// WeatherForecastConfig represents forecast display settings
type WeatherForecastConfig struct {
	// Hours: number of hours for hourly forecast (default: 24, max: 48)
	Hours int `json:"hours,omitempty"`
	// Days: number of days for daily forecast (default: 3, max: 7)
	Days int `json:"days,omitempty"`
	// ScrollSpeed: pixels per second for {forecast:scroll} (default: 30)
	ScrollSpeed float64 `json:"scroll_speed,omitempty"`
}

// WeatherCycleConfig represents format cycling and transition settings
type WeatherCycleConfig struct {
	// Interval: seconds between format changes (0 to disable cycling, default: 10)
	Interval int `json:"interval,omitempty"`
	// Transition: transition effect type (default: "none")
	// Values: "none", "push_left", "push_right", "push_up", "push_down",
	//         "slide_left", "slide_right", "slide_up", "slide_down",
	//         "dissolve_fade", "dissolve_pixel", "dissolve_dither",
	//         "box_in", "box_out", "clock_wipe", "random"
	Transition string `json:"transition,omitempty"`
	// Speed: transition duration in seconds (default: 0.5)
	Speed float64 `json:"speed,omitempty"`
}
