package weather

import (
	"context"
	"time"
)

// OpenWeatherProvider is a stub implementation of Provider for the OpenWeather API.
// Real HTTP calls and response mapping will be implemented later.
type OpenWeatherProvider struct {
	baseURL string
	apiKey  string
}

// NewOpenWeatherProvider creates a new OpenWeatherProvider instance.
func NewOpenWeatherProvider(baseURL, apiKey string) *OpenWeatherProvider {
	return &OpenWeatherProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// Name returns provider identifier.
func (p *OpenWeatherProvider) Name() string {
	return string(SourceOpenWeather)
}

// FetchCurrent returns stubbed error for now.
// Real implementation will call external API.
func (p *OpenWeatherProvider) FetchCurrent(ctx context.Context, city string) (CurrentWeather, error) {
	return CurrentWeather{}, ErrProviderUnavailable
}

// FetchForecast returns stubbed error for now.
// Real implementation will call external API.
func (p *OpenWeatherProvider) FetchForecast(ctx context.Context, city string, from, to time.Time) (Forecast, error) {
	return Forecast{}, ErrProviderUnavailable
}
