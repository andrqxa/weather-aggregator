package scheduler

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/andrqxa/weather-aggregator/internal/storage"
	"github.com/andrqxa/weather-aggregator/internal/weather"
)

// Scheduler periodically fetches weather data for configured cities
// and stores results in the in-memory storage.
type Scheduler struct {
	service        *weather.Service
	store          *storage.InMemoryStore
	cities         []string
	interval       time.Duration
	requestTimeout time.Duration
	defaultDays    int

	log     *slog.Logger
	running int32 // 0 - idle, 1 - job in progress
}

// NewScheduler creates a new Scheduler instance.
func NewScheduler(
	service *weather.Service,
	store *storage.InMemoryStore,
	cities []string,
	interval time.Duration,
	requestTimeout time.Duration,
	defaultDays int,
	log *slog.Logger,
) *Scheduler {
	return &Scheduler{
		service:        service,
		store:          store,
		cities:         cities,
		interval:       interval,
		requestTimeout: requestTimeout,
		defaultDays:    defaultDays,
		log:            log,
	}
}

// Start runs periodic jobs until the context is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	s.log.Info("scheduler started",
		"interval", s.interval.String(),
		"cities", s.cities,
	)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("scheduler stopping due to context cancellation")
			return
		case <-ticker.C:
			s.runOnce()
		}
	}
}

// runOnce executes a single scheduler tick.
// It ensures that jobs do not overlap using an atomic flag.
func (s *Scheduler) runOnce() {
	// Prevent overlapping runs.
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		s.log.Warn("previous scheduler run still in progress, skipping this tick")
		return
	}
	defer atomic.StoreInt32(&s.running, 0)

	start := time.Now()
	s.log.Info("scheduler tick started")

	for _, city := range s.cities {
		s.runForCity(city)
	}

	duration := time.Since(start)
	s.log.Info("scheduler tick finished",
		"duration", duration.String(),
		"cities", len(s.cities),
	)
}

// runForCity fetches current weather and forecast for a single city
// and stores results in the in-memory storage.
func (s *Scheduler) runForCity(city string) {
	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()

	s.log.Info("scheduler fetching weather",
		"city", city,
		"days", s.defaultDays,
	)

	// Fetch current weather.
	current, err := s.service.GetCurrentWeather(ctx, city)
	if err != nil {
		s.log.Warn("scheduler failed to fetch current weather",
			"city", city,
			"error", err,
		)
	} else {
		s.store.SaveCurrent(city, current, time.Now().UTC())
	}

	// Fetch forecast.
	forecast, err := s.service.GetForecast(ctx, city, s.defaultDays)
	if err != nil {
		s.log.Warn("scheduler failed to fetch forecast",
			"city", city,
			"days", s.defaultDays,
			"error", err,
		)
	} else {
		s.store.SaveForecast(city, s.defaultDays, forecast, time.Now().UTC())
	}
}
