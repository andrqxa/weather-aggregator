package weather

import (
	"context"
)

// OpenWeatherMapProvider is a stub implementation of Provider for the OpenWeather API.
// Real HTTP calls and response mapping will be implemented later.
type OpenWeatherMapProvider struct {
	baseURL string
	apiKey  string
}

// NewOpenWeatherMapProvider creates a new OpenWeatherMapProvider instance.
func NewOpenWeatherMapProvider(apiKey string) *OpenWeatherMapProvider {
	return &OpenWeatherMapProvider{
		baseURL: "https://api.openweathermap.org/data/2.5",
		apiKey:  apiKey,
	}
}

// Name returns provider identifier.
func (p *OpenWeatherMapProvider) Name() string {
	return string(SourceOpenWeather)
}

// FetchCurrent returns stubbed error for now.
// Real implementation will call external API.
func (p *OpenWeatherMapProvider) FetchCurrent(ctx context.Context, city string) (CurrentWeather, error) {
	return CurrentWeather{}, ErrProviderUnavailable
}

// FetchForecast returns stubbed error for now.
// Real implementation will call external API.
func (p *OpenWeatherMapProvider) FetchForecast(ctx context.Context, city string, days int) (Forecast, error) {
	return Forecast{}, ErrProviderUnavailable
}
