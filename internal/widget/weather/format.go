package weather

import (
	"fmt"
	"strings"
	"time"

	"github.com/pozitronik/steelclock-go/internal/shared/render"
)

// parseWeatherFormat parses a format string into tokens
func parseWeatherFormat(format string) []render.Token {
	// Handle newlines - split by \n and process each line
	format = strings.ReplaceAll(format, "\\n", "\n")

	return render.ParseFormatTokens(format, getWeatherTokenType)
}

// getWeatherTokenType determines the type of token by name
func getWeatherTokenType(name string) render.TokenType {
	switch name {
	// Icon tokens
	case "icon", "aqi_icon", "uv_icon", "humidity_icon", "wind_icon", "wind_dir_icon":
		return render.TokenIcon
	// Large tokens (expand to fill space)
	case "forecast":
		return TokenLarge
	// Everything else is text
	default:
		// Check for day/hour icon tokens
		if strings.HasPrefix(name, "day") && strings.HasSuffix(name, "icon") {
			return render.TokenIcon
		}
		if strings.HasPrefix(name, "hour") && strings.HasSuffix(name, "icon") {
			return render.TokenIcon
		}
		return render.TokenText
	}
}

// getWeatherTokenText returns the text value for a text token
// units should be "metric" or "imperial"
func getWeatherTokenText(t *render.Token, weather *WData, forecast *ForecastData, aqi *AirQualityData, uv *UVIndexData, units string) string {
	// Guard against nil weather data
	if weather == nil {
		return "-"
	}

	unit := "C"
	speedUnit := "m/s"
	if units == unitsImperial {
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
		if units == unitsImperial {
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
		return getForecastTokenText(t, forecast, unit)
	}
}

// getForecastTokenText handles {day:+N:temp} and {hour:+N:temp} tokens
func getForecastTokenText(t *render.Token, forecast *ForecastData, unit string) string {
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
	_, _ = fmt.Sscanf(offsetStr, "+%d", &offset)

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

// needsWeatherForecast checks if any format needs forecast data
func needsWeatherForecast(formatCycle []string) bool {
	for _, f := range formatCycle {
		if strings.Contains(f, "{forecast") ||
			strings.Contains(f, "{day:") ||
			strings.Contains(f, "{hour:") {
			return true
		}
	}
	return false
}
