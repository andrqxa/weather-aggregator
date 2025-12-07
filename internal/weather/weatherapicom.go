package weather

import (
	"context"
	"time"
)

// WeatherAPIComProvider is a stub implementation of Provider for the WeatherAPICom API.
// Real HTTP calls and response mapping will be implemented later.
type WeatherAPIComProvider struct {
	baseURL string
	apiKey  string
}

// NewWeatherAPIComProvider creates a new WeatherAPIComProvider instance.
func NewWeatherAPIComProvider(apiKey string) *WeatherAPIComProvider {
	return &WeatherAPIComProvider{
		baseURL: "https://api.weatherapi.com/v1",
		apiKey:  apiKey,
	}
}

// Name returns provider identifier.
func (p *WeatherAPIComProvider) Name() string {
	return string(SourceWeatherAPI)
}

// FetchCurrent returns stubbed error for now.
// Real implementation will call external API.
func (p *WeatherAPIComProvider) FetchCurrent(ctx context.Context, city string) (CurrentWeather, error) {
	return CurrentWeather{}, ErrProviderUnavailable
}

// FetchForecast returns stubbed error for now.
// Real implementation will call external API.
func (p *WeatherAPIComProvider) FetchForecast(ctx context.Context, city string, from, to time.Time) (Forecast, error) {
	return Forecast{}, ErrProviderUnavailable
}
