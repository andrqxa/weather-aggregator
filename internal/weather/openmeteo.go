package weather

import (
	"context"
	"time"
)

// OpenMeteoProvider is a stub implementation of Provider for the Open-Meteo API.
// Real HTTP calls and response mapping will be implemented later.
type OpenMeteoProvider struct {
	baseURL string
}

// NewOpenMeteoProvider creates a new OpenMeteoProvider instance.
func NewOpenMeteoProvider(baseURL string) *OpenMeteoProvider {
	return &OpenMeteoProvider{
		baseURL: baseURL,
	}
}

// Name returns provider identifier.
func (p *OpenMeteoProvider) Name() string {
	return string(SourceOpenMeteo)
}

// FetchCurrent returns stubbed error for now.
// Real implementation will call external API.
func (p *OpenMeteoProvider) FetchCurrent(ctx context.Context, city string) (CurrentWeather, error) {
	return CurrentWeather{}, ErrProviderUnavailable
}

// FetchForecast returns stubbed error for now.
// Real implementation will call external API.
func (p *OpenMeteoProvider) FetchForecast(ctx context.Context, city string, from, to time.Time) (Forecast, error) {
	return Forecast{}, ErrProviderUnavailable
}
