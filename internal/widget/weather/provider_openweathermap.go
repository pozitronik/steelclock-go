package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// OpenWeatherMapProvider implements WeatherProvider for OpenWeatherMap API
type OpenWeatherMapProvider struct {
	config     WeatherProviderConfig
	apiKey     string
	httpClient *http.Client
}

// NewOpenWeatherMapProvider creates a new OpenWeatherMap provider
func NewOpenWeatherMapProvider(cfg WeatherProviderConfig, apiKey string, client *http.Client) *OpenWeatherMapProvider {
	return &OpenWeatherMapProvider{
		config:     cfg,
		apiKey:     apiKey,
		httpClient: client,
	}
}

// Name returns the provider name
func (p *OpenWeatherMapProvider) Name() string {
	return providerOpenWeatherMap
}

// FetchWeather fetches weather data from OpenWeatherMap API
func (p *OpenWeatherMapProvider) FetchWeather(needForecast bool) (*Data, *ForecastData, error) {
	baseURL := "https://api.openweathermap.org/data/2.5/weather"
	params := url.Values{}
	params.Set("appid", p.apiKey)
	params.Set("units", p.config.Units)

	if p.config.City != "" {
		params.Set("q", p.config.City)
	} else {
		params.Set("lat", fmt.Sprintf("%f", p.config.Lat))
		params.Set("lon", fmt.Sprintf("%f", p.config.Lon))
	}

	resp, err := p.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Main struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			Humidity  int     `json:"humidity"`
			Pressure  float64 `json:"pressure"`
		} `json:"main"`
		Weather []struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
		} `json:"weather"`
		Wind struct {
			Speed float64 `json:"speed"`
			Deg   float64 `json:"deg"`
		} `json:"wind"`
		Visibility int `json:"visibility"`
		Sys        struct {
			Sunrise int64 `json:"sunrise"`
			Sunset  int64 `json:"sunset"`
		} `json:"sys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	condition := Clear
	description := ""
	if len(result.Weather) > 0 {
		condition = mapOpenWeatherMapCondition(result.Weather[0].ID)
		description = result.Weather[0].Description
	}

	weatherData := &Data{
		Temperature:   result.Main.Temp,
		FeelsLike:     result.Main.FeelsLike,
		Condition:     condition,
		Description:   description,
		Humidity:      result.Main.Humidity,
		WindSpeed:     result.Wind.Speed,
		WindDirection: degreesToDirection(result.Wind.Deg),
		Pressure:      result.Main.Pressure,
		Visibility:    float64(result.Visibility),
		Sunrise:       time.Unix(result.Sys.Sunrise, 0),
		Sunset:        time.Unix(result.Sys.Sunset, 0),
	}

	var forecastData *ForecastData
	if needForecast {
		forecastData, _ = p.fetchForecast()
	}

	return weatherData, forecastData, nil
}

// fetchForecast fetches forecast from OpenWeatherMap
func (p *OpenWeatherMapProvider) fetchForecast() (*ForecastData, error) {
	baseURL := "https://api.openweathermap.org/data/2.5/forecast"
	params := url.Values{}
	params.Set("appid", p.apiKey)
	params.Set("units", p.config.Units)

	if p.config.City != "" {
		params.Set("q", p.config.City)
	} else {
		params.Set("lat", fmt.Sprintf("%f", p.config.Lat))
		params.Set("lon", fmt.Sprintf("%f", p.config.Lon))
	}

	resp, err := p.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forecast API error: status %d", resp.StatusCode)
	}

	var result struct {
		List []struct {
			Dt   int64 `json:"dt"`
			Main struct {
				Temp float64 `json:"temp"`
			} `json:"main"`
			Weather []struct {
				ID          int    `json:"id"`
				Description string `json:"description"`
			} `json:"weather"`
		} `json:"list"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	forecast := &ForecastData{
		Hourly: make([]ForecastPoint, 0),
		Daily:  make([]ForecastPoint, 0),
	}

	dailyMap := make(map[string]ForecastPoint)
	for i, item := range result.List {
		t := time.Unix(item.Dt, 0)
		condition := Clear
		description := ""
		if len(item.Weather) > 0 {
			condition = mapOpenWeatherMapCondition(item.Weather[0].ID)
			description = item.Weather[0].Description
		}

		point := ForecastPoint{
			Time:        t,
			Temperature: item.Main.Temp,
			Condition:   condition,
			Description: description,
		}

		if i < p.config.ForecastHours/3 {
			forecast.Hourly = append(forecast.Hourly, point)
		}

		dayKey := t.Format("2006-01-02")
		if _, exists := dailyMap[dayKey]; !exists || t.Hour() == 12 {
			dailyMap[dayKey] = point
		}
	}

	// Sort and limit daily forecast
	days := make([]string, 0, len(dailyMap))
	for day := range dailyMap {
		days = append(days, day)
	}
	for i := 0; i < len(days)-1; i++ {
		for j := i + 1; j < len(days); j++ {
			if days[i] > days[j] {
				days[i], days[j] = days[j], days[i]
			}
		}
	}
	for i, day := range days {
		if i >= p.config.ForecastDays {
			break
		}
		forecast.Daily = append(forecast.Daily, dailyMap[day])
	}

	return forecast, nil
}

// FetchAirQuality fetches air quality data from OpenWeatherMap
func (p *OpenWeatherMapProvider) FetchAirQuality() (*AirQualityData, error) {
	baseURL := "https://api.openweathermap.org/data/2.5/air_pollution"
	params := url.Values{}
	params.Set("appid", p.apiKey)
	params.Set("lat", fmt.Sprintf("%f", p.config.Lat))
	params.Set("lon", fmt.Sprintf("%f", p.config.Lon))

	resp, err := p.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AQI API error: status %d", resp.StatusCode)
	}

	var result struct {
		List []struct {
			Main struct {
				AQI int `json:"aqi"` // 1-5 scale
			} `json:"main"`
			Components struct {
				PM25 float64 `json:"pm2_5"`
				PM10 float64 `json:"pm10"`
			} `json:"components"`
		} `json:"list"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.List) == 0 {
		return nil, fmt.Errorf("no AQI data available")
	}

	data := result.List[0]
	// Convert EU AQI (1-5) to US AQI approximation
	usAQI := data.Main.AQI * 40 // Rough conversion
	level := getAQILevel(usAQI)

	return &AirQualityData{
		AQI:   usAQI,
		Level: level,
		PM25:  data.Components.PM25,
		PM10:  data.Components.PM10,
	}, nil
}

// FetchUVIndex returns nil as OpenWeatherMap doesn't have a free UV endpoint
func (p *OpenWeatherMapProvider) FetchUVIndex() (*UVIndexData, error) {
	return nil, nil
}

// mapOpenWeatherMapCondition maps OpenWeatherMap weather ID to condition
func mapOpenWeatherMapCondition(id int) string {
	switch {
	case id >= 200 && id < 300:
		return Storm
	case id >= 300 && id < 400:
		return Drizzle
	case id >= 500 && id < 600:
		return Rain
	case id >= 600 && id < 700:
		return Snow
	case id >= 700 && id < 800:
		return Fog
	case id == 800:
		return Clear
	case id == 801:
		return PartlyCloudy
	case id >= 802:
		return Cloudy
	default:
		return Clear
	}
}
