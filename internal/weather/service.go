package weather

import (
	"context"
	"errors"
	"log/slog"
)

type Service struct {
	providers []Provider
}

func NewService(providers []Provider) *Service {
	return &Service{
		providers: providers,
	}
}

// GetCurrentWeather tries to fetch current weather from the first available provider.
// Later this may aggregate results or choose best source.
func (s *Service) GetCurrentWeather(ctx context.Context, city string) (CurrentWeather, error) {
	for _, p := range s.providers {
		slog.Info("fetching current weather", "provider", p.Name(), "city", city)

		w, err := p.FetchCurrent(ctx, city)
		if err != nil {
			logProviderError("current", p, city, err)
			continue
		}

		// TODO: in future, allow merging instead of first success
		return w, nil
	}
	return CurrentWeather{}, ErrProviderUnavailable
}

// GetForecast tries the providers sequentially.
// Later this will be improved with aggregation.
func (s *Service) GetForecast(ctx context.Context, city string, days int) (Forecast, error) {
	for _, p := range s.providers {
		slog.Info("fetching forecast", "provider", p.Name(), "city", city)

		w, err := p.FetchForecast(ctx, city, days)
		if err != nil {
			logProviderError("forecast", p, city, err)
			continue
		}
		// TODO: in future, allow merging instead of first success
		return w, nil
	}
	return Forecast{}, ErrProviderUnavailable
}

func logProviderError(op string, p Provider, city string, err error) {
	switch {
	case errors.Is(err, ErrProviderUnavailable):
		slog.Warn("provider unavailable",
			"op", op,
			"provider", p.Name(),
			"city", city,
			"error", err)

	case errors.Is(err, ErrCityNotFound):
		slog.Warn("city not found for provider",
			"op", op,
			"provider", p.Name(),
			"city", city,
			"error", err)

	default:
		slog.Warn("unexpected provider error",
			"op", op,
			"provider", p.Name(),
			"city", city,
			"error", err)
	}
}
