package weather

import "time"

// Weather condition codes for icon mapping
const (
	Clear        = "clear"
	PartlyCloudy = "partly_cloudy"
	Cloudy       = "cloud"
	Rain         = "rain"
	Drizzle      = "drizzle"
	Snow         = "snow"
	Storm        = "storm"
	Fog          = "fog"
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
type Data struct {
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
