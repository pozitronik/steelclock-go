package weather

// Provider WeatherProvider defines the interface for weather data providers
type Provider interface {
	// FetchWeather fetches current weather and optionally forecast
	FetchWeather(needForecast bool) (*WData, *ForecastData, error)

	// FetchAirQuality fetches air quality data (may return nil, nil if not supported)
	FetchAirQuality() (*AirQualityData, error)

	// FetchUVIndex fetches UV index data (may return nil, nil if not supported)
	FetchUVIndex() (*UVIndexData, error)

	// Name returns the provider name for logging
	Name() string
}

// ProviderConfig WeatherProviderConfig holds common configuration for weather providers
type ProviderConfig struct {
	City          string
	Lat           float64
	Lon           float64
	Units         string
	ForecastHours int
	ForecastDays  int
}
