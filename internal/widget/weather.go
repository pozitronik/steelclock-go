package widget

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// Weather condition codes for icon mapping
const (
	WeatherClear        = "clear"
	WeatherPartlyCloudy = "partly_cloudy"
	WeatherCloudy       = "cloud"
	WeatherRain         = "rain"
	WeatherDrizzle      = "drizzle"
	WeatherSnow         = "snow"
	WeatherStorm        = "storm"
	WeatherFog          = "fog"
)

// AQI levels
const (
	AQIGood               = "Good"
	AQIModerate           = "Moderate"
	AQIUnhealthySensitive = "Unhealthy for Sensitive"
	AQIUnhealthy          = "Unhealthy"
	AQIVeryUnhealthy      = "Very Unhealthy"
	AQIHazardous          = "Hazardous"
)

// UV levels
const (
	UVLow      = "Low"
	UVModerate = "Moderate"
	UVHigh     = "High"
	UVVeryHigh = "Very High"
	UVExtreme  = "Extreme"
)

// TokenType represents the type of format token
type TokenType int

const (
	TokenLiteral TokenType = iota // Plain text
	TokenText                     // Text-based token (temp, humidity, etc.)
	TokenIcon                     // Icon token (icon, aqi_icon, etc.)
	TokenLarge                    // Large expanding token (forecast:graph, etc.)
)

// Token represents a parsed token from the format string
type Token struct {
	Type    TokenType
	Name    string // Token name without braces
	Param   string // Optional parameter (e.g., "24" in {icon:24})
	Literal string // For literal tokens, the text content
}

// WeatherData holds the current weather information
type WeatherData struct {
	Temperature   float64
	FeelsLike     float64
	Condition     string // One of the Weather* constants
	Description   string // Human-readable description
	Humidity      int
	WindSpeed     float64
	WindDirection string
	Pressure      float64
	Visibility    float64
	Sunrise       time.Time
	Sunset        time.Time
}

// AirQualityData holds air quality information
type AirQualityData struct {
	AQI   int     // Air Quality Index (1-5 scale for EU, converted to US AQI)
	Level string  // AQI level description
	PM25  float64 // PM2.5 concentration
	PM10  float64 // PM10 concentration
}

// UVIndexData holds UV index information
type UVIndexData struct {
	Index float64 // UV index value
	Level string  // UV level description
}

// ForecastPoint holds weather data for a single time point
type ForecastPoint struct {
	Time        time.Time
	Temperature float64
	Condition   string
	Description string
}

// ForecastData holds forecast information
type ForecastData struct {
	Hourly []ForecastPoint // Hourly forecast (next 24-48 hours)
	Daily  []ForecastPoint // Daily forecast (next 3-7 days)
}

// WeatherWidget displays weather information using a format string
type WeatherWidget struct {
	*BaseWidget
	// Configuration
	provider      string
	apiKey        string
	city          string
	lat           float64
	lon           float64
	units         string
	iconSize      int
	formatCycle   []string // Format strings (single or multiple for cycling)
	cycleInterval int
	forecastHours int
	forecastDays  int
	scrollSpeed   float64
	aqiEnabled    bool
	uvEnabled     bool
	// Transition configuration
	transitionType  string
	transitionSpeed float64
	// Display settings
	fontSize   int
	fontName   string
	horizAlign string
	vertAlign  string
	padding    int
	fontFace   font.Face
	// Parsed tokens (cached)
	tokens        []Token
	currentFormat int // Index into formatCycle
	lastCycleTime time.Time
	// Transition state
	transitionActive    bool
	transitionProgress  float64
	transitionStartTime time.Time
	oldFrame            *image.Gray
	pendingFormat       int    // Format index to transition to
	activeTransition    string // Current transition being used (for random)
	pixelOrder          []int  // Pre-shuffled pixel indices for dissolve_pixel
	// Scroll state
	scrollOffset float64
	lastUpdate   time.Time
	// Weather data
	weather    *WeatherData
	forecast   *ForecastData
	airQuality *AirQualityData
	uvIndex    *UVIndexData
	lastError  string
	mu         sync.RWMutex
	// HTTP client
	httpClient *http.Client
}

// NewWeatherWidget creates a new weather widget
func NewWeatherWidget(cfg config.WidgetConfig) (*WeatherWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract common settings
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()

	// Weather-specific settings with defaults
	provider := "open-meteo"
	apiKey := ""
	city := ""
	lat := 0.0
	lon := 0.0
	units := "metric"
	iconSize := 16
	formatCycle := []string{"{icon} {temp}"}
	cycleInterval := 10
	transitionType := "none"
	transitionSpeed := 0.5
	forecastHours := 24
	forecastDays := 3
	scrollSpeed := 30.0
	aqiEnabled := false
	uvEnabled := false

	if cfg.Weather != nil {
		if cfg.Weather.Provider != "" {
			provider = cfg.Weather.Provider
		}
		apiKey = cfg.Weather.ApiKey
		if cfg.Weather.Location != nil {
			city = cfg.Weather.Location.City
			lat = cfg.Weather.Location.Lat
			lon = cfg.Weather.Location.Lon
		}
		if cfg.Weather.Units != "" {
			units = cfg.Weather.Units
		}
		if cfg.Weather.IconSize > 0 {
			iconSize = cfg.Weather.IconSize
		}
		if len(cfg.Weather.Format) > 0 {
			formatCycle = cfg.Weather.Format
		}
		// Cycle config
		if cfg.Weather.Cycle != nil {
			if cfg.Weather.Cycle.Interval >= 0 {
				cycleInterval = cfg.Weather.Cycle.Interval
			}
			if cfg.Weather.Cycle.Transition != "" {
				transitionType = cfg.Weather.Cycle.Transition
			}
			if cfg.Weather.Cycle.Speed > 0 {
				transitionSpeed = cfg.Weather.Cycle.Speed
			}
		}

		// New forecast config
		if cfg.Weather.Forecast != nil {
			if cfg.Weather.Forecast.Hours > 0 {
				forecastHours = cfg.Weather.Forecast.Hours
			}
			if cfg.Weather.Forecast.Days > 0 {
				forecastDays = cfg.Weather.Forecast.Days
			}
			if cfg.Weather.Forecast.ScrollSpeed > 0 {
				scrollSpeed = cfg.Weather.Forecast.ScrollSpeed
			}
		}

		// Backward compatibility for old config
		if cfg.Weather.ForecastHours > 0 && (cfg.Weather.Forecast == nil || cfg.Weather.Forecast.Hours == 0) {
			forecastHours = cfg.Weather.ForecastHours
		}
		if cfg.Weather.ForecastDays > 0 && (cfg.Weather.Forecast == nil || cfg.Weather.Forecast.Days == 0) {
			forecastDays = cfg.Weather.ForecastDays
		}
		if cfg.Weather.ScrollSpeed > 0 && (cfg.Weather.Forecast == nil || cfg.Weather.Forecast.ScrollSpeed == 0) {
			scrollSpeed = cfg.Weather.ScrollSpeed
		}
	}

	// Validate configuration
	if provider == "openweathermap" && apiKey == "" {
		return nil, fmt.Errorf("api_key is required for OpenWeatherMap provider")
	}

	// Location validation
	hasCity := city != ""
	hasCoords := lat != 0 || lon != 0
	if !hasCity && !hasCoords {
		return nil, fmt.Errorf("location is required: specify either city or lat/lon coordinates")
	}

	// Open-Meteo requires coordinates
	if provider == "open-meteo" && hasCity && !hasCoords {
		return nil, fmt.Errorf("open-meteo provider requires lat/lon coordinates; city name is only supported with openweathermap")
	}

	// Font settings
	fontSize := textSettings.FontSize
	if fontSize == 0 {
		fontSize = 10
	}
	fontName := textSettings.FontName

	// Load font
	fontFace, err := bitmap.LoadFont(fontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	// Auto-detect if AQI/UV tokens are used in any format
	for _, f := range formatCycle {
		if strings.Contains(f, "{aqi") {
			aqiEnabled = true
		}
		if strings.Contains(f, "{uv") {
			uvEnabled = true
		}
	}

	w := &WeatherWidget{
		BaseWidget:      base,
		provider:        provider,
		apiKey:          apiKey,
		city:            city,
		lat:             lat,
		lon:             lon,
		units:           units,
		iconSize:        iconSize,
		formatCycle:     formatCycle,
		cycleInterval:   cycleInterval,
		transitionType:  transitionType,
		transitionSpeed: transitionSpeed,
		forecastHours:   forecastHours,
		forecastDays:    forecastDays,
		scrollSpeed:     scrollSpeed,
		aqiEnabled:      aqiEnabled,
		uvEnabled:       uvEnabled,
		fontSize:        fontSize,
		fontName:        fontName,
		horizAlign:      textSettings.HorizAlign,
		vertAlign:       textSettings.VertAlign,
		padding:         padding,
		fontFace:        fontFace,
		lastCycleTime:   time.Now(),
		lastUpdate:      time.Now(),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Parse initial format (first format in cycle)
	w.tokens = w.parseFormat(formatCycle[0])

	return w, nil
}

// parseFormat parses a format string into tokens
func (w *WeatherWidget) parseFormat(format string) []Token {
	var tokens []Token

	// Handle newlines - split by \n and process each line
	format = strings.ReplaceAll(format, "\\n", "\n")

	// Regex to match {token} or {token:param}
	re := regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(?::([^}]*))?\}`)

	lastEnd := 0
	for _, match := range re.FindAllStringSubmatchIndex(format, -1) {
		// Add literal text before this token
		if match[0] > lastEnd {
			tokens = append(tokens, Token{
				Type:    TokenLiteral,
				Literal: format[lastEnd:match[0]],
			})
		}

		// Extract token name and optional parameter
		name := format[match[2]:match[3]]
		param := ""
		if match[4] >= 0 && match[5] >= 0 {
			param = format[match[4]:match[5]]
		}

		// Determine token type
		tokenType := w.getTokenType(name)
		tokens = append(tokens, Token{
			Type:  tokenType,
			Name:  name,
			Param: param,
		})

		lastEnd = match[1]
	}

	// Add any remaining literal text
	if lastEnd < len(format) {
		tokens = append(tokens, Token{
			Type:    TokenLiteral,
			Literal: format[lastEnd:],
		})
	}

	return tokens
}

// getTokenType determines the type of token by name
func (w *WeatherWidget) getTokenType(name string) TokenType {
	switch name {
	// Icon tokens
	case "icon", "aqi_icon", "uv_icon", "humidity_icon", "wind_icon", "wind_dir_icon":
		return TokenIcon
	// Large tokens (expand to fill space)
	case "forecast":
		return TokenLarge
	// Everything else is text
	default:
		// Check for day/hour icon tokens
		if strings.HasPrefix(name, "day") && strings.HasSuffix(name, "icon") {
			return TokenIcon
		}
		if strings.HasPrefix(name, "hour") && strings.HasSuffix(name, "icon") {
			return TokenIcon
		}
		return TokenText
	}
}

// Update fetches fresh weather data from the API
func (w *WeatherWidget) Update() error {
	var weather *WeatherData
	var forecast *ForecastData
	var aqi *AirQualityData
	var uv *UVIndexData
	var err error

	// Check if we need forecast data
	needForecast := w.needsForecast()

	switch w.provider {
	case "openweathermap":
		weather, forecast, err = w.fetchOpenWeatherMap(needForecast)
		if err == nil && w.aqiEnabled {
			aqi, _ = w.fetchOpenWeatherMapAQI()
		}
		// OpenWeatherMap doesn't have a free UV endpoint
	case "open-meteo":
		weather, forecast, aqi, uv, err = w.fetchOpenMeteo(needForecast)
	default:
		err = fmt.Errorf("unknown weather provider: %s", w.provider)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if err != nil {
		w.lastError = err.Error()
		log.Printf("Weather update error: %v", err)
		return nil // Don't return error to keep widget running
	}

	w.weather = weather
	w.forecast = forecast
	if aqi != nil {
		w.airQuality = aqi
	}
	if uv != nil {
		w.uvIndex = uv
	}
	w.lastError = ""

	return nil
}

// needsForecast checks if any format needs forecast data
func (w *WeatherWidget) needsForecast() bool {
	for _, f := range w.formatCycle {
		if strings.Contains(f, "{forecast") ||
			strings.Contains(f, "{day:") ||
			strings.Contains(f, "{hour:") {
			return true
		}
	}
	return false
}

// Render creates the weather widget image
func (w *WeatherWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Update scroll offset for animation
	now := time.Now()
	w.mu.Lock()
	elapsed := now.Sub(w.lastUpdate).Seconds()
	w.scrollOffset += w.scrollSpeed * elapsed
	w.lastUpdate = now

	// Handle transition progress
	if w.transitionActive {
		transitionElapsed := now.Sub(w.transitionStartTime).Seconds()
		w.transitionProgress = transitionElapsed / w.transitionSpeed
		if w.transitionProgress >= 1.0 {
			// Transition complete
			w.transitionProgress = 1.0
			w.transitionActive = false
			w.currentFormat = w.pendingFormat
			w.tokens = w.parseFormat(w.formatCycle[w.currentFormat])
			w.oldFrame = nil
			w.scrollOffset = 0
		}
	}

	// Handle format cycling (start new transition when it's time)
	if len(w.formatCycle) > 1 && w.cycleInterval > 0 && !w.transitionActive {
		cycleElapsed := now.Sub(w.lastCycleTime).Seconds()
		if cycleElapsed >= float64(w.cycleInterval) {
			w.startTransition(now)
		}
	}
	w.mu.Unlock()

	w.mu.RLock()
	weather := w.weather
	forecast := w.forecast
	aqi := w.airQuality
	uv := w.uvIndex
	lastError := w.lastError
	tokens := w.tokens
	scrollOffset := w.scrollOffset
	transitionActive := w.transitionActive
	transitionProgress := w.transitionProgress
	oldFrame := w.oldFrame
	activeTransition := w.activeTransition
	pendingFormat := w.pendingFormat
	pixelOrder := w.pixelOrder
	w.mu.RUnlock()

	// Handle error state
	if lastError != "" && weather == nil {
		bitmap.SmartDrawAlignedText(img, "ERR", w.fontFace, w.fontName, "center", "center", w.padding)
		return img, nil
	}

	// Handle no data yet
	if weather == nil {
		bitmap.SmartDrawAlignedText(img, "...", w.fontFace, w.fontName, "center", "center", w.padding)
		return img, nil
	}

	// If transition is active, render both frames and composite
	if transitionActive && oldFrame != nil {
		// Render new frame
		newFrame := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())
		newTokens := w.parseFormat(w.formatCycle[pendingFormat])
		w.renderTokens(newFrame, newTokens, weather, forecast, aqi, uv, 0) // Reset scroll for new format

		// Apply transition
		w.applyTransition(img, oldFrame, newFrame, transitionProgress, activeTransition, pixelOrder)
	} else {
		// Normal rendering
		w.renderTokens(img, tokens, weather, forecast, aqi, uv, scrollOffset)
	}

	// Draw border if enabled (always on top)
	style := w.GetStyle()
	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	return img, nil
}

// startTransition initiates a transition to the next format
func (w *WeatherWidget) startTransition(now time.Time) {
	pos := w.GetPosition()

	// Capture current frame
	w.oldFrame = bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())
	w.renderTokens(w.oldFrame, w.tokens, w.weather, w.forecast, w.airQuality, w.uvIndex, w.scrollOffset)

	// Set up transition
	w.pendingFormat = (w.currentFormat + 1) % len(w.formatCycle)
	w.transitionActive = true
	w.transitionProgress = 0.0
	w.transitionStartTime = now
	w.lastCycleTime = now

	// Select actual transition (handle "random")
	w.activeTransition = w.selectTransition()

	// Pre-generate pixel order for dissolve_pixel
	if w.activeTransition == "dissolve_pixel" {
		w.pixelOrder = w.generatePixelOrder(pos.W, pos.H)
	}
}

// selectTransition returns the actual transition type (handles "random")
func (w *WeatherWidget) selectTransition() string {
	if w.transitionType != "random" {
		return w.transitionType
	}

	// List of all transitions except "none" and "random"
	transitions := []string{
		"push_left", "push_right", "push_up", "push_down",
		"slide_left", "slide_right", "slide_up", "slide_down",
		"dissolve_fade", "dissolve_pixel", "dissolve_dither",
		"box_in", "box_out", "clock_wipe",
	}
	return transitions[rand.Intn(len(transitions))]
}

// generatePixelOrder creates a shuffled list of pixel indices for dissolve_pixel
func (w *WeatherWidget) generatePixelOrder(width, height int) []int {
	total := width * height
	order := make([]int, total)
	for i := 0; i < total; i++ {
		order[i] = i
	}
	// Fisher-Yates shuffle
	for i := total - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		order[i], order[j] = order[j], order[i]
	}
	return order
}

// applyTransition composites old and new frames based on transition type and progress
func (w *WeatherWidget) applyTransition(dst, oldFrame, newFrame *image.Gray, progress float64, transitionType string, pixelOrder []int) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	switch transitionType {
	case "none":
		// Instant switch at 50%
		if progress < 0.5 {
			copyGrayImage(dst, oldFrame)
		} else {
			copyGrayImage(dst, newFrame)
		}

	case "push_left":
		w.applyPushTransition(dst, oldFrame, newFrame, progress, -1, 0)
	case "push_right":
		w.applyPushTransition(dst, oldFrame, newFrame, progress, 1, 0)
	case "push_up":
		w.applyPushTransition(dst, oldFrame, newFrame, progress, 0, -1)
	case "push_down":
		w.applyPushTransition(dst, oldFrame, newFrame, progress, 0, 1)

	case "slide_left":
		w.applySlideTransition(dst, oldFrame, newFrame, progress, -1, 0)
	case "slide_right":
		w.applySlideTransition(dst, oldFrame, newFrame, progress, 1, 0)
	case "slide_up":
		w.applySlideTransition(dst, oldFrame, newFrame, progress, 0, -1)
	case "slide_down":
		w.applySlideTransition(dst, oldFrame, newFrame, progress, 0, 1)

	case "dissolve_fade":
		w.applyDissolveFade(dst, oldFrame, newFrame, progress)

	case "dissolve_pixel":
		w.applyDissolvePixel(dst, oldFrame, newFrame, progress, pixelOrder)

	case "dissolve_dither":
		w.applyDissolveDither(dst, oldFrame, newFrame, progress)

	case "box_in":
		w.applyBoxTransition(dst, oldFrame, newFrame, progress, true)
	case "box_out":
		w.applyBoxTransition(dst, oldFrame, newFrame, progress, false)

	case "clock_wipe":
		w.applyClockWipe(dst, oldFrame, newFrame, progress, width, height)

	default:
		// Unknown transition, just copy new frame
		copyGrayImage(dst, newFrame)
	}
}

// copyGrayImage copies src to dst
func copyGrayImage(dst, src *image.Gray) {
	copy(dst.Pix, src.Pix)
}

// applyPushTransition pushes old frame out while new frame comes in
func (w *WeatherWidget) applyPushTransition(dst, oldFrame, newFrame *image.Gray, progress float64, dirX, dirY int) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate offset based on progress
	offsetX := int(float64(width) * progress * float64(dirX))
	offsetY := int(float64(height) * progress * float64(dirY))

	// Draw old frame (being pushed out)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x - offsetX
			srcY := y - offsetY
			if srcX >= 0 && srcX < width && srcY >= 0 && srcY < height {
				dst.SetGray(x+bounds.Min.X, y+bounds.Min.Y, oldFrame.GrayAt(srcX+bounds.Min.X, srcY+bounds.Min.Y))
			}
		}
	}

	// Draw new frame (coming in)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// New frame enters from opposite side
			srcX := x - offsetX + width*dirX
			srcY := y - offsetY + height*dirY
			dstX := x + bounds.Min.X
			dstY := y + bounds.Min.Y
			if srcX >= 0 && srcX < width && srcY >= 0 && srcY < height {
				dst.SetGray(dstX, dstY, newFrame.GrayAt(srcX+bounds.Min.X, srcY+bounds.Min.Y))
			}
		}
	}
}

// applySlideTransition slides new frame over old frame
func (w *WeatherWidget) applySlideTransition(dst, oldFrame, newFrame *image.Gray, progress float64, dirX, dirY int) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// First, copy old frame
	copyGrayImage(dst, oldFrame)

	// Calculate new frame position (slides in from edge)
	var startX, startY int
	if dirX < 0 {
		startX = width - int(float64(width)*progress)
	} else if dirX > 0 {
		startX = int(float64(width)*progress) - width
	}
	if dirY < 0 {
		startY = height - int(float64(height)*progress)
	} else if dirY > 0 {
		startY = int(float64(height)*progress) - height
	}

	// Draw new frame on top
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x - startX
			srcY := y - startY
			if srcX >= 0 && srcX < width && srcY >= 0 && srcY < height {
				dstX := x + bounds.Min.X
				dstY := y + bounds.Min.Y
				dst.SetGray(dstX, dstY, newFrame.GrayAt(srcX+bounds.Min.X, srcY+bounds.Min.Y))
			}
		}
	}
}

// applyDissolveFade crossfades between old and new frames
func (w *WeatherWidget) applyDissolveFade(dst, oldFrame, newFrame *image.Gray, progress float64) {
	bounds := dst.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			oldVal := float64(oldFrame.GrayAt(x, y).Y)
			newVal := float64(newFrame.GrayAt(x, y).Y)
			blended := uint8(oldVal*(1-progress) + newVal*progress)
			dst.SetGray(x, y, color.Gray{Y: blended})
		}
	}
}

// applyDissolvePixel randomly switches pixels from old to new
func (w *WeatherWidget) applyDissolvePixel(dst, oldFrame, newFrame *image.Gray, progress float64, pixelOrder []int) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	total := len(pixelOrder)
	threshold := int(float64(total) * progress)

	// Copy old frame first
	copyGrayImage(dst, oldFrame)

	// Replace pixels up to threshold with new frame
	for i := 0; i < threshold && i < total; i++ {
		idx := pixelOrder[i]
		x := idx % width
		y := idx / width
		dst.SetGray(x+bounds.Min.X, y+bounds.Min.Y, newFrame.GrayAt(x+bounds.Min.X, y+bounds.Min.Y))
	}
}

// applyDissolveDither uses ordered dithering pattern for transition
func (w *WeatherWidget) applyDissolveDither(dst, oldFrame, newFrame *image.Gray, progress float64) {
	bounds := dst.Bounds()

	// 8x8 Bayer dithering matrix (values 0-63)
	bayer8x8 := [8][8]float64{
		{0, 32, 8, 40, 2, 34, 10, 42},
		{48, 16, 56, 24, 50, 18, 58, 26},
		{12, 44, 4, 36, 14, 46, 6, 38},
		{60, 28, 52, 20, 62, 30, 54, 22},
		{3, 35, 11, 43, 1, 33, 9, 41},
		{51, 19, 59, 27, 49, 17, 57, 25},
		{15, 47, 7, 39, 13, 45, 5, 37},
		{63, 31, 55, 23, 61, 29, 53, 21},
	}

	threshold := progress * 64.0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ditherVal := bayer8x8[y%8][x%8]
			if ditherVal < threshold {
				dst.SetGray(x, y, newFrame.GrayAt(x, y))
			} else {
				dst.SetGray(x, y, oldFrame.GrayAt(x, y))
			}
		}
	}
}

// applyBoxTransition reveals new frame through expanding/contracting box
func (w *WeatherWidget) applyBoxTransition(dst, oldFrame, newFrame *image.Gray, progress float64, boxIn bool) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	centerX := width / 2
	centerY := height / 2

	// Calculate box dimensions
	var boxW, boxH int
	if boxIn {
		// Box shrinks from edges, revealing new content
		boxW = int(float64(width) * (1 - progress) / 2)
		boxH = int(float64(height) * (1 - progress) / 2)
	} else {
		// Box expands from center, revealing new content
		boxW = int(float64(width) * progress / 2)
		boxH = int(float64(height) * progress / 2)
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dstX := x + bounds.Min.X
			dstY := y + bounds.Min.Y

			// Check if pixel is inside the box
			inBox := x >= centerX-boxW && x < centerX+boxW && y >= centerY-boxH && y < centerY+boxH

			if boxIn {
				// Box shrinks: outside box = new, inside box = old
				if inBox {
					dst.SetGray(dstX, dstY, oldFrame.GrayAt(dstX, dstY))
				} else {
					dst.SetGray(dstX, dstY, newFrame.GrayAt(dstX, dstY))
				}
			} else {
				// Box expands: inside box = new, outside box = old
				if inBox {
					dst.SetGray(dstX, dstY, newFrame.GrayAt(dstX, dstY))
				} else {
					dst.SetGray(dstX, dstY, oldFrame.GrayAt(dstX, dstY))
				}
			}
		}
	}
}

// applyClockWipe reveals new frame through clockwise radial sweep from 12 o'clock
func (w *WeatherWidget) applyClockWipe(dst, oldFrame, newFrame *image.Gray, progress float64, width, height int) {
	bounds := dst.Bounds()
	centerX := float64(width) / 2
	centerY := float64(height) / 2

	// Sweep angle: 0 = 12 o'clock, progress 1.0 = full circle (360 degrees)
	sweepAngle := progress * 2 * math.Pi

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dstX := x + bounds.Min.X
			dstY := y + bounds.Min.Y

			// Calculate angle from center to this pixel
			// atan2 returns angle from positive X axis, so adjust for 12 o'clock start
			dx := float64(x) - centerX
			dy := float64(y) - centerY

			// Angle from 12 o'clock (top), clockwise
			// atan2(dx, -dy) gives angle from top, positive clockwise
			pixelAngle := math.Atan2(dx, -dy)
			if pixelAngle < 0 {
				pixelAngle += 2 * math.Pi
			}

			// If pixel angle is less than sweep angle, show new frame
			if pixelAngle < sweepAngle {
				dst.SetGray(dstX, dstY, newFrame.GrayAt(dstX, dstY))
			} else {
				dst.SetGray(dstX, dstY, oldFrame.GrayAt(dstX, dstY))
			}
		}
	}
}

// renderTokens renders all tokens to the image
func (w *WeatherWidget) renderTokens(img *image.Gray, tokens []Token, weather *WeatherData, forecast *ForecastData, aqi *AirQualityData, uv *UVIndexData, scrollOffset float64) {
	pos := w.GetPosition()

	// Check if format contains newlines (multi-line layout)
	hasNewlines := false
	for _, t := range tokens {
		if t.Type == TokenLiteral && strings.Contains(t.Literal, "\n") {
			hasNewlines = true
			break
		}
	}

	if hasNewlines {
		w.renderMultiLine(img, tokens, weather, forecast, aqi, uv, scrollOffset)
		return
	}

	// Single line layout
	// First pass: measure all non-large tokens
	totalWidth := 0
	hasLargeToken := false

	for i := range tokens {
		t := &tokens[i]
		if t.Type == TokenLarge {
			hasLargeToken = true
			continue
		}
		totalWidth += w.measureToken(t, weather, forecast, aqi, uv)
	}

	// Calculate available space for large token
	availableWidth := pos.W - 2*w.padding - totalWidth
	if availableWidth < 0 {
		availableWidth = 0
	}

	// Calculate starting X based on horizontal alignment
	x := w.padding
	if !hasLargeToken {
		switch w.horizAlign {
		case "left":
			x = w.padding
		case "right":
			x = pos.W - totalWidth - w.padding
			if x < w.padding {
				x = w.padding
			}
		default: // center
			x = (pos.W - totalWidth) / 2
			if x < w.padding {
				x = w.padding
			}
		}
	}

	// Render tokens (vertical alignment handled by renderTokenInRect)
	for i := range tokens {
		t := &tokens[i]
		if t.Type == TokenLarge {
			// Render large token with available space
			w.renderLargeTokenInRect(img, t, x, 0, availableWidth, pos.H, weather, forecast, scrollOffset)
			x += availableWidth
		} else {
			width := w.renderTokenInRect(img, t, x, 0, pos.H, weather, forecast, aqi, uv)
			x += width
		}
	}
}

// renderMultiLine renders tokens with newline support
func (w *WeatherWidget) renderMultiLine(img *image.Gray, tokens []Token, weather *WeatherData, forecast *ForecastData, aqi *AirQualityData, uv *UVIndexData, scrollOffset float64) {
	pos := w.GetPosition()

	// Split tokens into lines
	var lines [][]Token
	var currentLine []Token

	for _, t := range tokens {
		if t.Type == TokenLiteral && strings.Contains(t.Literal, "\n") {
			// Split literal by newlines
			parts := strings.Split(t.Literal, "\n")
			for i, part := range parts {
				if part != "" {
					currentLine = append(currentLine, Token{Type: TokenLiteral, Literal: part})
				}
				if i < len(parts)-1 {
					lines = append(lines, currentLine)
					currentLine = nil
				}
			}
		} else {
			currentLine = append(currentLine, t)
		}
	}
	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	// Calculate line height
	lineHeight := pos.H / len(lines)
	if lineHeight < 8 {
		lineHeight = 8
	}

	// Calculate total content height and starting Y based on vertical alignment
	totalHeight := len(lines) * lineHeight
	startY := 0
	switch w.vertAlign {
	case "top":
		startY = w.padding
	case "bottom":
		startY = pos.H - totalHeight - w.padding
		if startY < w.padding {
			startY = w.padding
		}
	default: // center
		startY = (pos.H - totalHeight) / 2
		if startY < 0 {
			startY = 0
		}
	}

	// Render each line
	for i, line := range lines {
		y := startY + i*lineHeight
		w.renderLine(img, line, y, lineHeight, weather, forecast, aqi, uv, scrollOffset)
	}
}

// renderLine renders a single line of tokens
func (w *WeatherWidget) renderLine(img *image.Gray, tokens []Token, y, height int, weather *WeatherData, forecast *ForecastData, aqi *AirQualityData, uv *UVIndexData, scrollOffset float64) {
	pos := w.GetPosition()

	// Measure line width
	totalWidth := 0
	hasLargeToken := false
	var largeTokenIdx int

	for i, t := range tokens {
		if t.Type == TokenLarge {
			hasLargeToken = true
			largeTokenIdx = i
			continue
		}
		totalWidth += w.measureToken(&t, weather, forecast, aqi, uv)
	}

	// Calculate starting X based on horizontal alignment
	availableWidth := pos.W - 2*w.padding - totalWidth
	x := w.padding
	if !hasLargeToken {
		switch w.horizAlign {
		case "left":
			x = w.padding
		case "right":
			x = pos.W - totalWidth - w.padding
			if x < w.padding {
				x = w.padding
			}
		default: // center
			x = (pos.W - totalWidth) / 2
			if x < w.padding {
				x = w.padding
			}
		}
	}

	// Clamp height to image bounds
	actualHeight := height
	if y+height > pos.H {
		actualHeight = pos.H - y
	}
	if y >= pos.H || actualHeight <= 0 {
		return // Line is completely off-screen
	}

	// Render tokens on this line directly to the image at the correct y position
	// (Don't use SubImage as Go's SubImage preserves parent coordinates)
	for i := range tokens {
		t := &tokens[i]
		if t.Type == TokenLarge && i == largeTokenIdx {
			w.renderLargeTokenInRect(img, t, x, y, availableWidth, actualHeight, weather, forecast, scrollOffset)
			x += availableWidth
		} else if t.Type != TokenLarge {
			width := w.renderTokenInRectWithAlign(img, t, x, y, actualHeight, "center", weather, forecast, aqi, uv)
			x += width
		}
	}
}

// measureToken returns the width of a token
func (w *WeatherWidget) measureToken(t *Token, weather *WeatherData, forecast *ForecastData, aqi *AirQualityData, uv *UVIndexData) int {
	switch t.Type {
	case TokenLiteral:
		width, _ := bitmap.SmartMeasureText(t.Literal, w.fontFace, w.fontName)
		return width
	case TokenIcon:
		return w.getIconSize(t)
	case TokenText:
		text := w.getTokenText(t, weather, forecast, aqi, uv)
		width, _ := bitmap.SmartMeasureText(text, w.fontFace, w.fontName)
		return width
	case TokenLarge:
		return 0 // Large tokens are measured separately
	}
	return 0
}

// getIconSize returns the icon size for an icon token
func (w *WeatherWidget) getIconSize(t *Token) int {
	if t.Param != "" {
		// Parse size from parameter
		var size int
		fmt.Sscanf(t.Param, "%d", &size)
		if size > 0 {
			return size
		}
	}
	return w.iconSize
}

// renderTokenInRect renders a token within a rectangle using widget's vertical alignment
func (w *WeatherWidget) renderTokenInRect(img *image.Gray, t *Token, x, y, height int, weather *WeatherData, forecast *ForecastData, aqi *AirQualityData, uv *UVIndexData) int {
	return w.renderTokenInRectWithAlign(img, t, x, y, height, w.vertAlign, weather, forecast, aqi, uv)
}

// renderTokenInRectWithAlign renders a token within a rectangle with explicit vertical alignment
func (w *WeatherWidget) renderTokenInRectWithAlign(img *image.Gray, t *Token, x, y, height int, vAlign string, weather *WeatherData, forecast *ForecastData, aqi *AirQualityData, uv *UVIndexData) int {
	switch t.Type {
	case TokenLiteral:
		width, _ := bitmap.SmartMeasureText(t.Literal, w.fontFace, w.fontName)
		bitmap.SmartDrawTextInRect(img, t.Literal, w.fontFace, w.fontName, x, y, width+10, height, "left", vAlign, 0)
		return width

	case TokenIcon:
		return w.renderIconTokenWithAlign(img, t, x, y, height, vAlign, weather, forecast, aqi, uv)

	case TokenText:
		text := w.getTokenText(t, weather, forecast, aqi, uv)
		width, _ := bitmap.SmartMeasureText(text, w.fontFace, w.fontName)
		bitmap.SmartDrawTextInRect(img, text, w.fontFace, w.fontName, x, y, width+10, height, "left", vAlign, 0)
		return width

	case TokenLarge:
		// Large tokens are handled separately in renderLine/renderTokens
		return 0
	}
	return 0
}

// renderIconToken renders an icon token using widget's vertical alignment
func (w *WeatherWidget) renderIconToken(img *image.Gray, t *Token, x, y, height int, weather *WeatherData, forecast *ForecastData, aqi *AirQualityData, uv *UVIndexData) int {
	return w.renderIconTokenWithAlign(img, t, x, y, height, w.vertAlign, weather, forecast, aqi, uv)
}

// renderIconTokenWithAlign renders an icon token with explicit vertical alignment
func (w *WeatherWidget) renderIconTokenWithAlign(img *image.Gray, t *Token, x, y, height int, vAlign string, weather *WeatherData, forecast *ForecastData, aqi *AirQualityData, uv *UVIndexData) int {
	iconSize := w.getIconSize(t)

	var iconSet *glyphs.GlyphSet
	if iconSize >= 24 {
		iconSet = glyphs.WeatherIcons24x24
	} else {
		iconSet = glyphs.WeatherIcons16x16
	}

	var iconName string
	switch t.Name {
	case "icon":
		iconName = w.getIconName(weather.Condition)
	case "aqi_icon":
		iconName = w.getAQIIconName(aqi)
	case "uv_icon":
		iconName = w.getUVIconName(uv)
	case "humidity_icon":
		iconName = w.getHumidityIconName(weather.Humidity)
	case "wind_icon":
		iconName = w.getWindIconName(weather.WindSpeed)
	case "wind_dir_icon":
		iconName = w.getWindDirIconName(weather.WindDirection)
	default:
		// Handle day/hour icons
		iconName = w.getForecastIconName(t, forecast)
	}

	icon := glyphs.GetIcon(iconSet, iconName)
	if icon != nil {
		var iconY int
		switch vAlign {
		case "top":
			iconY = y + w.padding
		case "bottom":
			iconY = y + height - icon.Height - w.padding
		default: // center
			iconY = y + (height-icon.Height)/2
		}
		glyphs.DrawGlyph(img, icon, x, iconY, color.Gray{Y: 255})
		return icon.Width
	}

	return iconSize
}

// getTokenText returns the text value for a text token
func (w *WeatherWidget) getTokenText(t *Token, weather *WeatherData, forecast *ForecastData, aqi *AirQualityData, uv *UVIndexData) string {
	unit := "C"
	speedUnit := "m/s"
	if w.units == "imperial" {
		unit = "F"
		speedUnit = "mph"
	}

	switch t.Name {
	case "temp":
		return fmt.Sprintf("%.0f%s", weather.Temperature, unit)
	case "temp_raw":
		return fmt.Sprintf("%.0f", weather.Temperature)
	case "feels_like", "feels":
		return fmt.Sprintf("%.0f%s", weather.FeelsLike, unit)
	case "humidity":
		return fmt.Sprintf("%d%%", weather.Humidity)
	case "wind":
		return fmt.Sprintf("%.1f%s", weather.WindSpeed, speedUnit)
	case "wind_dir":
		return weather.WindDirection
	case "pressure":
		return fmt.Sprintf("%.0fhPa", weather.Pressure)
	case "description":
		return weather.Description
	case "condition":
		return getWeatherDescription(weather.Condition)
	case "visibility":
		if w.units == "imperial" {
			return fmt.Sprintf("%.1fmi", weather.Visibility/1609.34)
		}
		return fmt.Sprintf("%.0fkm", weather.Visibility/1000)
	case "sunrise":
		return weather.Sunrise.Format("15:04")
	case "sunset":
		return weather.Sunset.Format("15:04")
	case "daylight":
		remaining := time.Until(weather.Sunset)
		if remaining < 0 {
			return "0h"
		}
		return fmt.Sprintf("%dh %dm", int(remaining.Hours()), int(remaining.Minutes())%60)
	case "aqi":
		if aqi != nil {
			return fmt.Sprintf("%d", aqi.AQI)
		}
		return "-"
	case "aqi_text":
		if aqi != nil {
			return aqi.Level
		}
		return "-"
	case "pm25":
		if aqi != nil {
			return fmt.Sprintf("%.0f", aqi.PM25)
		}
		return "-"
	case "pm10":
		if aqi != nil {
			return fmt.Sprintf("%.0f", aqi.PM10)
		}
		return "-"
	case "uv":
		if uv != nil {
			return fmt.Sprintf("%.0f", uv.Index)
		}
		return "-"
	case "uv_text":
		if uv != nil {
			return uv.Level
		}
		return "-"
	default:
		// Handle day/hour tokens
		return w.getForecastText(t, forecast, unit)
	}
}

// getForecastText handles {day:+N:temp} and {hour:+N:temp} tokens
func (w *WeatherWidget) getForecastText(t *Token, forecast *ForecastData, unit string) string {
	if forecast == nil {
		return "-"
	}

	// Parse token name: day:+1:temp or hour:+3:temp
	parts := strings.Split(t.Name, ":")
	if len(parts) != 3 {
		return "-"
	}

	timeType := parts[0]  // "day" or "hour"
	offsetStr := parts[1] // "+1", "+2", etc.
	field := parts[2]     // "temp", "name", etc.

	var offset int
	fmt.Sscanf(offsetStr, "+%d", &offset)

	var point *ForecastPoint
	if timeType == "day" && offset > 0 && offset <= len(forecast.Daily) {
		point = &forecast.Daily[offset-1]
	} else if timeType == "hour" && offset > 0 {
		// Find the forecast point closest to offset hours from now
		targetTime := time.Now().Add(time.Duration(offset) * time.Hour)
		for i := range forecast.Hourly {
			if forecast.Hourly[i].Time.After(targetTime.Add(-90 * time.Minute)) {
				point = &forecast.Hourly[i]
				break
			}
		}
	}

	if point == nil {
		return "-"
	}

	switch field {
	case "temp":
		return fmt.Sprintf("%.0f%s", point.Temperature, unit)
	case "name":
		return point.Time.Format("Mon")
	case "time":
		return point.Time.Format("15:04")
	case "condition":
		return getWeatherDescription(point.Condition)
	}

	return "-"
}

// getForecastIconName handles {day:+N:icon} and {hour:+N:icon} tokens
func (w *WeatherWidget) getForecastIconName(t *Token, forecast *ForecastData) string {
	if forecast == nil {
		return "sun"
	}

	// Parse: day:+1:icon or hour:+3:icon
	name := t.Name
	if strings.HasPrefix(name, "day:") {
		parts := strings.Split(name, ":")
		if len(parts) >= 2 {
			var offset int
			fmt.Sscanf(parts[1], "+%d", &offset)
			if offset > 0 && offset <= len(forecast.Daily) {
				return w.getIconName(forecast.Daily[offset-1].Condition)
			}
		}
	} else if strings.HasPrefix(name, "hour:") {
		parts := strings.Split(name, ":")
		if len(parts) >= 2 {
			var offset int
			fmt.Sscanf(parts[1], "+%d", &offset)
			targetTime := time.Now().Add(time.Duration(offset) * time.Hour)
			for _, p := range forecast.Hourly {
				if p.Time.After(targetTime.Add(-90 * time.Minute)) {
					return w.getIconName(p.Condition)
				}
			}
		}
	}

	return "sun"
}

// renderLargeTokenInRect renders a large token within a rectangle
func (w *WeatherWidget) renderLargeTokenInRect(img *image.Gray, t *Token, x, y, width, height int, weather *WeatherData, forecast *ForecastData, scrollOffset float64) {
	if width < 10 || height < 5 {
		return
	}

	// Get sub-image for the token area
	bounds := img.Bounds()
	if x < bounds.Min.X {
		x = bounds.Min.X
	}
	if x+width > bounds.Max.X {
		width = bounds.Max.X - x
	}

	switch t.Param {
	case "graph":
		w.renderForecastGraph(img, x, y, width, height, weather, forecast)
	case "icons":
		w.renderForecastIcons(img, x, y, width, height, forecast)
	case "scroll":
		w.renderForecastScroll(img, x, y, width, height, weather, forecast, scrollOffset)
	default:
		// Default to icons if no parameter
		w.renderForecastIcons(img, x, y, width, height, forecast)
	}
}

// renderForecastGraph renders a temperature trend line graph
func (w *WeatherWidget) renderForecastGraph(img *image.Gray, x, y, width, height int, weather *WeatherData, forecast *ForecastData) {
	if forecast == nil || len(forecast.Hourly) == 0 {
		return
	}

	// Find min/max temperatures for scaling
	minTemp := weather.Temperature
	maxTemp := weather.Temperature
	for _, pt := range forecast.Hourly {
		if pt.Temperature < minTemp {
			minTemp = pt.Temperature
		}
		if pt.Temperature > maxTemp {
			maxTemp = pt.Temperature
		}
	}

	// Add padding to range
	tempRange := maxTemp - minTemp
	if tempRange < 2 {
		tempRange = 2
		minTemp -= 1
		maxTemp += 1
	}

	// Draw the graph line
	points := len(forecast.Hourly)
	if points > 1 {
		prevX := 0
		prevY := 0
		for i, pt := range forecast.Hourly {
			px := x + (i * width / (points - 1))
			normalizedTemp := (pt.Temperature - minTemp) / tempRange
			py := y + height - 1 - int(normalizedTemp*float64(height-1))

			if i > 0 {
				bitmap.DrawLine(img, prevX, prevY, px, py, color.Gray{Y: 255})
			}

			prevX = px
			prevY = py
		}
	}
}

// renderForecastIcons renders multi-day forecast with icons
func (w *WeatherWidget) renderForecastIcons(img *image.Gray, x, y, width, height int, forecast *ForecastData) {
	if forecast == nil || len(forecast.Daily) == 0 {
		return
	}

	daysToShow := len(forecast.Daily)
	if daysToShow > w.forecastDays {
		daysToShow = w.forecastDays
	}

	iconSize := 16
	if w.iconSize >= 24 && height >= 30 {
		iconSize = 24
	}

	var iconSet *glyphs.GlyphSet
	if iconSize >= 24 {
		iconSet = glyphs.WeatherIcons24x24
	} else {
		iconSet = glyphs.WeatherIcons16x16
	}

	dayWidth := width / daysToShow
	if dayWidth < iconSize+4 {
		dayWidth = iconSize + 4
		daysToShow = width / dayWidth
		if daysToShow < 1 {
			daysToShow = 1
		}
	}

	unit := "C"
	if w.units == "imperial" {
		unit = "F"
	}

	// Load small font for temperatures
	smallFontSize := 8
	if height < 30 {
		smallFontSize = 6
	}
	smallFont, err := bitmap.LoadFont(w.fontName, smallFontSize)
	if err != nil {
		smallFont = w.fontFace
	}

	for i := 0; i < daysToShow && i < len(forecast.Daily); i++ {
		day := forecast.Daily[i]
		startX := x + i*dayWidth

		iconName := w.getIconName(day.Condition)
		icon := glyphs.GetIcon(iconSet, iconName)

		if icon != nil {
			iconX := startX + (dayWidth-icon.Width)/2
			iconY := y + 1
			glyphs.DrawGlyph(img, icon, iconX, iconY, color.Gray{Y: 255})

			tempStr := fmt.Sprintf("%.0f%s", day.Temperature, unit)
			tempY := iconY + icon.Height + 1
			if tempY < y+height-smallFontSize {
				bitmap.SmartDrawTextInRect(img, tempStr, smallFont, w.fontName, startX, tempY, dayWidth, height-tempY, "center", "top", 0)
			}
		}
	}
}

// renderForecastScroll renders scrolling forecast text
func (w *WeatherWidget) renderForecastScroll(img *image.Gray, x, y, width, height int, weather *WeatherData, forecast *ForecastData, scrollOffset float64) {
	unit := "C"
	if w.units == "imperial" {
		unit = "F"
	}

	// Build scrolling text
	text := fmt.Sprintf("Now: %.0f%s %s", weather.Temperature, unit, weather.Description)

	if forecast != nil {
		// Add hourly highlights
		for i := 0; i < len(forecast.Hourly) && i < 8; i += 3 {
			pt := forecast.Hourly[i]
			text += fmt.Sprintf(" | %s: %.0f%s", pt.Time.Format("15:04"), pt.Temperature, unit)
		}

		// Add daily forecast
		for _, day := range forecast.Daily {
			text += fmt.Sprintf(" | %s: %.0f%s %s", day.Time.Format("Mon"), day.Temperature, unit, getWeatherDescription(day.Condition))
		}
	}

	text += "    ***    "

	textWidth, _ := bitmap.SmartMeasureText(text, w.fontFace, w.fontName)
	offset := int(scrollOffset) % (textWidth + width)

	drawX := x + width - offset
	bitmap.SmartDrawTextInRect(img, text, w.fontFace, w.fontName, drawX, y, textWidth+width, height, "left", "center", 0)

	if drawX+textWidth < x+width {
		bitmap.SmartDrawTextInRect(img, text, w.fontFace, w.fontName, drawX+textWidth, y, textWidth, height, "left", "center", 0)
	}
}

// getIconName maps weather condition to icon name
func (w *WeatherWidget) getIconName(condition string) string {
	switch condition {
	case WeatherClear:
		return "sun"
	case WeatherPartlyCloudy:
		return "partly_cloudy"
	case WeatherCloudy:
		return "cloud"
	case WeatherRain:
		return "rain"
	case WeatherDrizzle:
		return "drizzle"
	case WeatherSnow:
		return "snow"
	case WeatherStorm:
		return "storm"
	case WeatherFog:
		return "fog"
	default:
		return "sun"
	}
}

// getAQIIconName returns icon name for AQI level
func (w *WeatherWidget) getAQIIconName(aqi *AirQualityData) string {
	if aqi == nil {
		return "aqi_unknown"
	}
	// Map AQI to icon (we'll use simple indicators)
	switch aqi.Level {
	case AQIGood:
		return "aqi_good"
	case AQIModerate:
		return "aqi_moderate"
	default:
		return "aqi_bad"
	}
}

// getUVIconName returns icon name for UV level
func (w *WeatherWidget) getUVIconName(uv *UVIndexData) string {
	if uv == nil {
		return "uv_unknown"
	}
	switch uv.Level {
	case UVLow:
		return "uv_low"
	case UVModerate:
		return "uv_moderate"
	default:
		return "uv_high"
	}
}

// getHumidityIconName returns icon name for humidity level
func (w *WeatherWidget) getHumidityIconName(humidity int) string {
	switch {
	case humidity < 30:
		return "humidity_low"
	case humidity < 60:
		return "humidity_moderate"
	default:
		return "humidity_high"
	}
}

// getWindIconName returns icon name for wind speed level
func (w *WeatherWidget) getWindIconName(windSpeed float64) string {
	// Wind speed thresholds in m/s (convert if imperial)
	speed := windSpeed
	if w.units == "imperial" {
		speed = windSpeed * 0.44704 // Convert mph to m/s
	}

	switch {
	case speed < 1:
		return "wind_calm"
	case speed < 5:
		return "wind_light"
	case speed < 10:
		return "wind_moderate"
	default:
		return "wind_strong"
	}
}

// getWindDirIconName returns arrow icon name for wind direction
func (w *WeatherWidget) getWindDirIconName(direction string) string {
	switch strings.ToUpper(direction) {
	case "N":
		return "arrow_n"
	case "NE":
		return "arrow_ne"
	case "E":
		return "arrow_e"
	case "SE":
		return "arrow_se"
	case "S":
		return "arrow_s"
	case "SW":
		return "arrow_sw"
	case "W":
		return "arrow_w"
	case "NW":
		return "arrow_nw"
	default:
		return "arrow_n" // Default to north if unknown
	}
}

// getWeatherDescription returns a human-readable description for a condition
func getWeatherDescription(condition string) string {
	switch condition {
	case WeatherClear:
		return "Clear"
	case WeatherPartlyCloudy:
		return "Partly cloudy"
	case WeatherCloudy:
		return "Cloudy"
	case WeatherRain:
		return "Rain"
	case WeatherDrizzle:
		return "Drizzle"
	case WeatherSnow:
		return "Snow"
	case WeatherStorm:
		return "Storm"
	case WeatherFog:
		return "Fog"
	default:
		return "Unknown"
	}
}

// API fetching functions

// fetchOpenWeatherMap fetches weather data from OpenWeatherMap API
func (w *WeatherWidget) fetchOpenWeatherMap(needForecast bool) (*WeatherData, *ForecastData, error) {
	baseURL := "https://api.openweathermap.org/data/2.5/weather"
	params := url.Values{}
	params.Set("appid", w.apiKey)
	params.Set("units", w.units)

	if w.city != "" {
		params.Set("q", w.city)
	} else {
		params.Set("lat", fmt.Sprintf("%f", w.lat))
		params.Set("lon", fmt.Sprintf("%f", w.lon))
	}

	resp, err := w.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Main struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			Humidity  int     `json:"humidity"`
			Pressure  float64 `json:"pressure"`
		} `json:"main"`
		Weather []struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
		} `json:"weather"`
		Wind struct {
			Speed float64 `json:"speed"`
			Deg   float64 `json:"deg"`
		} `json:"wind"`
		Visibility int `json:"visibility"`
		Sys        struct {
			Sunrise int64 `json:"sunrise"`
			Sunset  int64 `json:"sunset"`
		} `json:"sys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	condition := WeatherClear
	description := ""
	if len(result.Weather) > 0 {
		condition = mapOpenWeatherMapCondition(result.Weather[0].ID)
		description = result.Weather[0].Description
	}

	weatherData := &WeatherData{
		Temperature:   result.Main.Temp,
		FeelsLike:     result.Main.FeelsLike,
		Condition:     condition,
		Description:   description,
		Humidity:      result.Main.Humidity,
		WindSpeed:     result.Wind.Speed,
		WindDirection: degreesToDirection(result.Wind.Deg),
		Pressure:      result.Main.Pressure,
		Visibility:    float64(result.Visibility),
		Sunrise:       time.Unix(result.Sys.Sunrise, 0),
		Sunset:        time.Unix(result.Sys.Sunset, 0),
	}

	var forecastData *ForecastData
	if needForecast {
		forecastData, _ = w.fetchOpenWeatherMapForecast()
	}

	return weatherData, forecastData, nil
}

// fetchOpenWeatherMapForecast fetches forecast from OpenWeatherMap
func (w *WeatherWidget) fetchOpenWeatherMapForecast() (*ForecastData, error) {
	baseURL := "https://api.openweathermap.org/data/2.5/forecast"
	params := url.Values{}
	params.Set("appid", w.apiKey)
	params.Set("units", w.units)

	if w.city != "" {
		params.Set("q", w.city)
	} else {
		params.Set("lat", fmt.Sprintf("%f", w.lat))
		params.Set("lon", fmt.Sprintf("%f", w.lon))
	}

	resp, err := w.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forecast API error: status %d", resp.StatusCode)
	}

	var result struct {
		List []struct {
			Dt   int64 `json:"dt"`
			Main struct {
				Temp float64 `json:"temp"`
			} `json:"main"`
			Weather []struct {
				ID          int    `json:"id"`
				Description string `json:"description"`
			} `json:"weather"`
		} `json:"list"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	forecast := &ForecastData{
		Hourly: make([]ForecastPoint, 0),
		Daily:  make([]ForecastPoint, 0),
	}

	dailyMap := make(map[string]ForecastPoint)
	for i, item := range result.List {
		t := time.Unix(item.Dt, 0)
		condition := WeatherClear
		description := ""
		if len(item.Weather) > 0 {
			condition = mapOpenWeatherMapCondition(item.Weather[0].ID)
			description = item.Weather[0].Description
		}

		point := ForecastPoint{
			Time:        t,
			Temperature: item.Main.Temp,
			Condition:   condition,
			Description: description,
		}

		if i < w.forecastHours/3 {
			forecast.Hourly = append(forecast.Hourly, point)
		}

		dayKey := t.Format("2006-01-02")
		if _, exists := dailyMap[dayKey]; !exists || t.Hour() == 12 {
			dailyMap[dayKey] = point
		}
	}

	// Sort and limit daily forecast
	days := make([]string, 0, len(dailyMap))
	for day := range dailyMap {
		days = append(days, day)
	}
	for i := 0; i < len(days)-1; i++ {
		for j := i + 1; j < len(days); j++ {
			if days[i] > days[j] {
				days[i], days[j] = days[j], days[i]
			}
		}
	}
	for i, day := range days {
		if i >= w.forecastDays {
			break
		}
		forecast.Daily = append(forecast.Daily, dailyMap[day])
	}

	return forecast, nil
}

// fetchOpenWeatherMapAQI fetches air quality data from OpenWeatherMap
func (w *WeatherWidget) fetchOpenWeatherMapAQI() (*AirQualityData, error) {
	baseURL := "https://api.openweathermap.org/data/2.5/air_pollution"
	params := url.Values{}
	params.Set("appid", w.apiKey)
	params.Set("lat", fmt.Sprintf("%f", w.lat))
	params.Set("lon", fmt.Sprintf("%f", w.lon))

	resp, err := w.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AQI API error: status %d", resp.StatusCode)
	}

	var result struct {
		List []struct {
			Main struct {
				AQI int `json:"aqi"` // 1-5 scale
			} `json:"main"`
			Components struct {
				PM25 float64 `json:"pm2_5"`
				PM10 float64 `json:"pm10"`
			} `json:"components"`
		} `json:"list"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.List) == 0 {
		return nil, fmt.Errorf("no AQI data available")
	}

	data := result.List[0]
	// Convert EU AQI (1-5) to US AQI approximation
	usAQI := data.Main.AQI * 40 // Rough conversion
	level := getAQILevel(usAQI)

	return &AirQualityData{
		AQI:   usAQI,
		Level: level,
		PM25:  data.Components.PM25,
		PM10:  data.Components.PM10,
	}, nil
}

// fetchOpenMeteo fetches all weather data from Open-Meteo API
func (w *WeatherWidget) fetchOpenMeteo(needForecast bool) (*WeatherData, *ForecastData, *AirQualityData, *UVIndexData, error) {
	baseURL := "https://api.open-meteo.com/v1/forecast"
	params := url.Values{}
	params.Set("latitude", fmt.Sprintf("%f", w.lat))
	params.Set("longitude", fmt.Sprintf("%f", w.lon))
	params.Set("current", "temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m,wind_direction_10m,surface_pressure,visibility")
	params.Set("daily", "sunrise,sunset")
	params.Set("timezone", "auto")

	if needForecast {
		params.Set("hourly", "temperature_2m,weather_code")
		params.Add("daily", "temperature_2m_max,weather_code")
		params.Set("forecast_days", fmt.Sprintf("%d", w.forecastDays+1))
	}

	if w.units == "imperial" {
		params.Set("temperature_unit", "fahrenheit")
		params.Set("wind_speed_unit", "mph")
	}

	resp, err := w.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, nil, nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Current struct {
			Temperature      float64 `json:"temperature_2m"`
			RelativeHumidity int     `json:"relative_humidity_2m"`
			WeatherCode      int     `json:"weather_code"`
			WindSpeed        float64 `json:"wind_speed_10m"`
			WindDirection    float64 `json:"wind_direction_10m"`
			Pressure         float64 `json:"surface_pressure"`
			Visibility       float64 `json:"visibility"`
		} `json:"current"`
		Hourly struct {
			Time        []string  `json:"time"`
			Temperature []float64 `json:"temperature_2m"`
			WeatherCode []int     `json:"weather_code"`
		} `json:"hourly"`
		Daily struct {
			Time        []string  `json:"time"`
			TempMax     []float64 `json:"temperature_2m_max"`
			WeatherCode []int     `json:"weather_code"`
			Sunrise     []string  `json:"sunrise"`
			Sunset      []string  `json:"sunset"`
		} `json:"daily"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	condition := mapOpenMeteoWeatherCode(result.Current.WeatherCode)

	// Parse sunrise/sunset
	var sunrise, sunset time.Time
	if len(result.Daily.Sunrise) > 0 {
		sunrise, _ = time.Parse("2006-01-02T15:04", result.Daily.Sunrise[0])
	}
	if len(result.Daily.Sunset) > 0 {
		sunset, _ = time.Parse("2006-01-02T15:04", result.Daily.Sunset[0])
	}

	weatherData := &WeatherData{
		Temperature:   result.Current.Temperature,
		FeelsLike:     result.Current.Temperature, // Open-Meteo doesn't provide feels_like in free tier
		Condition:     condition,
		Description:   getWeatherDescription(condition),
		Humidity:      result.Current.RelativeHumidity,
		WindSpeed:     result.Current.WindSpeed,
		WindDirection: degreesToDirection(result.Current.WindDirection),
		Pressure:      result.Current.Pressure,
		Visibility:    result.Current.Visibility,
		Sunrise:       sunrise,
		Sunset:        sunset,
	}

	var forecastData *ForecastData
	if needForecast {
		forecastData = &ForecastData{
			Hourly: make([]ForecastPoint, 0),
			Daily:  make([]ForecastPoint, 0),
		}

		now := time.Now()
		for i := 0; i < len(result.Hourly.Time) && len(forecastData.Hourly) < w.forecastHours; i++ {
			t, err := time.Parse("2006-01-02T15:04", result.Hourly.Time[i])
			if err != nil || t.Before(now) {
				continue
			}
			cond := mapOpenMeteoWeatherCode(result.Hourly.WeatherCode[i])
			forecastData.Hourly = append(forecastData.Hourly, ForecastPoint{
				Time:        t,
				Temperature: result.Hourly.Temperature[i],
				Condition:   cond,
				Description: getWeatherDescription(cond),
			})
		}

		for i := 0; i < len(result.Daily.Time) && i < w.forecastDays; i++ {
			t, err := time.Parse("2006-01-02", result.Daily.Time[i])
			if err != nil {
				continue
			}
			cond := mapOpenMeteoWeatherCode(result.Daily.WeatherCode[i])
			forecastData.Daily = append(forecastData.Daily, ForecastPoint{
				Time:        t,
				Temperature: result.Daily.TempMax[i],
				Condition:   cond,
				Description: getWeatherDescription(cond),
			})
		}
	}

	// Fetch AQI if enabled
	var aqiData *AirQualityData
	if w.aqiEnabled {
		aqiData, _ = w.fetchOpenMeteoAQI()
	}

	// Fetch UV if enabled
	var uvData *UVIndexData
	if w.uvEnabled {
		uvData, _ = w.fetchOpenMeteoUV()
	}

	return weatherData, forecastData, aqiData, uvData, nil
}

// fetchOpenMeteoAQI fetches air quality from Open-Meteo
func (w *WeatherWidget) fetchOpenMeteoAQI() (*AirQualityData, error) {
	baseURL := "https://air-quality-api.open-meteo.com/v1/air-quality"
	params := url.Values{}
	params.Set("latitude", fmt.Sprintf("%f", w.lat))
	params.Set("longitude", fmt.Sprintf("%f", w.lon))
	params.Set("current", "us_aqi,pm2_5,pm10")

	resp, err := w.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AQI API error: status %d", resp.StatusCode)
	}

	var result struct {
		Current struct {
			USAQI int     `json:"us_aqi"`
			PM25  float64 `json:"pm2_5"`
			PM10  float64 `json:"pm10"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &AirQualityData{
		AQI:   result.Current.USAQI,
		Level: getAQILevel(result.Current.USAQI),
		PM25:  result.Current.PM25,
		PM10:  result.Current.PM10,
	}, nil
}

// fetchOpenMeteoUV fetches UV index from Open-Meteo
func (w *WeatherWidget) fetchOpenMeteoUV() (*UVIndexData, error) {
	baseURL := "https://api.open-meteo.com/v1/forecast"
	params := url.Values{}
	params.Set("latitude", fmt.Sprintf("%f", w.lat))
	params.Set("longitude", fmt.Sprintf("%f", w.lon))
	params.Set("daily", "uv_index_max")
	params.Set("forecast_days", "1")
	params.Set("timezone", "auto")

	resp, err := w.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("UV API error: status %d", resp.StatusCode)
	}

	var result struct {
		Daily struct {
			UVIndexMax []float64 `json:"uv_index_max"`
		} `json:"daily"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Daily.UVIndexMax) == 0 {
		return nil, fmt.Errorf("no UV data available")
	}

	uvIndex := result.Daily.UVIndexMax[0]
	return &UVIndexData{
		Index: uvIndex,
		Level: getUVLevel(uvIndex),
	}, nil
}

// Helper functions

// mapOpenWeatherMapCondition maps OpenWeatherMap weather ID to condition
func mapOpenWeatherMapCondition(id int) string {
	switch {
	case id >= 200 && id < 300:
		return WeatherStorm
	case id >= 300 && id < 400:
		return WeatherDrizzle
	case id >= 500 && id < 600:
		return WeatherRain
	case id >= 600 && id < 700:
		return WeatherSnow
	case id >= 700 && id < 800:
		return WeatherFog
	case id == 800:
		return WeatherClear
	case id == 801:
		return WeatherPartlyCloudy
	case id >= 802:
		return WeatherCloudy
	default:
		return WeatherClear
	}
}

// mapOpenMeteoWeatherCode maps WMO weather code to condition
func mapOpenMeteoWeatherCode(code int) string {
	switch {
	case code == 0:
		return WeatherClear
	case code == 1 || code == 2:
		return WeatherPartlyCloudy
	case code == 3:
		return WeatherCloudy
	case code >= 45 && code <= 48:
		return WeatherFog
	case code >= 51 && code <= 57:
		return WeatherDrizzle
	case code >= 61 && code <= 67:
		return WeatherRain
	case code >= 71 && code <= 77:
		return WeatherSnow
	case code >= 80 && code <= 82:
		return WeatherRain
	case code >= 85 && code <= 86:
		return WeatherSnow
	case code >= 95 && code <= 99:
		return WeatherStorm
	default:
		return WeatherClear
	}
}

// degreesToDirection converts wind degrees to cardinal direction
func degreesToDirection(deg float64) string {
	directions := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	index := int(math.Round(deg/45.0)) % 8
	return directions[index]
}

// getAQILevel returns AQI level description
func getAQILevel(aqi int) string {
	switch {
	case aqi <= 50:
		return AQIGood
	case aqi <= 100:
		return AQIModerate
	case aqi <= 150:
		return AQIUnhealthySensitive
	case aqi <= 200:
		return AQIUnhealthy
	case aqi <= 300:
		return AQIVeryUnhealthy
	default:
		return AQIHazardous
	}
}

// getUVLevel returns UV index level description
func getUVLevel(index float64) string {
	switch {
	case index < 3:
		return UVLow
	case index < 6:
		return UVModerate
	case index < 8:
		return UVHigh
	case index < 11:
		return UVVeryHigh
	default:
		return UVExtreme
	}
}
