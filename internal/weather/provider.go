package weather

import (
	"context"
	"errors"
	"time"
)

// Provider describes a weather data provider.
//
// Implementations are expected to call external HTTP APIs and normalize
// responses into the domain models defined in models.go.
type Provider interface {
	// Name returns a human-readable provider identifier, e.g. "openmeteo".
	Name() string

	// FetchCurrent returns normalized current weather data for a given city.
	FetchCurrent(ctx context.Context, city string, from, to time.Time) (CurrentWeather, error)

	// FetchForecast returns normalized forecast for a given city
	// in the provided time range.
	FetchForecast(ctx context.Context, city string, from, to time.Time) (Forecast, error)
}

var (
	// ErrCityNotFound is returned when provider does not know the requested city.
	ErrCityNotFound = errors.New("city not found")

	// ErrProviderUnavailable is returned when provider cannot serve the request
	// due to temporary issues (network, rate limiting, etc.).
	ErrProviderUnavailable = errors.New("provider unavailable")
)
