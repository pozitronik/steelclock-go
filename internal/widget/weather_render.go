package widget

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
)

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
		text := getWeatherTokenText(t, weather, forecast, aqi, uv, w.units)
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
		_, _ = fmt.Sscanf(t.Param, "%d", &size)
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
		text := getWeatherTokenText(t, weather, forecast, aqi, uv, w.units)
		width, _ := bitmap.SmartMeasureText(text, w.fontFace, w.fontName)
		bitmap.SmartDrawTextInRect(img, text, w.fontFace, w.fontName, x, y, width+10, height, "left", vAlign, 0)
		return width

	case TokenLarge:
		// Large tokens are handled separately in renderLine/renderTokens
		return 0
	}
	return 0
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
		if weather != nil {
			iconName = getWeatherIconName(weather.Condition)
		} else {
			iconName = "sun" // default fallback
		}
	case "aqi_icon":
		iconName = getAQIIcon(aqi)
	case "uv_icon":
		iconName = getUVIcon(uv)
	case "humidity_icon":
		if weather != nil {
			iconName = getHumidityIcon(weather.Humidity)
		} else {
			iconName = "humidity_low"
		}
	case "wind_icon":
		if weather != nil {
			iconName = getWindIcon(weather.WindSpeed, w.units)
		} else {
			iconName = "wind_calm"
		}
	case "wind_dir_icon":
		if weather != nil {
			iconName = getWindDirIcon(weather.WindDirection)
		} else {
			iconName = "wind_n"
		}
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
			_, _ = fmt.Sscanf(parts[1], "+%d", &offset)
			if offset > 0 && offset <= len(forecast.Daily) {
				return getWeatherIconName(forecast.Daily[offset-1].Condition)
			}
		}
	} else if strings.HasPrefix(name, "hour:") {
		parts := strings.Split(name, ":")
		if len(parts) >= 2 {
			var offset int
			_, _ = fmt.Sscanf(parts[1], "+%d", &offset)
			targetTime := time.Now().Add(time.Duration(offset) * time.Hour)
			for _, p := range forecast.Hourly {
				if p.Time.After(targetTime.Add(-90 * time.Minute)) {
					return getWeatherIconName(p.Condition)
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
	// Use first forecast point if weather is nil
	var minTemp, maxTemp float64
	if weather != nil {
		minTemp = weather.Temperature
		maxTemp = weather.Temperature
	} else {
		minTemp = forecast.Hourly[0].Temperature
		maxTemp = forecast.Hourly[0].Temperature
	}
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
		// Note: maxTemp not updated as it's not used after this point
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

		iconName := getWeatherIconName(day.Condition)
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
	var text string
	if weather != nil {
		text = fmt.Sprintf("Now: %.0f%s %s", weather.Temperature, unit, weather.Description)
	} else {
		text = "No data"
	}

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
