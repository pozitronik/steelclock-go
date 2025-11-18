package config

// Config represents the complete SteelClock configuration
type Config struct {
	GameName         string         `json:"game_name"`
	GameDisplayName  string         `json:"game_display_name"`
	RefreshRateMs    int            `json:"refresh_rate_ms"`
	UnregisterOnExit bool           `json:"unregister_on_exit,omitempty"`
	BundledFontURL   string         `json:"bundled_font_url,omitempty"`
	Display          DisplayConfig  `json:"display"`
	Layout           *LayoutConfig  `json:"layout,omitempty"`
	Widgets          []WidgetConfig `json:"widgets"`
}

// DisplayConfig represents display settings
type DisplayConfig struct {
	Width           int `json:"width"`
	Height          int `json:"height"`
	BackgroundColor int `json:"background_color"`
}

// LayoutConfig represents virtual canvas layout settings
type LayoutConfig struct {
	Type          string `json:"type"`
	VirtualWidth  int    `json:"virtual_width,omitempty"`
	VirtualHeight int    `json:"virtual_height,omitempty"`
}

// WidgetConfig represents a widget configuration
type WidgetConfig struct {
	Type       string           `json:"type"`
	ID         string           `json:"id"`
	Enabled    bool             `json:"enabled"`
	Position   PositionConfig   `json:"position"`
	Style      StyleConfig      `json:"style"`
	Properties WidgetProperties `json:"properties"`
}

// PositionConfig represents widget position and size
type PositionConfig struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	W      int `json:"w"`
	H      int `json:"h"`
	ZOrder int `json:"z_order"`
}

// StyleConfig represents widget styling
type StyleConfig struct {
	BackgroundColor   int  `json:"background_color"`
	BackgroundOpacity int  `json:"background_opacity"`
	Border            bool `json:"border"`
	BorderColor       int  `json:"border_color"`
}

// WidgetProperties contains all possible widget properties
// Different widget types use different subsets of these fields
type WidgetProperties struct {
	// Common properties
	UpdateInterval  float64 `json:"update_interval,omitempty"`
	Font            string  `json:"font,omitempty"`
	FontSize        int     `json:"font_size,omitempty"`
	HorizontalAlign string  `json:"horizontal_align,omitempty"`
	VerticalAlign   string  `json:"vertical_align,omitempty"`
	Padding         int     `json:"padding,omitempty"`
	AutoHide        bool    `json:"auto_hide,omitempty"`
	AutoHideTimeout float64 `json:"auto_hide_timeout,omitempty"` // seconds

	// Clock widget
	Format string `json:"format,omitempty"`

	// CPU/Memory/Network/Disk widgets
	DisplayMode   string `json:"display_mode,omitempty"`
	FillColor     int    `json:"fill_color,omitempty"`
	BarBorder     bool   `json:"bar_border,omitempty"`
	BarMargin     int    `json:"bar_margin,omitempty"`
	HistoryLength int    `json:"history_length,omitempty"`

	// CPU widget
	PerCore  bool `json:"per_core,omitempty"`
	MaxCores int  `json:"max_cores,omitempty"`

	// Network widget
	Interface      *string `json:"interface"`
	DynamicScaling bool    `json:"dynamic_scaling,omitempty"`
	MaxSpeedMbps   float64 `json:"max_speed_mbps,omitempty"`
	SpeedUnit      string  `json:"speed_unit,omitempty"`
	RxColor        int     `json:"rx_color,omitempty"`
	TxColor        int     `json:"tx_color,omitempty"`
	RxNeedleColor  int     `json:"rx_needle_color,omitempty"`
	TxNeedleColor  int     `json:"tx_needle_color,omitempty"`

	// Disk widget
	DiskName   *string `json:"disk_name"`
	ReadColor  int     `json:"read_color,omitempty"`
	WriteColor int     `json:"write_color,omitempty"`

	// Keyboard widget
	Spacing           int    `json:"spacing,omitempty"`
	CapsLockOn        string `json:"caps_lock_on,omitempty"`
	CapsLockOff       string `json:"caps_lock_off,omitempty"`
	NumLockOn         string `json:"num_lock_on,omitempty"`
	NumLockOff        string `json:"num_lock_off,omitempty"`
	ScrollLockOn      string `json:"scroll_lock_on,omitempty"`
	ScrollLockOff     string `json:"scroll_lock_off,omitempty"`
	IndicatorColorOn  int    `json:"indicator_color_on,omitempty"`
	IndicatorColorOff int    `json:"indicator_color_off,omitempty"`

	// Volume widget
	GaugeColor        int  `json:"gauge_color,omitempty"`
	GaugeNeedleColor  int  `json:"gauge_needle_color,omitempty"`
	TriangleFillColor int  `json:"triangle_fill_color,omitempty"`
	TriangleBorder    bool `json:"triangle_border,omitempty"`

	// Volume meter widget
	ClippingColor       int     `json:"clipping_color,omitempty"`
	LeftChannelColor    int     `json:"left_channel_color,omitempty"`
	RightChannelColor   int     `json:"right_channel_color,omitempty"`
	StereoMode          bool    `json:"stereo_mode,omitempty"`
	UseDBScale          bool    `json:"use_db_scale,omitempty"`
	ShowClipping        bool    `json:"show_clipping,omitempty"`
	ClippingThreshold   float64 `json:"clipping_threshold,omitempty"`
	SilenceThreshold    float64 `json:"silence_threshold,omitempty"`
	DecayRate           float64 `json:"decay_rate,omitempty"`
	ShowPeak            bool    `json:"show_peak,omitempty"`
	ShowPeakHold        bool    `json:"show_peak_hold,omitempty"`
	PeakHoldTime        float64 `json:"peak_hold_time,omitempty"`
	AutoHideOnSilence   bool    `json:"auto_hide_on_silence,omitempty"`
	AutoHideSilenceTime float64 `json:"auto_hide_silence_time,omitempty"`
}
