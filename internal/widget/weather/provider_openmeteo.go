package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// OpenMeteoProvider implements WeatherProvider for Open-Meteo API
type OpenMeteoProvider struct {
	config     WeatherProviderConfig
	httpClient *http.Client
}

// NewOpenMeteoProvider creates a new Open-Meteo provider
func NewOpenMeteoProvider(cfg WeatherProviderConfig, client *http.Client) *OpenMeteoProvider {
	return &OpenMeteoProvider{
		config:     cfg,
		httpClient: client,
	}
}

// Name returns the provider name
func (p *OpenMeteoProvider) Name() string {
	return providerOpenMeteo
}

// FetchWeather fetches weather data from Open-Meteo API
func (p *OpenMeteoProvider) FetchWeather(needForecast bool) (*WeatherData, *ForecastData, error) {
	baseURL := "https://api.open-meteo.com/v1/forecast"
	params := url.Values{}
	params.Set("latitude", fmt.Sprintf("%f", p.config.Lat))
	params.Set("longitude", fmt.Sprintf("%f", p.config.Lon))
	params.Set("current", "temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m,wind_direction_10m,surface_pressure,visibility")
	params.Set("daily", "sunrise,sunset")
	params.Set("timezone", "auto")

	if needForecast {
		params.Set("hourly", "temperature_2m,weather_code")
		params.Add("daily", "temperature_2m_max,weather_code")
		params.Set("forecast_days", fmt.Sprintf("%d", p.config.ForecastDays+1))
	}

	if p.config.Units == unitsImperial {
		params.Set("temperature_unit", "fahrenheit")
		params.Set("wind_speed_unit", "mph")
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
		Current struct {
			Temperature      float64 `json:"temperature_2m"`
			RelativeHumidity int     `json:"relative_humidity_2m"`
			WeatherCode      int     `json:"weather_code"`
			WindSpeed        float64 `json:"wind_speed_10m"`
			WindDirection    float64 `json:"wind_direction_10m"`
			Pressure         float64 `json:"surface_pressure"`
			Visibility       float64 `json:"visibility"`
		} `json:"current"`
		Hourly struct {
			Time        []string  `json:"time"`
			Temperature []float64 `json:"temperature_2m"`
			WeatherCode []int     `json:"weather_code"`
		} `json:"hourly"`
		Daily struct {
			Time        []string  `json:"time"`
			TempMax     []float64 `json:"temperature_2m_max"`
			WeatherCode []int     `json:"weather_code"`
			Sunrise     []string  `json:"sunrise"`
			Sunset      []string  `json:"sunset"`
		} `json:"daily"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	condition := mapOpenMeteoWeatherCode(result.Current.WeatherCode)

	// Parse sunrise/sunset
	var sunrise, sunset time.Time
	if len(result.Daily.Sunrise) > 0 {
		sunrise, _ = time.Parse("2006-01-02T15:04", result.Daily.Sunrise[0])
	}
	if len(result.Daily.Sunset) > 0 {
		sunset, _ = time.Parse("2006-01-02T15:04", result.Daily.Sunset[0])
	}

	weatherData := &WeatherData{
		Temperature:   result.Current.Temperature,
		FeelsLike:     result.Current.Temperature, // Open-Meteo doesn't provide feels_like in free tier
		Condition:     condition,
		Description:   getWeatherDescription(condition),
		Humidity:      result.Current.RelativeHumidity,
		WindSpeed:     result.Current.WindSpeed,
		WindDirection: degreesToDirection(result.Current.WindDirection),
		Pressure:      result.Current.Pressure,
		Visibility:    result.Current.Visibility,
		Sunrise:       sunrise,
		Sunset:        sunset,
	}

	var forecastData *ForecastData
	if needForecast {
		forecastData = &ForecastData{
			Hourly: make([]ForecastPoint, 0),
			Daily:  make([]ForecastPoint, 0),
		}

		now := time.Now()
		for i := 0; i < len(result.Hourly.Time) && len(forecastData.Hourly) < p.config.ForecastHours; i++ {
			t, err := time.Parse("2006-01-02T15:04", result.Hourly.Time[i])
			if err != nil || t.Before(now) {
				continue
			}
			cond := mapOpenMeteoWeatherCode(result.Hourly.WeatherCode[i])
			forecastData.Hourly = append(forecastData.Hourly, ForecastPoint{
				Time:        t,
				Temperature: result.Hourly.Temperature[i],
				Condition:   cond,
				Description: getWeatherDescription(cond),
			})
		}

		for i := 0; i < len(result.Daily.Time) && i < p.config.ForecastDays; i++ {
			t, err := time.Parse("2006-01-02", result.Daily.Time[i])
			if err != nil {
				continue
			}
			cond := mapOpenMeteoWeatherCode(result.Daily.WeatherCode[i])
			forecastData.Daily = append(forecastData.Daily, ForecastPoint{
				Time:        t,
				Temperature: result.Daily.TempMax[i],
				Condition:   cond,
				Description: getWeatherDescription(cond),
			})
		}
	}

	return weatherData, forecastData, nil
}

// FetchAirQuality fetches air quality from Open-Meteo
func (p *OpenMeteoProvider) FetchAirQuality() (*AirQualityData, error) {
	baseURL := "https://air-quality-api.open-meteo.com/v1/air-quality"
	params := url.Values{}
	params.Set("latitude", fmt.Sprintf("%f", p.config.Lat))
	params.Set("longitude", fmt.Sprintf("%f", p.config.Lon))
	params.Set("current", "us_aqi,pm2_5,pm10")

	resp, err := p.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AQI API error: status %d", resp.StatusCode)
	}

	var result struct {
		Current struct {
			USAQI int     `json:"us_aqi"`
			PM25  float64 `json:"pm2_5"`
			PM10  float64 `json:"pm10"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &AirQualityData{
		AQI:   result.Current.USAQI,
		Level: getAQILevel(result.Current.USAQI),
		PM25:  result.Current.PM25,
		PM10:  result.Current.PM10,
	}, nil
}

// FetchUVIndex fetches UV index from Open-Meteo
func (p *OpenMeteoProvider) FetchUVIndex() (*UVIndexData, error) {
	baseURL := "https://api.open-meteo.com/v1/forecast"
	params := url.Values{}
	params.Set("latitude", fmt.Sprintf("%f", p.config.Lat))
	params.Set("longitude", fmt.Sprintf("%f", p.config.Lon))
	params.Set("daily", "uv_index_max")
	params.Set("forecast_days", "1")
	params.Set("timezone", "auto")

	resp, err := p.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("UV API error: status %d", resp.StatusCode)
	}

	var result struct {
		Daily struct {
			UVIndexMax []float64 `json:"uv_index_max"`
		} `json:"daily"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Daily.UVIndexMax) == 0 {
		return nil, fmt.Errorf("no UV data available")
	}

	uvIndex := result.Daily.UVIndexMax[0]
	return &UVIndexData{
		Index: uvIndex,
		Level: getUVLevel(uvIndex),
	}, nil
}

// mapOpenMeteoWeatherCode maps WMO weather code to condition
func mapOpenMeteoWeatherCode(code int) string {
	switch {
	case code == 0:
		return WeatherClear
	case code == 1 || code == 2:
		return WeatherPartlyCloudy
	case code == 3:
		return WeatherCloudy
	case code >= 45 && code <= 48:
		return WeatherFog
	case code >= 51 && code <= 57:
		return WeatherDrizzle
	case code >= 61 && code <= 67:
		return WeatherRain
	case code >= 71 && code <= 77:
		return WeatherSnow
	case code >= 80 && code <= 82:
		return WeatherRain
	case code >= 85 && code <= 86:
		return WeatherSnow
	case code >= 95 && code <= 99:
		return WeatherStorm
	default:
		return WeatherClear
	}
}
