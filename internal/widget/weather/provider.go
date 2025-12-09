package weather

// WeatherProvider defines the interface for weather data providers
type WeatherProvider interface {
	// FetchWeather fetches current weather and optionally forecast
	FetchWeather(needForecast bool) (*WeatherData, *ForecastData, error)

	// FetchAirQuality fetches air quality data (may return nil, nil if not supported)
	FetchAirQuality() (*AirQualityData, error)

	// FetchUVIndex fetches UV index data (may return nil, nil if not supported)
	FetchUVIndex() (*UVIndexData, error)

	// Name returns the provider name for logging
	Name() string
}

// WeatherProviderConfig holds common configuration for weather providers
type WeatherProviderConfig struct {
	City          string
	Lat           float64
	Lon           float64
	Units         string
	ForecastHours int
	ForecastDays  int
}
