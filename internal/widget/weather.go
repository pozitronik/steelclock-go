package widget

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"net/http"
	"net/url"
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

// WeatherData holds the current weather information
type WeatherData struct {
	Temperature float64
	Condition   string // One of the Weather* constants
	Description string // Human-readable description
	Humidity    int
	WindSpeed   float64
}

// WeatherWidget displays current weather conditions
type WeatherWidget struct {
	*BaseWidget
	// Configuration
	provider    string // "openweathermap" or "open-meteo"
	apiKey      string
	city        string
	lat         float64
	lon         float64
	units       string // "metric" or "imperial"
	showIcon    bool
	iconSize    int
	displayMode string
	// Display settings
	fontSize   int
	fontName   string
	horizAlign string
	vertAlign  string
	padding    int
	fontFace   font.Face
	// Current weather data
	weather   *WeatherData
	lastError string
	mu        sync.RWMutex
	// HTTP client
	httpClient *http.Client
}

// NewWeatherWidget creates a new weather widget
func NewWeatherWidget(cfg config.WidgetConfig) (*WeatherWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract common settings
	displayMode := helper.GetDisplayMode("icon")
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()

	// Weather-specific settings with defaults
	provider := "open-meteo"
	apiKey := ""
	city := ""
	lat := 0.0
	lon := 0.0
	units := "metric"
	showIcon := true
	iconSize := 16

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
		if cfg.Weather.ShowIcon != nil {
			showIcon = *cfg.Weather.ShowIcon
		}
		if cfg.Weather.IconSize > 0 {
			iconSize = cfg.Weather.IconSize
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

	return &WeatherWidget{
		BaseWidget:  base,
		provider:    provider,
		apiKey:      apiKey,
		city:        city,
		lat:         lat,
		lon:         lon,
		units:       units,
		showIcon:    showIcon,
		iconSize:    iconSize,
		displayMode: displayMode,
		fontSize:    fontSize,
		fontName:    fontName,
		horizAlign:  textSettings.HorizAlign,
		vertAlign:   textSettings.VertAlign,
		padding:     padding,
		fontFace:    fontFace,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// Update fetches fresh weather data from the API
func (w *WeatherWidget) Update() error {
	var data *WeatherData
	var err error

	switch w.provider {
	case "openweathermap":
		data, err = w.fetchOpenWeatherMap()
	case "open-meteo":
		data, err = w.fetchOpenMeteo()
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

	w.weather = data
	w.lastError = ""
	return nil
}

// fetchOpenWeatherMap fetches weather from OpenWeatherMap API
func (w *WeatherWidget) fetchOpenWeatherMap() (*WeatherData, error) {
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
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Main struct {
			Temp     float64 `json:"temp"`
			Humidity int     `json:"humidity"`
		} `json:"main"`
		Weather []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
		} `json:"weather"`
		Wind struct {
			Speed float64 `json:"speed"`
		} `json:"wind"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	condition := WeatherClear
	description := ""
	if len(result.Weather) > 0 {
		condition = mapOpenWeatherMapCondition(result.Weather[0].ID)
		description = result.Weather[0].Description
	}

	return &WeatherData{
		Temperature: result.Main.Temp,
		Condition:   condition,
		Description: description,
		Humidity:    result.Main.Humidity,
		WindSpeed:   result.Wind.Speed,
	}, nil
}

// fetchOpenMeteo fetches weather from Open-Meteo API (free, no API key)
func (w *WeatherWidget) fetchOpenMeteo() (*WeatherData, error) {
	baseURL := "https://api.open-meteo.com/v1/forecast"
	params := url.Values{}
	params.Set("latitude", fmt.Sprintf("%f", w.lat))
	params.Set("longitude", fmt.Sprintf("%f", w.lon))
	params.Set("current", "temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m")

	if w.units == "imperial" {
		params.Set("temperature_unit", "fahrenheit")
		params.Set("wind_speed_unit", "mph")
	}

	resp, err := w.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Current struct {
			Temperature      float64 `json:"temperature_2m"`
			RelativeHumidity int     `json:"relative_humidity_2m"`
			WeatherCode      int     `json:"weather_code"`
			WindSpeed        float64 `json:"wind_speed_10m"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	condition := mapOpenMeteoWeatherCode(result.Current.WeatherCode)

	return &WeatherData{
		Temperature: result.Current.Temperature,
		Condition:   condition,
		Description: getWeatherDescription(condition),
		Humidity:    result.Current.RelativeHumidity,
		WindSpeed:   result.Current.WindSpeed,
	}, nil
}

// mapOpenWeatherMapCondition maps OpenWeatherMap weather ID to our condition constants
// See: https://openweathermap.org/weather-conditions
func mapOpenWeatherMapCondition(id int) string {
	switch {
	case id >= 200 && id < 300:
		return WeatherStorm // Thunderstorm
	case id >= 300 && id < 400:
		return WeatherDrizzle // Drizzle
	case id >= 500 && id < 600:
		return WeatherRain // Rain
	case id >= 600 && id < 700:
		return WeatherSnow // Snow
	case id >= 700 && id < 800:
		return WeatherFog // Atmosphere (mist, fog, etc.)
	case id == 800:
		return WeatherClear // Clear
	case id == 801:
		return WeatherPartlyCloudy // Few clouds
	case id >= 802:
		return WeatherCloudy // Cloudy
	default:
		return WeatherClear
	}
}

// mapOpenMeteoWeatherCode maps WMO weather code to our condition constants
// See: https://open-meteo.com/en/docs (WMO Weather interpretation codes)
func mapOpenMeteoWeatherCode(code int) string {
	switch {
	case code == 0:
		return WeatherClear // Clear sky
	case code == 1 || code == 2:
		return WeatherPartlyCloudy // Mainly clear, partly cloudy
	case code == 3:
		return WeatherCloudy // Overcast
	case code >= 45 && code <= 48:
		return WeatherFog // Fog
	case code >= 51 && code <= 55:
		return WeatherDrizzle // Drizzle
	case code >= 56 && code <= 57:
		return WeatherDrizzle // Freezing drizzle
	case code >= 61 && code <= 65:
		return WeatherRain // Rain
	case code >= 66 && code <= 67:
		return WeatherRain // Freezing rain
	case code >= 71 && code <= 77:
		return WeatherSnow // Snow
	case code >= 80 && code <= 82:
		return WeatherRain // Rain showers
	case code >= 85 && code <= 86:
		return WeatherSnow // Snow showers
	case code >= 95 && code <= 99:
		return WeatherStorm // Thunderstorm
	default:
		return WeatherClear
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

// Render creates the weather widget image
func (w *WeatherWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Draw border if enabled
	style := w.GetStyle()
	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	w.mu.RLock()
	weather := w.weather
	lastError := w.lastError
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

	// Format temperature
	tempUnit := "C"
	if w.units == "imperial" {
		tempUnit = "F"
	}
	tempStr := fmt.Sprintf("%.0f%s", weather.Temperature, tempUnit)

	switch w.displayMode {
	case "text":
		// Text-only mode: "15C Cloudy"
		text := fmt.Sprintf("%s %s", tempStr, weather.Description)
		bitmap.SmartDrawAlignedText(img, text, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
	default:
		// Icon mode (default)
		w.renderWithIcon(img, weather, tempStr)
	}

	return img, nil
}

// renderWithIcon renders the weather with an icon and temperature
func (w *WeatherWidget) renderWithIcon(img *image.Gray, weather *WeatherData, tempStr string) {
	pos := w.GetPosition()

	// Get the appropriate icon set based on size
	var iconSet *glyphs.GlyphSet
	iconName := w.getIconName(weather.Condition)

	if w.iconSize >= 24 {
		iconSet = glyphs.WeatherIcons24x24
	} else {
		iconSet = glyphs.WeatherIcons16x16
	}

	icon := glyphs.GetIcon(iconSet, iconName)
	iconWidth := 0
	iconHeight := 0
	if icon != nil {
		iconWidth = icon.Width
		iconHeight = icon.Height
	}

	// Calculate positions
	textWidth, _ := bitmap.SmartMeasureText(tempStr, w.fontFace, w.fontName)
	totalWidth := iconWidth + 2 + textWidth // 2px gap between icon and text

	startX := (pos.W - totalWidth) / 2
	if startX < w.padding {
		startX = w.padding
	}

	// Draw icon
	if icon != nil {
		iconY := (pos.H - iconHeight) / 2
		glyphs.DrawGlyph(img, icon, startX, iconY, color.Gray{Y: 255})
	}

	// Draw temperature text to the right of the icon
	textX := startX + iconWidth + 2
	textRectW := pos.W - textX - w.padding
	if textRectW < 1 {
		textRectW = 1
	}
	bitmap.SmartDrawTextInRect(img, tempStr, w.fontFace, w.fontName, textX, 0, textRectW, pos.H, "left", "center", 0)
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
