package weather

import (
	"math"
	"regexp"
	"strings"
)

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

// abbreviateWeatherError converts error messages to short display-friendly text
// for the small OLED screen (e.g., "HTTP 401", "TIMEOUT", "NO NET")
func abbreviateWeatherError(errMsg string) string {
	// Check for HTTP status codes first
	httpStatusRe := regexp.MustCompile(`(?i)(?:status[:\s]*|HTTP[:\s]*)(\d{3})`)
	if match := httpStatusRe.FindStringSubmatch(errMsg); len(match) > 1 {
		return "HTTP " + match[1]
	}

	// Check for common error patterns
	lowerErr := strings.ToLower(errMsg)

	switch {
	case strings.Contains(lowerErr, "timeout"):
		return "TIMEOUT"
	case strings.Contains(lowerErr, "no such host"):
		return "NO HOST"
	case strings.Contains(lowerErr, "connection refused"):
		return "CONN ERR"
	case strings.Contains(lowerErr, "network"):
		return "NET ERR"
	case strings.Contains(lowerErr, "dns"):
		return "DNS ERR"
	case strings.Contains(lowerErr, "certificate"):
		return "CERT ERR"
	case strings.Contains(lowerErr, "unauthorized") || strings.Contains(lowerErr, "401"):
		return "HTTP 401"
	case strings.Contains(lowerErr, "forbidden") || strings.Contains(lowerErr, "403"):
		return "HTTP 403"
	case strings.Contains(lowerErr, "not found") || strings.Contains(lowerErr, "404"):
		return "HTTP 404"
	case strings.Contains(lowerErr, "rate limit") || strings.Contains(lowerErr, "429"):
		return "RATE LIM"
	case strings.Contains(lowerErr, "server error") || strings.Contains(lowerErr, "500"):
		return "HTTP 500"
	case strings.Contains(lowerErr, "bad gateway") || strings.Contains(lowerErr, "502"):
		return "HTTP 502"
	case strings.Contains(lowerErr, "unavailable") || strings.Contains(lowerErr, "503"):
		return "HTTP 503"
	case strings.Contains(lowerErr, "json") || strings.Contains(lowerErr, "unmarshal"):
		return "BAD DATA"
	case strings.Contains(lowerErr, "eof"):
		return "NO RESP"
	default:
		// Fallback: truncate to fit screen
		if len(errMsg) > 12 {
			return errMsg[:12]
		}
		return errMsg
	}
}
