package widget

import "math"

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
