package widget

import "strings"

// getWeatherIconName maps weather condition to icon name
func getWeatherIconName(condition string) string {
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

// getAQIIcon returns icon name for AQI level
func getAQIIcon(aqi *AirQualityData) string {
	if aqi == nil {
		return "aqi_unknown"
	}
	switch aqi.Level {
	case AQIGood:
		return "aqi_good"
	case AQIModerate:
		return "aqi_moderate"
	default:
		return "aqi_bad"
	}
}

// getUVIcon returns icon name for UV level
func getUVIcon(uv *UVIndexData) string {
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

// getHumidityIcon returns icon name for humidity level
func getHumidityIcon(humidity int) string {
	switch {
	case humidity < 30:
		return "humidity_low"
	case humidity < 60:
		return "humidity_moderate"
	default:
		return "humidity_high"
	}
}

// getWindIcon returns icon name for wind speed level
// windSpeed is expected in m/s for metric or mph for imperial
// units should be "metric" or "imperial"
func getWindIcon(windSpeed float64, units string) string {
	// Wind speed thresholds in m/s (convert if imperial)
	speed := windSpeed
	if units == "imperial" {
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

// getWindDirIcon returns arrow icon name for wind direction
func getWindDirIcon(direction string) string {
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
