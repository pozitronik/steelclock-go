package widget

import (
	"image"
	"testing"

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

func TestWeatherWidget_TextMode(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "weather",
		ID:      "test_weather",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
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

	if widget.displayMode != "text" {
		t.Errorf("displayMode = %s, want text", widget.displayMode)
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
			result := widget.getIconName(tt.condition)
			if result != tt.expected {
				t.Errorf("getIconName(%s) = %s, want %s", tt.condition, result, tt.expected)
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
