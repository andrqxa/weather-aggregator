package weather

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

type Service struct {
	providers []Provider
}

type result[T any] struct {
	provider Provider
	data     T
	err      error
}

func NewService(providers []Provider) *Service {
	return &Service{
		providers: providers,
	}
}

// GetCurrentWeather concurrently fetches current weather from all providers,
// logs individual provider errors and aggregates successful results.
func (s *Service) GetCurrentWeather(ctx context.Context, city string) (CurrentWeather, error) {
	if len(s.providers) == 0 {
		return CurrentWeather{}, ErrProviderUnavailable
	}

	resultsCh := make(chan result[CurrentWeather], len(s.providers))
	var wg sync.WaitGroup

	for _, prov := range s.providers {
		p := prov // capture, because WaitGroup.Go is not "go func()"
		wg.Go(func() {
			slog.Info("fetching current weather",
				"provider", p.Name(),
				"city", city,
			)

			w, err := p.FetchCurrent(ctx, city)

			resultsCh <- result[CurrentWeather]{
				provider: p,
				data:     w,
				err:      err,
			}
		})
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var (
		successes []CurrentWeather
		lastErr   error
	)

	for res := range resultsCh {
		if res.err != nil {
			logProviderError("current", res.provider, city, res.err)
			lastErr = res.err
			continue
		}
		successes = append(successes, res.data)
	}

	if len(successes) == 0 {
		if lastErr != nil {
			slog.Warn("all providers failed for current weather",
				"city", city,
				"error", lastErr,
			)
		}
		return CurrentWeather{}, ErrProviderUnavailable
	}

	agg := AggregateCurrentWeather(successes)
	return agg, nil
}

// GetForecast concurrently fetches forecast data from all providers,
// logs individual provider errors and aggregates successful results.
func (s *Service) GetForecast(ctx context.Context, city string, days int) (Forecast, error) {
	if len(s.providers) == 0 {
		return Forecast{}, ErrProviderUnavailable
	}

	resultsCh := make(chan result[Forecast], len(s.providers))
	var wg sync.WaitGroup

	for _, prov := range s.providers {
		p := prov
		wg.Go(func() {
			slog.Info("fetching forecast",
				"provider", p.Name(),
				"city", city,
				"days", days,
			)

			fc, err := p.FetchForecast(ctx, city, days)

			resultsCh <- result[Forecast]{
				provider: p,
				data:     fc,
				err:      err,
			}
		})
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var (
		successes []Forecast
		lastErr   error
	)

	for res := range resultsCh {
		if res.err != nil {
			logProviderError("forecast", res.provider, city, res.err)
			lastErr = res.err
			continue
		}
		successes = append(successes, res.data)
	}

	if len(successes) == 0 {
		if lastErr != nil {
			slog.Warn("all providers failed for forecast",
				"city", city,
				"days", days,
				"error", lastErr,
			)
		}
		return Forecast{}, ErrProviderUnavailable
	}

	agg := AggregateForecast(successes)
	return agg, nil
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
