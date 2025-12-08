package widget

import (
	"image"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewWeatherWidget(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.WidgetConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid open-meteo config",
			cfg: config.WidgetConfig{
				Type:    "weather",
				ID:      "test_weather",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Weather: &config.WeatherConfig{
					Provider: "open-meteo",
					Location: &config.WeatherLocationConfig{
						Lat: 51.5074,
						Lon: -0.1278,
					},
					Units: "metric",
				},
			},
			wantErr: false,
		},
		{
			name: "valid openweathermap config with city",
			cfg: config.WidgetConfig{
				Type:    "weather",
				ID:      "test_weather",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Weather: &config.WeatherConfig{
					Provider: "openweathermap",
					ApiKey:   "test_api_key",
					Location: &config.WeatherLocationConfig{
						City: "London",
					},
					Units: "metric",
				},
			},
			wantErr: false,
		},
		{
			name: "openweathermap without api key",
			cfg: config.WidgetConfig{
				Type:    "weather",
				ID:      "test_weather",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Weather: &config.WeatherConfig{
					Provider: "openweathermap",
					Location: &config.WeatherLocationConfig{
						City: "London",
					},
				},
			},
			wantErr: true,
			errMsg:  "api_key is required",
		},
		{
			name: "open-meteo with city only (no coords)",
			cfg: config.WidgetConfig{
				Type:    "weather",
				ID:      "test_weather",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Weather: &config.WeatherConfig{
					Provider: "open-meteo",
					Location: &config.WeatherLocationConfig{
						City: "London",
					},
				},
			},
			wantErr: true,
			errMsg:  "lat/lon coordinates",
		},
		{
			name: "no location specified",
			cfg: config.WidgetConfig{
				Type:    "weather",
				ID:      "test_weather",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Weather: &config.WeatherConfig{
					Provider: "open-meteo",
				},
			},
			wantErr: true,
			errMsg:  "location is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widget, err := NewWeatherWidget(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewWeatherWidget() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("NewWeatherWidget() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewWeatherWidget() unexpected error: %v", err)
			}
			if widget == nil {
				t.Fatal("NewWeatherWidget() returned nil")
			}
			if widget.Name() != tt.cfg.ID {
				t.Errorf("Name() = %s, want %s", widget.Name(), tt.cfg.ID)
			}
		})
	}
}

func TestWeatherWidget_Render_NoData(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "weather",
		ID:      "test_weather",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Weather: &config.WeatherConfig{
			Provider: "open-meteo",
			Location: &config.WeatherLocationConfig{
				Lat: 51.5074,
				Lon: -0.1278,
			},
		},
	}

	widget, err := NewWeatherWidget(cfg)
	if err != nil {
		t.Fatalf("NewWeatherWidget() error = %v", err)
	}

	// Render without calling Update first - should show "..."
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}

	// Check image dimensions
	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Render() image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

func TestWeatherWidget_Render_WithData(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "weather",
		ID:      "test_weather",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Weather: &config.WeatherConfig{
			Provider: "open-meteo",
			Location: &config.WeatherLocationConfig{
				Lat: 51.5074,
				Lon: -0.1278,
			},
			Units: "metric",
		},
	}

	widget, err := NewWeatherWidget(cfg)
	if err != nil {
		t.Fatalf("NewWeatherWidget() error = %v", err)
	}

	// Manually set weather data to avoid network call
	widget.mu.Lock()
	widget.weather = &WeatherData{
		Temperature: 15.5,
		FeelsLike:   14.0,
		Condition:   WeatherClear,
		Description: "Clear sky",
		Humidity:    65,
		WindSpeed:   5.2,
	}
	widget.mu.Unlock()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}

	// Verify it's a proper image with expected dimensions
	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Render() image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}

	// Verify the image is a grayscale image
	if _, ok := img.(*image.Gray); !ok {
		t.Error("Render() did not return a grayscale image")
	}
}

func TestWeatherWidget_TextFormat(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "weather",
		ID:      "test_weather",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Weather: &config.WeatherConfig{
			Provider: "open-meteo",
			Location: &config.WeatherLocationConfig{
				Lat: 51.5074,
				Lon: -0.1278,
			},
			Units:  "metric",
			Format: config.StringOrSlice{"{temp} {description}"},
		},
	}

	widget, err := NewWeatherWidget(cfg)
	if err != nil {
		t.Fatalf("NewWeatherWidget() error = %v", err)
	}

	if widget.formatCycle[0] != "{temp} {description}" {
		t.Errorf("format = %s, want {temp} {description}", widget.formatCycle[0])
	}

	// Set weather data
	widget.mu.Lock()
	widget.weather = &WeatherData{
		Temperature: 20.0,
		Condition:   WeatherRain,
		Description: "Light rain",
	}
	widget.mu.Unlock()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestWeatherWidget_ImperialUnits(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "weather",
		ID:      "test_weather",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Weather: &config.WeatherConfig{
			Provider: "open-meteo",
			Location: &config.WeatherLocationConfig{
				Lat: 40.7128,
				Lon: -74.0060,
			},
			Units: "imperial",
		},
	}

	widget, err := NewWeatherWidget(cfg)
	if err != nil {
		t.Fatalf("NewWeatherWidget() error = %v", err)
	}

	if widget.units != "imperial" {
		t.Errorf("units = %s, want imperial", widget.units)
	}
}

func TestMapOpenWeatherMapCondition(t *testing.T) {
	tests := []struct {
		id       int
		expected string
	}{
		{200, WeatherStorm},   // Thunderstorm
		{300, WeatherDrizzle}, // Drizzle
		{500, WeatherRain},    // Rain
		{600, WeatherSnow},    // Snow
		{700, WeatherFog},     // Fog
		{800, WeatherClear},   // Clear
		{801, WeatherPartlyCloudy},
		{802, WeatherCloudy},
		{803, WeatherCloudy},
		{999, WeatherCloudy}, // Falls into >= 802 range
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := mapOpenWeatherMapCondition(tt.id)
			if result != tt.expected {
				t.Errorf("mapOpenWeatherMapCondition(%d) = %s, want %s", tt.id, result, tt.expected)
			}
		})
	}
}

func TestMapOpenMeteoWeatherCode(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{0, WeatherClear},
		{1, WeatherPartlyCloudy},
		{2, WeatherPartlyCloudy},
		{3, WeatherCloudy},
		{45, WeatherFog},
		{51, WeatherDrizzle},
		{61, WeatherRain},
		{71, WeatherSnow},
		{95, WeatherStorm},
		{999, WeatherClear}, // Unknown defaults to clear
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := mapOpenMeteoWeatherCode(tt.code)
			if result != tt.expected {
				t.Errorf("mapOpenMeteoWeatherCode(%d) = %s, want %s", tt.code, result, tt.expected)
			}
		})
	}
}

func TestGetWeatherDescription(t *testing.T) {
	tests := []struct {
		condition string
		expected  string
	}{
		{WeatherClear, "Clear"},
		{WeatherPartlyCloudy, "Partly cloudy"},
		{WeatherCloudy, "Cloudy"},
		{WeatherRain, "Rain"},
		{WeatherDrizzle, "Drizzle"},
		{WeatherSnow, "Snow"},
		{WeatherStorm, "Storm"},
		{WeatherFog, "Fog"},
		{"unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.condition, func(t *testing.T) {
			result := getWeatherDescription(tt.condition)
			if result != tt.expected {
				t.Errorf("getWeatherDescription(%s) = %s, want %s", tt.condition, result, tt.expected)
			}
		})
	}
}

func TestWeatherWidget_GetIconName(t *testing.T) {
	// getWeatherIconName is now a standalone function
	tests := []struct {
		condition string
		expected  string
	}{
		{WeatherClear, "sun"},
		{WeatherPartlyCloudy, "partly_cloudy"},
		{WeatherCloudy, "cloud"},
		{WeatherRain, "rain"},
		{WeatherDrizzle, "drizzle"},
		{WeatherSnow, "snow"},
		{WeatherStorm, "storm"},
		{WeatherFog, "fog"},
		{"unknown", "sun"}, // Default to sun
	}

	for _, tt := range tests {
		t.Run(tt.condition, func(t *testing.T) {
			result := getWeatherIconName(tt.condition)
			if result != tt.expected {
				t.Errorf("getIconName(%s) = %s, want %s", tt.condition, result, tt.expected)
			}
		})
	}
}

func TestWeatherWidget_ForecastGraphFormat(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "weather",
		ID:      "test_weather",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Weather: &config.WeatherConfig{
			Provider: "open-meteo",
			Location: &config.WeatherLocationConfig{
				Lat: 51.5074,
				Lon: -0.1278,
			},
			Format: config.StringOrSlice{"{temp} {forecast:graph}"},
			Forecast: &config.WeatherForecastConfig{
				Hours: 12,
			},
		},
	}

	widget, err := NewWeatherWidget(cfg)
	if err != nil {
		t.Fatalf("NewWeatherWidget() error = %v", err)
	}

	if widget.forecastHours != 12 {
		t.Errorf("forecastHours = %d, want 12", widget.forecastHours)
	}

	// Set weather and forecast data
	widget.mu.Lock()
	widget.weather = &WeatherData{
		Temperature: 15.0,
		Condition:   WeatherClear,
		Description: "Clear",
	}
	widget.forecast = &ForecastData{
		Hourly: []ForecastPoint{
			{Temperature: 15.0, Condition: WeatherClear},
			{Temperature: 16.0, Condition: WeatherClear},
			{Temperature: 18.0, Condition: WeatherPartlyCloudy},
			{Temperature: 17.0, Condition: WeatherCloudy},
			{Temperature: 14.0, Condition: WeatherRain},
		},
	}
	widget.mu.Unlock()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Render() image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

func TestWeatherWidget_ForecastIconsFormat(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "weather",
		ID:      "test_weather",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Weather: &config.WeatherConfig{
			Provider: "open-meteo",
			Location: &config.WeatherLocationConfig{
				Lat: 51.5074,
				Lon: -0.1278,
			},
			Format: config.StringOrSlice{"{forecast:icons}"},
			Forecast: &config.WeatherForecastConfig{
				Days: 3,
			},
		},
	}

	widget, err := NewWeatherWidget(cfg)
	if err != nil {
		t.Fatalf("NewWeatherWidget() error = %v", err)
	}

	if widget.forecastDays != 3 {
		t.Errorf("forecastDays = %d, want 3", widget.forecastDays)
	}

	// Set weather and daily forecast data
	widget.mu.Lock()
	widget.weather = &WeatherData{
		Temperature: 15.0,
		Condition:   WeatherClear,
		Description: "Clear",
	}
	widget.forecast = &ForecastData{
		Daily: []ForecastPoint{
			{Time: time.Now(), Temperature: 15.0, Condition: WeatherClear},
			{Time: time.Now().Add(24 * time.Hour), Temperature: 18.0, Condition: WeatherPartlyCloudy},
			{Time: time.Now().Add(48 * time.Hour), Temperature: 12.0, Condition: WeatherRain},
		},
	}
	widget.mu.Unlock()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Render() image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

func TestWeatherWidget_ScrollFormat(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "weather",
		ID:      "test_weather",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Weather: &config.WeatherConfig{
			Provider: "open-meteo",
			Location: &config.WeatherLocationConfig{
				Lat: 51.5074,
				Lon: -0.1278,
			},
			Format: config.StringOrSlice{"{forecast:scroll}"},
			Forecast: &config.WeatherForecastConfig{
				ScrollSpeed: 50,
			},
		},
	}

	widget, err := NewWeatherWidget(cfg)
	if err != nil {
		t.Fatalf("NewWeatherWidget() error = %v", err)
	}

	if widget.scrollSpeed != 50 {
		t.Errorf("scrollSpeed = %f, want 50", widget.scrollSpeed)
	}

	// Set weather and forecast data
	widget.mu.Lock()
	widget.weather = &WeatherData{
		Temperature: 15.0,
		Condition:   WeatherClear,
		Description: "Clear sky",
	}
	widget.forecast = &ForecastData{
		Hourly: []ForecastPoint{
			{Time: time.Now().Add(time.Hour), Temperature: 16.0, Condition: WeatherClear},
			{Time: time.Now().Add(2 * time.Hour), Temperature: 18.0, Condition: WeatherPartlyCloudy},
		},
		Daily: []ForecastPoint{
			{Time: time.Now(), Temperature: 15.0, Condition: WeatherClear},
			{Time: time.Now().Add(24 * time.Hour), Temperature: 18.0, Condition: WeatherRain},
		},
	}
	widget.scrollOffset = 100 // Simulate some scroll progress
	widget.mu.Unlock()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Render() image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

func TestWeatherWidget_FormatCycle(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "weather",
		ID:      "test_weather",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Weather: &config.WeatherConfig{
			Provider: "open-meteo",
			Location: &config.WeatherLocationConfig{
				Lat: 51.5074,
				Lon: -0.1278,
			},
			Format: config.StringOrSlice{"{icon} {temp}", "{humidity}", "{wind}"},
			Cycle: &config.WeatherCycleConfig{
				Interval: 5,
			},
		},
	}

	widget, err := NewWeatherWidget(cfg)
	if err != nil {
		t.Fatalf("NewWeatherWidget() error = %v", err)
	}

	if len(widget.formatCycle) != 3 {
		t.Errorf("formatCycle length = %d, want 3", len(widget.formatCycle))
	}

	if widget.cycleInterval != 5 {
		t.Errorf("cycleInterval = %d, want 5", widget.cycleInterval)
	}

	// Set weather data
	widget.mu.Lock()
	widget.weather = &WeatherData{
		Temperature:   15.0,
		Condition:     WeatherClear,
		Description:   "Clear",
		Humidity:      65,
		WindSpeed:     5.2,
		WindDirection: "NW",
	}
	widget.mu.Unlock()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestWeatherWidget_ConfigDefaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "weather",
		ID:      "test_weather",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Weather: &config.WeatherConfig{
			Provider: "open-meteo",
			Location: &config.WeatherLocationConfig{
				Lat: 51.5074,
				Lon: -0.1278,
			},
			// No other settings - should use defaults
		},
	}

	widget, err := NewWeatherWidget(cfg)
	if err != nil {
		t.Fatalf("NewWeatherWidget() error = %v", err)
	}

	// Check defaults
	if widget.forecastHours != 24 {
		t.Errorf("forecastHours = %d, want default 24", widget.forecastHours)
	}
	if widget.forecastDays != 3 {
		t.Errorf("forecastDays = %d, want default 3", widget.forecastDays)
	}
	if widget.scrollSpeed != 30 {
		t.Errorf("scrollSpeed = %f, want default 30", widget.scrollSpeed)
	}
	if widget.iconSize != 16 {
		t.Errorf("iconSize = %d, want default 16", widget.iconSize)
	}
	if widget.formatCycle[0] != "{icon} {temp}" {
		t.Errorf("format = %s, want default {icon} {temp}", widget.formatCycle[0])
	}
	if widget.cycleInterval != 10 {
		t.Errorf("cycleInterval = %d, want default 10", widget.cycleInterval)
	}
}

func TestWeatherWidget_AQIFormat(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "weather",
		ID:      "test_weather",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Weather: &config.WeatherConfig{
			Provider: "open-meteo",
			Location: &config.WeatherLocationConfig{
				Lat: 51.5074,
				Lon: -0.1278,
			},
			Format: config.StringOrSlice{"{icon} {temp} AQI:{aqi}"},
		},
	}

	widget, err := NewWeatherWidget(cfg)
	if err != nil {
		t.Fatalf("NewWeatherWidget() error = %v", err)
	}

	// AQI should be auto-enabled because format contains {aqi}
	if !widget.aqiEnabled {
		t.Error("aqiEnabled should be true when format contains {aqi}")
	}

	// Set weather and AQI data
	widget.mu.Lock()
	widget.weather = &WeatherData{
		Temperature: 15.0,
		Condition:   WeatherClear,
		Description: "Clear",
	}
	widget.airQuality = &AirQualityData{
		AQI:   42,
		Level: AQIGood,
		PM25:  10.5,
		PM10:  20.0,
	}
	widget.mu.Unlock()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestWeatherWidget_MultilineFormat(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "weather",
		ID:      "test_weather",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Weather: &config.WeatherConfig{
			Provider: "open-meteo",
			Location: &config.WeatherLocationConfig{
				Lat: 51.5074,
				Lon: -0.1278,
			},
			Format: config.StringOrSlice{"{icon} {temp}\\n{humidity} {wind}"},
		},
	}

	widget, err := NewWeatherWidget(cfg)
	if err != nil {
		t.Fatalf("NewWeatherWidget() error = %v", err)
	}

	// Set weather data
	widget.mu.Lock()
	widget.weather = &WeatherData{
		Temperature:   15.0,
		Condition:     WeatherClear,
		Description:   "Clear",
		Humidity:      65,
		WindSpeed:     5.2,
		WindDirection: "NW",
	}
	widget.mu.Unlock()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		format       string
		expectTokens int
	}{
		{"{icon} {temp}", 3},              // icon, space, temp
		{"{temp}", 1},                     // just temp
		{"Hello {temp} World", 3},         // literal, temp, literal
		{"{icon:24} {temp}", 3},           // icon with param, space, temp
		{"{forecast:graph}", 1},           // large token
		{"{day:+1:temp}", 1},              // forecast token
		{"{icon} {temp}\\n{humidity}", 5}, // with newline
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			tokens := parseWeatherFormat(tt.format)
			if len(tokens) != tt.expectTokens {
				t.Errorf("parseWeatherFormat(%q) returned %d tokens, want %d", tt.format, len(tokens), tt.expectTokens)
			}
		})
	}
}

func TestGetAQILevel(t *testing.T) {
	tests := []struct {
		aqi      int
		expected string
	}{
		{25, AQIGood},
		{50, AQIGood},
		{75, AQIModerate},
		{100, AQIModerate},
		{125, AQIUnhealthySensitive},
		{175, AQIUnhealthy},
		{250, AQIVeryUnhealthy},
		{350, AQIHazardous},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := getAQILevel(tt.aqi)
			if result != tt.expected {
				t.Errorf("getAQILevel(%d) = %s, want %s", tt.aqi, result, tt.expected)
			}
		})
	}
}

func TestGetUVLevel(t *testing.T) {
	tests := []struct {
		uv       float64
		expected string
	}{
		{1.0, UVLow},
		{2.5, UVLow},
		{4.0, UVModerate},
		{5.5, UVModerate},
		{7.0, UVHigh},
		{9.0, UVVeryHigh},
		{11.5, UVExtreme},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := getUVLevel(tt.uv)
			if result != tt.expected {
				t.Errorf("getUVLevel(%f) = %s, want %s", tt.uv, result, tt.expected)
			}
		})
	}
}

func TestDegreesToDirection(t *testing.T) {
	tests := []struct {
		deg      float64
		expected string
	}{
		{0, "N"},
		{45, "NE"},
		{90, "E"},
		{135, "SE"},
		{180, "S"},
		{225, "SW"},
		{270, "W"},
		{315, "NW"},
		{360, "N"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := degreesToDirection(tt.deg)
			if result != tt.expected {
				t.Errorf("degreesToDirection(%f) = %s, want %s", tt.deg, result, tt.expected)
			}
		})
	}
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAbbreviateWeatherError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		// HTTP status codes
		{"status 401", "status: 401", "HTTP 401"},
		{"HTTP 401", "HTTP: 401 Unauthorized", "HTTP 401"},
		{"status 500", "server returned status: 500", "HTTP 500"},
		{"HTTP 403", "HTTP 403 Forbidden", "HTTP 403"},

		// Common network errors
		{"timeout", "connection timeout exceeded", "TIMEOUT"},
		{"no such host", "dial tcp: no such host", "NO HOST"},
		{"connection refused", "dial tcp: connection refused", "CONN ERR"},
		{"network error", "network is unreachable", "NET ERR"},
		{"dns error", "dns lookup failed", "DNS ERR"},
		{"certificate error", "certificate verify failed", "CERT ERR"},

		// HTTP-specific patterns
		{"unauthorized", "unauthorized access", "HTTP 401"},
		{"forbidden", "forbidden resource", "HTTP 403"},
		{"not found", "resource not found", "HTTP 404"},
		{"rate limit", "rate limit exceeded", "RATE LIM"},
		{"server error", "internal server error", "HTTP 500"},
		{"bad gateway", "bad gateway response", "HTTP 502"},
		{"unavailable", "service unavailable", "HTTP 503"},

		// Data errors
		{"json error", "json: cannot unmarshal", "BAD DATA"},
		{"unmarshal error", "unmarshal failed", "BAD DATA"},
		{"eof error", "unexpected EOF", "NO RESP"},

		// Fallback truncation
		{"short message", "ERROR", "ERROR"},
		{"long message", "This is a very long error message", "This is a ve"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := abbreviateWeatherError(tt.errMsg)
			if result != tt.expected {
				t.Errorf("abbreviateWeatherError(%q) = %q, want %q", tt.errMsg, result, tt.expected)
			}
		})
	}
}
