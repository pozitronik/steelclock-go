package widget

import (
	"fmt"
	"image"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
	"golang.org/x/image/font"
)

func init() {
	Register("weather", func(cfg config.WidgetConfig) (Widget, error) {
		return NewWeatherWidget(cfg)
	})
}

// WeatherWidget displays weather information using a format string
type WeatherWidget struct {
	*BaseWidget
	// Configuration
	weatherProvider WeatherProvider
	units           string
	iconSize        int
	formatCycle     []string // Format strings (single or multiple for cycling)
	cycleInterval   int
	forecastHours   int // Used for rendering forecasts
	forecastDays    int // Used for rendering forecasts
	scrollSpeed     float64
	aqiEnabled      bool
	uvEnabled       bool
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
	transition    *shared.TransitionManager
	pendingFormat int // Format index to transition to
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
}

// NewWeatherWidget creates a new weather widget
func NewWeatherWidget(cfg config.WidgetConfig) (*WeatherWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract common settings
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()

	// Weather-specific settings with defaults
	providerName := "open-meteo"
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
			providerName = cfg.Weather.Provider
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
	if providerName == "openweathermap" && apiKey == "" {
		return nil, fmt.Errorf("api_key is required for OpenWeatherMap provider")
	}

	// Location validation
	hasCity := city != ""
	hasCoords := lat != 0 || lon != 0
	if !hasCity && !hasCoords {
		return nil, fmt.Errorf("location is required: specify either city or lat/lon coordinates")
	}

	// Open-Meteo requires coordinates
	if providerName == "open-meteo" && hasCity && !hasCoords {
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

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create provider config
	providerCfg := WeatherProviderConfig{
		City:          city,
		Lat:           lat,
		Lon:           lon,
		Units:         units,
		ForecastHours: forecastHours,
		ForecastDays:  forecastDays,
	}

	// Create the weather provider
	var weatherProvider WeatherProvider
	switch providerName {
	case "openweathermap":
		weatherProvider = NewOpenWeatherMapProvider(providerCfg, apiKey, httpClient)
	case "open-meteo":
		weatherProvider = NewOpenMeteoProvider(providerCfg, httpClient)
	default:
		return nil, fmt.Errorf("unknown weather provider: %s", providerName)
	}

	pos := base.GetPosition()
	w := &WeatherWidget{
		BaseWidget:      base,
		weatherProvider: weatherProvider,
		units:           units,
		iconSize:        iconSize,
		formatCycle:     formatCycle,
		cycleInterval:   cycleInterval,
		forecastHours:   forecastHours,
		forecastDays:    forecastDays,
		transitionType:  transitionType,
		transitionSpeed: transitionSpeed,
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
		transition:      shared.NewTransitionManager(pos.W, pos.H),
	}

	// Parse initial format (first format in cycle)
	w.tokens = parseWeatherFormat(formatCycle[0])

	return w, nil
}

// Update fetches fresh weather data from the API
func (w *WeatherWidget) Update() error {
	// Check if we need forecast data
	needForecast := needsWeatherForecast(w.formatCycle)

	// Fetch weather and forecast from provider
	weather, forecast, err := w.weatherProvider.FetchWeather(needForecast)

	w.mu.Lock()
	defer w.mu.Unlock()

	if err != nil {
		w.lastError = err.Error()
		log.Printf("Weather update error: %v", err)
		return nil // Don't return error to keep widget running
	}

	w.weather = weather
	w.forecast = forecast
	w.lastError = ""

	// Fetch AQI if enabled
	if w.aqiEnabled {
		if aqi, err := w.weatherProvider.FetchAirQuality(); err == nil && aqi != nil {
			w.airQuality = aqi
		}
	}

	// Fetch UV if enabled
	if w.uvEnabled {
		if uv, err := w.weatherProvider.FetchUVIndex(); err == nil && uv != nil {
			w.uvIndex = uv
		}
	}

	return nil
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
	if w.transition.IsActive() {
		if !w.transition.Update() {
			// Transition complete
			w.currentFormat = w.pendingFormat
			w.tokens = parseWeatherFormat(w.formatCycle[w.currentFormat])
			w.scrollOffset = 0
		}
	}

	// Handle format cycling (start new transition when it's time)
	if len(w.formatCycle) > 1 && w.cycleInterval > 0 && !w.transition.IsActive() {
		cycleElapsed := now.Sub(w.lastCycleTime).Seconds()
		if cycleElapsed >= float64(w.cycleInterval) {
			// Capture current frame
			oldFrame := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())
			w.renderTokens(oldFrame, w.tokens, w.weather, w.forecast, w.airQuality, w.uvIndex, w.scrollOffset)

			// Set up transition
			w.pendingFormat = (w.currentFormat + 1) % len(w.formatCycle)
			w.transition.Start(shared.TransitionType(w.transitionType), w.transitionSpeed, oldFrame)
			w.lastCycleTime = now
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
	pendingFormat := w.pendingFormat
	w.mu.RUnlock()

	// Handle error state
	if lastError != "" && weather == nil {
		errMsg := abbreviateWeatherError(lastError)
		bitmap.SmartDrawAlignedText(img, errMsg, w.fontFace, w.fontName, "center", "center", w.padding)
		return img, nil
	}

	// Handle no data yet
	if weather == nil {
		bitmap.SmartDrawAlignedText(img, "...", w.fontFace, w.fontName, "center", "center", w.padding)
		return img, nil
	}

	// If transition is active, render both frames and composite
	// Use IsActiveLive for accurate timing regardless of Update() frequency
	if w.transition.IsActiveLive() && w.transition.OldFrame() != nil {
		// Render new frame
		newFrame := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())
		newTokens := parseWeatherFormat(w.formatCycle[pendingFormat])
		w.renderTokens(newFrame, newTokens, weather, forecast, aqi, uv, 0) // Reset scroll for new format

		// Apply transition with live progress for smooth animation
		w.transition.ApplyLive(img, newFrame)
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
