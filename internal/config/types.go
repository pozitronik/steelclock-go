package config

import (
	"encoding/json"
	"fmt"
)

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

// IntOrRange represents a value that can be either a single integer or a range {min, max}.
// When unmarshaling a single integer, Min and Max are set to the same value.
// When unmarshaling an object with min/max, those values are used.
type IntOrRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// UnmarshalJSON implements json.Unmarshaler for IntOrRange
func (r *IntOrRange) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as an integer first
	var val int
	if err := json.Unmarshal(data, &val); err == nil {
		r.Min = val
		r.Max = val
		return nil
	}

	// Try to unmarshal as an object with min/max
	type rangeObj struct {
		Min int `json:"min"`
		Max int `json:"max"`
	}
	var obj rangeObj
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	r.Min = obj.Min
	r.Max = obj.Max
	return nil
}

// MarshalJSON implements json.Marshaler for IntOrRange
func (r *IntOrRange) MarshalJSON() ([]byte, error) {
	if r.Min == r.Max {
		return json.Marshal(r.Min)
	}
	return json.Marshal(map[string]int{"min": r.Min, "max": r.Max})
}

// IsRange returns true if this represents a range (min != max)
func (r *IntOrRange) IsRange() bool {
	return r.Min != r.Max
}

// Value returns the single value if not a range, or 0 if it is a range
func (r *IntOrRange) Value() int {
	if r.Min == r.Max {
		return r.Min
	}
	return 0
}

// IntOrString represents a value that can be either an integer or a string.
// Used for display selection: integer for index, string for name matching.
type IntOrString struct {
	IntValue    int
	StringValue string
	IsString    bool
}

// UnmarshalJSON implements json.Unmarshaler for IntOrString
func (v *IntOrString) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as an integer first
	var intVal int
	if err := json.Unmarshal(data, &intVal); err == nil {
		v.IntValue = intVal
		v.StringValue = ""
		v.IsString = false
		return nil
	}

	// Try to unmarshal as a string
	var strVal string
	if err := json.Unmarshal(data, &strVal); err == nil {
		v.StringValue = strVal
		v.IntValue = 0
		v.IsString = true
		return nil
	}

	return fmt.Errorf("display must be an integer or string, got: %s", string(data))
}

// MarshalJSON implements json.Marshaler for IntOrString
func (v *IntOrString) MarshalJSON() ([]byte, error) {
	if v.IsString {
		return json.Marshal(v.StringValue)
	}
	return json.Marshal(v.IntValue)
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
	FrameDedupEnabled    *bool               `json:"frame_dedup_enabled,omitempty"` // Skip sending unchanged frames (default: true)
	SupportedResolutions []ResolutionConfig  `json:"supported_resolutions,omitempty"`
	BundledFontURL       *string             `json:"bundled_font_url,omitempty"`
	Backend              string              `json:"backend,omitempty"`
	DirectDriver         *DirectDriverConfig `json:"direct_driver,omitempty"`
	WebClient            *WebClientConfig    `json:"webclient,omitempty"`
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

// WebClientConfig represents settings for web client backend
type WebClientConfig struct {
	// TargetFPS limits the frame rate sent to web clients (default: 30)
	TargetFPS int `json:"target_fps,omitempty"`
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
	Binary       *BinaryClockConfig  `json:"binary,omitempty"`  // Clock binary mode
	Segment      *SegmentClockConfig `json:"segment,omitempty"` // Clock segment mode
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

	// DOOM widget
	Doom *DoomConfig `json:"doom,omitempty"` // DOOM display settings

	// Battery widget
	Battery     *BatteryConfig     `json:"battery,omitempty"`      // Battery display settings
	PowerStatus *PowerStatusConfig `json:"power_status,omitempty"` // Power status indicator settings

	// Weather widget
	Weather *WeatherConfig `json:"weather,omitempty"` // Weather widget settings

	// Game of Life widget
	GameOfLife *GameOfLifeConfig `json:"game_of_life,omitempty"` // Game of Life settings

	// Hyperspace widget
	Hyperspace *HyperspaceConfig `json:"hyperspace,omitempty"` // Hyperspace effect settings

	// Star Wars intro crawl widget
	StarWarsIntro *StarWarsIntroConfig `json:"starwars_intro,omitempty"` // Star Wars intro crawl settings

	// Clipboard widget
	Clipboard *ClipboardWidgetConfig `json:"clipboard,omitempty"` // Clipboard widget settings

	// Screen mirror widget
	ScreenMirror *ScreenMirrorConfig `json:"screen_mirror,omitempty"` // Screen mirror widget settings

	// Hacker code widget
	HackerCode *HackerCodeConfig `json:"hacker_code,omitempty"` // Hacker code widget settings

	// Telegram widgets (shared auth config)
	Auth *TelegramAuthConfig `json:"auth,omitempty"` // Telegram API authentication (shared by telegram and telegram_counter)

	// Telegram widget filters (for both telegram and telegram_counter)
	Filters *TelegramFiltersConfig `json:"filters,omitempty"` // Message filters (private_chats, groups, channels)

	// Telegram notification widget appearance
	Appearance *TelegramAppearanceConfig `json:"appearance,omitempty"` // Notification appearance settings

	// Telegram counter widget specific
	Badge *TelegramBadgeConfig `json:"badge,omitempty"` // Badge mode settings (for telegram_counter)

	// Claude Code status widget
	ClaudeCode *ClaudeCodeConfig `json:"claude_code,omitempty"` // Claude Code status widget settings

	// Bluetooth widget
	Bluetooth *BluetoothConfig `json:"bluetooth,omitempty"` // Bluetooth device status settings

	// Beefweb widget (Foobar2000/DeaDBeeF)
	Beefweb         *BeefwebConfig         `json:"beefweb,omitempty"`           // Beefweb settings
	BeefwebAutoShow *BeefwebAutoShowConfig `json:"beefweb_auto_show,omitempty"` // Beefweb auto-show events

	// Spotify widget
	SpotifyAuth     *SpotifyAuthConfig     `json:"spotify_auth,omitempty"`      // Spotify authentication settings
	Spotify         *SpotifyConfig         `json:"spotify,omitempty"`           // Spotify display settings
	SpotifyAutoShow *SpotifyAutoShowConfig `json:"spotify_auto_show,omitempty"` // Spotify auto-show events
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
	Use12h   bool         `json:"use_12h,omitempty"`   // Use 12-hour format instead of 24-hour (clock widget)
	ShowAmPm bool         `json:"show_ampm,omitempty"` // Show AM/PM text when use_12h is true (clock widget)
}

// AlignConfig represents text alignment
type AlignConfig struct {
	H HAlign `json:"h,omitempty"` // "left", "center", "right"
	V VAlign `json:"v,omitempty"` // "top", "center", "bottom"
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

// BinaryClockConfig represents binary clock mode settings
type BinaryClockConfig struct {
	// Format: time format like "%H:%M" or "%H:%M:%S" (default: "%H:%M:%S")
	Format string `json:"format,omitempty"`
	// Style: "bcd" (each digit in binary) or "true" (whole number in binary)
	Style string `json:"style,omitempty"`
	// Layout: "vertical" (bits top-to-bottom) or "horizontal" (bits left-to-right)
	Layout string `json:"layout,omitempty"`
	// ShowLabels: show H:M:S labels
	ShowLabels bool `json:"show_labels,omitempty"`
	// ShowHint: show decimal digits alongside binary
	ShowHint bool `json:"show_hint,omitempty"`
	// DotSize: diameter of dots in pixels (default: 4)
	DotSize int `json:"dot_size,omitempty"`
	// DotSpacing: gap between dots in pixels (default: 2)
	DotSpacing int `json:"dot_spacing,omitempty"`
	// DotStyle: "circle" or "square" (default: "circle")
	DotStyle string `json:"dot_style,omitempty"`
	// OnColor: color for "on" bits (default: 255)
	OnColor *int `json:"on_color,omitempty"`
	// OffColor: color for "off" bits, 0 = invisible (default: 40)
	OffColor *int `json:"off_color,omitempty"`
	// Use12h: use 12-hour format instead of 24-hour (default: false)
	Use12h bool `json:"use_12h,omitempty"`
	// ShowAmPm: show AM/PM indicator bit (0=AM, 1=PM) when use_12h is true (default: false)
	ShowAmPm bool `json:"show_ampm,omitempty"`
}

// SegmentClockConfig represents seven-segment clock mode settings
type SegmentClockConfig struct {
	// Format: time format like "%H:%M" or "%H:%M:%S" (default: "%H:%M:%S")
	Format string `json:"format,omitempty"`
	// DigitHeight: height of digits in pixels (default: auto-fit)
	DigitHeight int `json:"digit_height,omitempty"`
	// SegmentThickness: thickness of segments in pixels (default: 2)
	SegmentThickness int `json:"segment_thickness,omitempty"`
	// SegmentStyle: shape of segments - "rectangle", "hexagon", "rounded" (default: "rectangle")
	SegmentStyle string `json:"segment_style,omitempty"`
	// DigitSpacing: gap between digits in pixels (default: 2)
	DigitSpacing int `json:"digit_spacing,omitempty"`
	// ColonStyle: "dots", "bar", or "none" (default: "dots")
	ColonStyle string `json:"colon_style,omitempty"`
	// ColonBlink: blink colon every second (default: true)
	ColonBlink *bool `json:"colon_blink,omitempty"`
	// OnColor: color for "on" segments (default: 255)
	OnColor *int `json:"on_color,omitempty"`
	// OffColor: color for "off" segments, 0 = invisible (default: 30)
	OffColor *int `json:"off_color,omitempty"`
	// Flip: flip animation settings
	Flip *FlipEffectConfig `json:"flip,omitempty"`
	// Use12h: use 12-hour format instead of 24-hour (default: false)
	Use12h bool `json:"use_12h,omitempty"`
	// ShowAmPm: show AM/PM indicator when use_12h is true (default: false)
	ShowAmPm bool `json:"show_ampm,omitempty"`
	// AmPmStyle: AM/PM indicator style - "dot" or "text" (default: "dot")
	AmPmStyle string `json:"ampm_style,omitempty"`
}

// FlipEffectConfig represents digit flip animation settings
type FlipEffectConfig struct {
	// Style: "none" (disabled), "fade" (crossfade between digits)
	Style string `json:"style,omitempty"`
	// Speed: animation duration in seconds (default: 0.15)
	Speed float64 `json:"speed,omitempty"`
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

// BeefwebConfig represents Beefweb widget settings (Foobar2000/DeaDBeeF)
// Format string is configured via text.format with placeholders:
// {artist}, {title}, {album}, {duration}, {position}, {state}
type BeefwebConfig struct {
	// ServerURL is the beefweb server URL (default: http://localhost:8880)
	ServerURL string `json:"server_url,omitempty"`
	// Placeholder configuration when player is not running or stopped
	Placeholder *BeefwebPlaceholderConfig `json:"placeholder,omitempty"`
}

// BeefwebAutoShowConfig represents events that trigger the beefweb widget to show
type BeefwebAutoShowConfig struct {
	// OnTrackChange - show widget when track changes (default: true)
	OnTrackChange *bool `json:"on_track_change,omitempty"`
	// OnPlay - show widget when playback starts
	OnPlay bool `json:"on_play,omitempty"`
	// OnPause - show widget when playback is paused
	OnPause bool `json:"on_pause,omitempty"`
	// OnStop - show widget when playback stops
	OnStop bool `json:"on_stop,omitempty"`
	// DurationSec - how long to show the widget (seconds, default: 5)
	DurationSec float64 `json:"duration_sec,omitempty"`
}

// BeefwebPlaceholderConfig represents what to show when player is not running
type BeefwebPlaceholderConfig struct {
	// Mode: "text" for custom text, "hide" to hide widget
	Mode string `json:"mode,omitempty"`
	// Text to display when mode is "text"
	Text string `json:"text,omitempty"`
}

// SpotifyAuthConfig contains Spotify authentication settings
type SpotifyAuthConfig struct {
	// Mode: "oauth" (interactive PKCE flow) or "manual" (token in config)
	// Default: "oauth"
	Mode string `json:"mode,omitempty"`
	// ClientID: Spotify application Client ID (required)
	ClientID string `json:"client_id"`
	// AccessToken: pre-obtained access token (for manual mode only)
	AccessToken string `json:"access_token,omitempty"`
	// RefreshToken: pre-obtained refresh token (for manual mode only)
	RefreshToken string `json:"refresh_token,omitempty"`
	// TokenPath: path to token storage file (default: spotify_token.json)
	TokenPath string `json:"token_path,omitempty"`
	// CallbackPort: local port for OAuth callback server (default: 8888)
	CallbackPort int `json:"callback_port,omitempty"`
}

// SpotifyConfig contains Spotify widget display settings
type SpotifyConfig struct {
	// Placeholder configuration when Spotify is not playing
	Placeholder *SpotifyPlaceholderConfig `json:"placeholder,omitempty"`
}

// SpotifyPlaceholderConfig represents what to show when Spotify is not playing
type SpotifyPlaceholderConfig struct {
	// Mode: "text" for custom text, "icon" for Spotify icon, "hide" to hide widget
	Mode string `json:"mode,omitempty"`
	// Text to display when mode is "text"
	Text string `json:"text,omitempty"`
}

// SpotifyAutoShowConfig represents events that trigger the Spotify widget to show
type SpotifyAutoShowConfig struct {
	// OnTrackChange - show widget when track changes (default: true)
	OnTrackChange *bool `json:"on_track_change,omitempty"`
	// OnPlay - show widget when playback starts
	OnPlay bool `json:"on_play,omitempty"`
	// OnPause - show widget when playback is paused
	OnPause bool `json:"on_pause,omitempty"`
	// OnStop - show widget when playback stops
	OnStop bool `json:"on_stop,omitempty"`
	// DurationSec - how long to show the widget (seconds, default: 5)
	DurationSec float64 `json:"duration_sec,omitempty"`
}

// ScrollConfig represents text scrolling settings
type ScrollConfig struct {
	// Enabled explicitly enables/disables scrolling
	Enabled bool `json:"enabled,omitempty"`
	// Direction: "left", "right", "up", "down"
	Direction ScrollDirection `json:"direction,omitempty"`
	// Speed in pixels per second
	Speed float64 `json:"speed,omitempty"`
	// Mode: "continuous" (loop), "bounce" (reverse at edges), "pause_ends" (pause at start/end)
	Mode ScrollMode `json:"mode,omitempty"`
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

// DoomConfig represents DOOM widget display settings
type DoomConfig struct {
	// RenderMode: grayscale conversion mode
	// "normal" - standard luminance (default)
	// "contrast" - auto-contrast stretching
	// "posterize" - reduce to N gray levels
	// "threshold" - pure black/white
	// "dither" - ordered dithering (Bayer matrix)
	// "gamma" - gamma correction with contrast boost
	RenderMode string `json:"render_mode,omitempty"`
	// PosterizeLevels: number of gray levels for posterize mode (2-16, default: 4)
	PosterizeLevels int `json:"posterize_levels,omitempty"`
	// ThresholdValue: cutoff value for threshold mode (0-255, default: 128)
	ThresholdValue int `json:"threshold_value,omitempty"`
	// Gamma: gamma value for gamma mode (0.1-3.0, default: 1.5)
	Gamma float64 `json:"gamma,omitempty"`
	// ContrastBoost: contrast multiplier for gamma mode (1.0-3.0, default: 1.2)
	ContrastBoost float64 `json:"contrast_boost,omitempty"`
	// DitherSize: Bayer matrix size for dither mode (2, 4, or 8, default: 4)
	DitherSize int `json:"dither_size,omitempty"`
}

// BatteryConfig represents Battery widget settings
// Modes: "icon" (compact tray-style), "battery" (progressbar shape), "text", "bar", "gauge", "graph"
// Note: Power status indicator display is controlled by PowerStatusConfig (power_status field)
type BatteryConfig struct {
	// Orientation: "horizontal" or "vertical" for battery/icon modes (default: "horizontal")
	Orientation string `json:"orientation,omitempty"`
	// ShowPercentage: show percentage text (default: true)
	ShowPercentage *bool `json:"show_percentage,omitempty"`
	// LowThreshold: percentage considered low (default: 20)
	LowThreshold int `json:"low_threshold,omitempty"`
	// CriticalThreshold: percentage considered critical (default: 10)
	CriticalThreshold int `json:"critical_threshold,omitempty"`
	// Colors: custom colors for battery states (uses pointers to allow 0/black)
	Colors *BatteryColorsConfig `json:"colors,omitempty"`
}

// BatteryColorsConfig represents color settings for battery widget
// Uses pointers to distinguish "not set" from "set to 0 (black)"
type BatteryColorsConfig struct {
	// Normal: fill color when battery level is normal (default: 255)
	Normal *int `json:"normal,omitempty"`
	// Low: fill color when battery is low (default: 200)
	Low *int `json:"low,omitempty"`
	// Critical: fill color when battery is critical (default: 150)
	Critical *int `json:"critical,omitempty"`
	// Charging: charging indicator color (default: 255)
	Charging *int `json:"charging,omitempty"`
	// Background: inner background color (default: 0)
	Background *int `json:"background,omitempty"`
	// Border: outline color (default: 255)
	Border *int `json:"border,omitempty"`
}

// PowerStatusConfig represents power status indicator display settings
// Used by battery widget to control how charging/plugged/economy indicators are shown
type PowerStatusConfig struct {
	// ShowEconomy: display mode for economy/power saver indicator
	// Values: "always", "never", "notify", "blink", "notify_blink" (default: "blink")
	ShowEconomy string `json:"show_economy,omitempty"`
	// ShowCharging: display mode for charging indicator
	// Values: "always", "never", "notify", "blink", "notify_blink" (default: "always")
	ShowCharging string `json:"show_charging,omitempty"`
	// ShowPlugged: display mode for AC power indicator
	// Values: "always", "never", "notify", "blink", "notify_blink" (default: "always")
	ShowPlugged string `json:"show_plugged,omitempty"`
	// NotifyDuration: seconds to show indicator in "notify" modes (default: 60)
	NotifyDuration int `json:"notify_duration,omitempty"`
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

// GameOfLifeConfig represents Conway's Game of Life widget settings
type GameOfLifeConfig struct {
	// Rules: cellular automaton rules in "B3/S23" format (default: "B3/S23" - standard Conway)
	// B = birth (neighbor counts that cause cell birth)
	// S = survival (neighbor counts that allow cell survival)
	// Examples: "B3/S23" (Conway), "B36/S23" (HighLife), "B1357/S1357" (Replicator)
	Rules string `json:"rules,omitempty"`
	// WrapEdges: wrap edges to form a torus topology (default: true)
	WrapEdges *bool `json:"wrap_edges,omitempty"`
	// InitialPattern: starting pattern (default: "random")
	// Values: "random", "clear", "glider", "r_pentomino", "acorn", "diehard", "lwss", "pulsar", "glider_gun"
	InitialPattern string `json:"initial_pattern,omitempty"`
	// RandomDensity: probability of cell being alive in random pattern (0.0-1.0, default: 0.3)
	RandomDensity float64 `json:"random_density,omitempty"`
	// CellSize: pixels per cell (1-4, default: 1)
	CellSize int `json:"cell_size,omitempty"`
	// TrailEffect: enable fading trail when cells die (default: true)
	TrailEffect *bool `json:"trail_effect,omitempty"`
	// TrailDecay: brightness decay per frame for dead cells (1-255, default: 30)
	TrailDecay int `json:"trail_decay,omitempty"`
	// CellColor: brightness of alive cells (1-255, default: 255)
	CellColor int `json:"cell_color,omitempty"`
	// RestartTimeout: seconds to wait before restarting when simulation ends (default: 3.0)
	// 0 = restart immediately, -1 = never restart (stay in final state)
	RestartTimeout *float64 `json:"restart_timeout,omitempty"`
	// RestartMode: how to restart (default: "reset")
	// "reset" = restart with initial_pattern, "inject" = add cells to existing grid, "random" = always use random
	RestartMode string `json:"restart_mode,omitempty"`
}

// HyperspaceConfig represents Star Wars hyperspace effect widget settings
type HyperspaceConfig struct {
	// StarCount: number of stars (default: 100)
	StarCount int `json:"star_count,omitempty"`
	// Speed: base star movement speed (default: 0.02)
	Speed float64 `json:"speed,omitempty"`
	// MaxSpeed: maximum speed during hyperspace jump (default: 0.5)
	MaxSpeed float64 `json:"max_speed,omitempty"`
	// TrailLength: trail length multiplier (default: 1.0)
	TrailLength float64 `json:"trail_length,omitempty"`
	// CenterX: focal point X coordinate (default: center of widget)
	CenterX *int `json:"center_x,omitempty"`
	// CenterY: focal point Y coordinate (default: center of widget)
	CenterY *int `json:"center_y,omitempty"`
	// StarColor: brightness of stars (1-255, default: 255)
	StarColor int `json:"star_color,omitempty"`
	// Mode: "continuous" (always hyperspeed) or "cycle" (idle -> jump -> hyperspace -> exit)
	Mode string `json:"mode,omitempty"`
	// IdleTime: seconds in idle/normal star mode before jump (cycle mode only, default: 5.0)
	IdleTime float64 `json:"idle_time,omitempty"`
	// TravelTime: seconds in hyperspace (cycle mode only, default: 3.0)
	TravelTime float64 `json:"travel_time,omitempty"`
	// Acceleration: speed change rate during jump/exit phases (default: 0.1)
	Acceleration float64 `json:"acceleration,omitempty"`
}

// StarWarsIntroConfig contains settings for the Star Wars intro crawl widget
type StarWarsIntroConfig struct {
	// Pre-intro phase: "A long time ago in a galaxy far, far away...."
	PreIntro *StarWarsPreIntroConfig `json:"pre_intro,omitempty"`

	// Logo phase: "STAR WARS" shrinking toward center
	Logo *StarWarsLogoConfig `json:"logo,omitempty"`

	// Background stars (visible during logo and crawl phases)
	Stars *StarWarsStarsConfig `json:"stars,omitempty"`

	// Crawl phase settings
	// Text: lines of text to display in the crawl
	Text []string `json:"text,omitempty"`
	// ScrollSpeed: how fast the text scrolls up (pixels per frame, default: 0.5)
	ScrollSpeed float64 `json:"scroll_speed,omitempty"`
	// Perspective: perspective strength (0.0 = none, 1.0 = strong, default: 0.7)
	Perspective float64 `json:"perspective,omitempty"`
	// Slant: text italic/slant angle in degrees (0.0 = upright, default: 15.0 to match perspective)
	Slant float64 `json:"slant,omitempty"`
	// FadeTop: where fade starts from top (0.0-1.0, default: 0.3)
	FadeTop float64 `json:"fade_top,omitempty"`
	// TextColor: brightness of text (1-255, default: 255)
	TextColor int `json:"text_color,omitempty"`
	// LineSpacing: pixels between lines (default: 8)
	LineSpacing int `json:"line_spacing,omitempty"`

	// General settings
	// Loop: whether to loop the entire sequence (default: true)
	Loop *bool `json:"loop,omitempty"`
	// PauseAtEnd: seconds to pause at end before looping (default: 3.0)
	PauseAtEnd float64 `json:"pause_at_end,omitempty"`
}

// StarWarsPreIntroConfig contains settings for the pre-intro text phase
type StarWarsPreIntroConfig struct {
	// Enabled: show the pre-intro phase (default: true)
	Enabled *bool `json:"enabled,omitempty"`
	// Text: the pre-intro message (default: "A long time ago in a galaxy far, far away....")
	Text string `json:"text,omitempty"`
	// Color: text brightness (1-255, default: 80 - bluish dim appearance)
	Color int `json:"color,omitempty"`
	// FadeIn: fade in duration in seconds (default: 2.0)
	FadeIn float64 `json:"fade_in,omitempty"`
	// Hold: hold duration in seconds after fade in (default: 2.0)
	Hold float64 `json:"hold,omitempty"`
	// FadeOut: fade out duration in seconds (default: 1.0)
	FadeOut float64 `json:"fade_out,omitempty"`
}

// StarWarsLogoConfig contains settings for the logo shrinking phase
type StarWarsLogoConfig struct {
	// Enabled: show the logo phase (default: true)
	Enabled *bool `json:"enabled,omitempty"`
	// Text: logo text, use \n for line breaks (default: "STAR\nWARS")
	Text string `json:"text,omitempty"`
	// Color: logo brightness (1-255, default: 255)
	Color int `json:"color,omitempty"`
	// HoldBefore: seconds to hold at full size before shrinking (default: 0.5)
	HoldBefore float64 `json:"hold_before,omitempty"`
	// ShrinkDuration: seconds for the shrink animation (default: 4.0)
	ShrinkDuration float64 `json:"shrink_duration,omitempty"`
	// FinalScale: scale at which logo disappears (0.0-1.0, default: 0.1)
	FinalScale float64 `json:"final_scale,omitempty"`
	// LineSpacing: pixels between logo lines (default: 1)
	LineSpacing int `json:"line_spacing,omitempty"`
}

// StarWarsStarsConfig contains settings for background stars
type StarWarsStarsConfig struct {
	// Enabled: show background stars (default: true)
	Enabled *bool `json:"enabled,omitempty"`
	// Count: number of stars (default: 50)
	Count int `json:"count,omitempty"`
	// Brightness: maximum star brightness (1-255, default: 200)
	Brightness int `json:"brightness,omitempty"`
}

// ClipboardWidgetConfig contains settings for the clipboard widget
type ClipboardWidgetConfig struct {
	// MaxLength: maximum characters to display (default: 100)
	MaxLength int `json:"max_length,omitempty"`
	// ShowType: show content type prefix (default: true)
	ShowType *bool `json:"show_type,omitempty"`
	// ScrollLongText: enable horizontal scrolling for long text (default: true)
	ScrollLongText *bool `json:"scroll_long_text,omitempty"`
	// PollIntervalMs: clipboard check interval in milliseconds (default: 500)
	PollIntervalMs int `json:"poll_interval_ms,omitempty"`
	// ShowInvisible: show invisible characters as escape sequences (default: false)
	// \n -> \n, \r -> \r, \t -> \t (literal backslash sequences)
	ShowInvisible bool `json:"show_invisible,omitempty"`
}

// ScreenMirrorConfig contains settings for the screen mirror widget
type ScreenMirrorConfig struct {
	// Display: which display to capture
	// - null: primary display (default)
	// - integer 0, 1, 2...: specific monitor by index
	// - integer -1: all monitors combined
	// - string: match monitor by name (partial, case-insensitive)
	Display *IntOrString `json:"display,omitempty"`
	// Region: rectangular region to capture (nil = full display)
	Region *ScreenMirrorRegionConfig `json:"region,omitempty"`
	// Window: window to capture (nil = capture display/region)
	Window *ScreenMirrorWindowConfig `json:"window,omitempty"`
	// ScaleMode: how to scale captured content - "fit", "stretch", "crop" (default: "fit")
	ScaleMode string `json:"scale_mode,omitempty"`
	// FPS: capture framerate, 1-30 (default: 15)
	FPS int `json:"fps,omitempty"`
	// DitherMode: dithering for monochrome output - "floyd_steinberg", "ordered", "none" (default: "floyd_steinberg")
	DitherMode string `json:"dither_mode,omitempty"`
}

// ScreenMirrorRegionConfig defines a rectangular region to capture
type ScreenMirrorRegionConfig struct {
	// X: left edge in screen coordinates
	X int `json:"x"`
	// Y: top edge in screen coordinates
	Y int `json:"y"`
	// W: region width in pixels
	W int `json:"w"`
	// H: region height in pixels
	H int `json:"h"`
}

// ScreenMirrorWindowConfig defines which window to capture
type ScreenMirrorWindowConfig struct {
	// Title: substring to match in window title
	Title string `json:"title,omitempty"`
	// Class: window class name to match (Windows-specific)
	Class string `json:"class,omitempty"`
	// Active: capture the currently active/focused window
	Active bool `json:"active,omitempty"`
}

// HackerCodeConfig contains settings for the hacker code widget.
// Font is configured via standard text.font setting ("3x5", "5x7", "pixel3x5", "pixel5x7").
type HackerCodeConfig struct {
	// Style: code generation style - "c", "asm", "mixed" (default: "c")
	Style string `json:"style,omitempty"`
	// TypingSpeed: characters per second, either integer or {"min": N, "max": M} for random range (default: 50)
	TypingSpeed *IntOrRange `json:"typing_speed,omitempty"`
	// LineDelay: milliseconds pause at end of line (default: 200)
	LineDelay int `json:"line_delay,omitempty"`
	// ShowCursor: show blinking cursor (default: true)
	ShowCursor *bool `json:"show_cursor,omitempty"`
	// CursorBlinkMs: cursor blink interval in milliseconds (default: 500)
	CursorBlinkMs int `json:"cursor_blink_ms,omitempty"`
	// IndentSize: spaces per indent level (default: 2)
	IndentSize int `json:"indent_size,omitempty"`
}

// SeparatorConfig represents a separator line between elements
type SeparatorConfig struct {
	// Color: separator color (-1 = disabled, 0-255 = grayscale, default: 128)
	Color int `json:"color,omitempty"`
	// Thickness: separator thickness in pixels (default: 1)
	Thickness int `json:"thickness,omitempty"`
}

// TransitionConfig represents transition effect settings
// Transition types: "none", "push_left", "push_right", "push_up", "push_down",
// "slide_left", "slide_right", "slide_up", "slide_down",
// "dissolve_fade", "dissolve_pixel", "dissolve_dither",
// "box_in", "box_out", "clock_wipe", "random"
type TransitionConfig struct {
	// In: transition effect when showing (default: "none")
	In string `json:"in,omitempty"`
	// InSpeed: transition duration in seconds (default: 0.5)
	InSpeed float64 `json:"in_speed,omitempty"`
	// Out: transition effect when hiding (default: "none")
	Out string `json:"out,omitempty"`
	// OutSpeed: transition duration in seconds (default: 0.5)
	OutSpeed float64 `json:"out_speed,omitempty"`
}

// TelegramAuthConfig contains Telegram API authentication credentials
type TelegramAuthConfig struct {
	// APIID: Telegram API ID from my.telegram.org (required)
	APIID int `json:"api_id"`
	// APIHash: Telegram API Hash from my.telegram.org (required)
	APIHash string `json:"api_hash"`
	// PhoneNumber: phone number in international format, e.g., "+1234567890" (required)
	PhoneNumber string `json:"phone_number"`
	// SessionPath: path to session file (default: telegram/{api_id}_{phone}.session)
	SessionPath string `json:"session_path,omitempty"`
}

// TelegramFiltersConfig contains filter settings for all chat types
type TelegramFiltersConfig struct {
	// PrivateChats: private message filter settings
	PrivateChats *TelegramChatFilterConfig `json:"private_chats,omitempty"`
	// Groups: group chat filter settings
	Groups *TelegramChatFilterConfig `json:"groups,omitempty"`
	// Channels: channel filter settings
	Channels *TelegramChatFilterConfig `json:"channels,omitempty"`
}

// TelegramChatFilterConfig contains filter settings for a chat type
type TelegramChatFilterConfig struct {
	// Enabled: enable messages from this chat type (default: true for private, false for groups/channels)
	Enabled *bool `json:"enabled,omitempty"`
	// Whitelist: always include these chat IDs, even if this chat type is disabled
	Whitelist []string `json:"whitelist,omitempty"`
	// Blacklist: never include these chat IDs, even if this chat type is enabled
	Blacklist []string `json:"blacklist,omitempty"`
	// PinnedMessages: include pinned message notifications (default: true)
	PinnedMessages *bool `json:"pinned_messages,omitempty"`
}

// TelegramBadgeConfig contains settings for badge display mode
type TelegramBadgeConfig struct {
	// Blink: blink mode - "never", "always", "progressive" (default: "never")
	// "progressive" increases blink frequency based on unread count (1/sec at 1 msg, 10/sec at 10+ msgs)
	Blink BlinkMode `json:"blink,omitempty"`
	// Colors: custom colors for the badge icon
	Colors *TelegramBadgeColorsConfig `json:"colors,omitempty"`
}

// TelegramBadgeColorsConfig contains color settings for badge icon
type TelegramBadgeColorsConfig struct {
	// Foreground: color for icon shape (0-255, or -1 for transparent)
	Foreground int `json:"foreground"`
	// Background: color for icon background (0-255, or -1 for transparent)
	Background int `json:"background"`
}

// TelegramAppearanceConfig contains appearance settings for notifications
type TelegramAppearanceConfig struct {
	// Header: header element settings (sender/chat name)
	Header *TelegramElementConfig `json:"header,omitempty"`
	// Message: message text element settings
	Message *TelegramElementConfig `json:"message,omitempty"`
	// Separator: separator line between header and message
	Separator *SeparatorConfig `json:"separator,omitempty"`
	// Timeout: seconds to show notification (0 = show until next message, default: 0)
	Timeout int `json:"timeout,omitempty"`
	// Transitions: transition effects for showing/hiding notifications
	Transitions *TransitionConfig `json:"transitions,omitempty"`
}

// TelegramElementConfig contains settings for a notification element (header or message)
type TelegramElementConfig struct {
	// Enabled: show this element (default: true)
	Enabled *bool `json:"enabled,omitempty"`
	// Blink: make element blink (default: false)
	Blink bool `json:"blink,omitempty"`
	// Text: text rendering settings (font, size, alignment)
	Text *TextConfig `json:"text,omitempty"`
	// Scroll: text scrolling settings when text doesn't fit
	Scroll *ScrollConfig `json:"scroll,omitempty"`
	// WordBreak: how to break lines - "normal" (break on spaces) or "break-all" (break anywhere)
	WordBreak string `json:"word_break,omitempty"`
}

// ClaudeCodeConfig contains settings for the Claude Code notification widget.
// This widget notifies about Claude Code's activity with the Clawd mascot.
//
// Status is received via HTTP POST to /api/claude-status endpoint.
// Configure Claude Code hooks to POST status updates to SteelClock's web server.
type ClaudeCodeConfig struct {
	// IntroDuration: intro animation duration in seconds (default: 3, 0 = no intro)
	IntroDuration int `json:"intro_duration,omitempty"`
	// IdleAnimations: enable idle animations like blinking (default: true)
	IdleAnimations *bool `json:"idle_animations,omitempty"`
	// IntroTitle: text shown during intro, supports \n for multiple lines
	IntroTitle string `json:"intro_title,omitempty"`
	// AutoHide: hide widget when there's no notification to show (default: true)
	AutoHide *bool `json:"auto_hide,omitempty"`
	// SpriteSize: Clawd sprite size - "large" (34x20), "medium" (17x10), "small" (9x6)
	// Default: "medium"
	SpriteSize string `json:"sprite_size,omitempty"`
	// ShowText: show status text next to Clawd (default: true)
	// When false, only Clawd animations are shown
	ShowText *bool `json:"show_text,omitempty"`
	// ShowTimer: show elapsed time during thinking/tool states (default: true)
	ShowTimer *bool `json:"show_timer,omitempty"`
	// ShowSubagent: show small Clawd sprite when subagent (Task) is running (default: true)
	ShowSubagent *bool `json:"show_subagent,omitempty"`
	// Notify: notification duration per state in seconds
	// 0 = don't show, -1 = show until next state change, N = show for N seconds
	Notify *ClaudeCodeNotifyConfig `json:"notify,omitempty"`
}

// ClaudeCodeNotifyConfig defines notification duration for each Claude Code state.
// Values: 0 = don't notify, -1 = notify until next event, N = notify for N seconds
type ClaudeCodeNotifyConfig struct {
	// Thinking: duration when Claude is thinking (default: 0)
	Thinking *int `json:"thinking,omitempty"`
	// Tool: duration when Claude is using a tool (default: 2)
	Tool *int `json:"tool,omitempty"`
	// Success: duration when task completes successfully (default: -1)
	Success *int `json:"success,omitempty"`
	// Error: duration when an error occurs (default: -1)
	Error *int `json:"error,omitempty"`
	// Idle: duration when Claude is idle (default: 0)
	Idle *int `json:"idle,omitempty"`
	// NotRunning: duration when Claude Code is not running (default: 0)
	NotRunning *int `json:"not_running,omitempty"`
}

// BluetoothConfig contains settings for the Bluetooth device status widget.
// Each widget instance tracks a single Bluetooth device by MAC address.
type BluetoothConfig struct {
	// Address: Bluetooth MAC address of the device to track (required)
	Address string `json:"address"`
	// APIURL: bqc API host:port (default: "127.0.0.1:8765")
	APIURL string `json:"api_url,omitempty"`
	// Format: display format string with tokens: {icon}, {name}, {level}, {state},
	// {battery:N}, {battery_h:N}, {battery_v:N}, {bar:N}, {bar_h:N}, {bar_v:N}.
	// N is always the horizontal width in pixels; vertical height comes from the content area.
	// (default: "{icon} {name} {battery:20}")
	Format string `json:"format,omitempty"`
	// LowBatteryThreshold: battery percentage at or below which the indicator blinks (0 = disabled, default: 0)
	LowBatteryThreshold int `json:"low_battery_threshold,omitempty"`
}
