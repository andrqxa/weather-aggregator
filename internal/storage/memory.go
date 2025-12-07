package storage

import (
	"strings"
	"sync"
	"time"

	"github.com/andrqxa/weather-aggregator/internal/weather"
)

type forecastKey struct {
	City string
	Days int
}

// InMemoryStore keeps latest and historical weather data in memory.
// It is safe for concurrent use.
type InMemoryStore struct {
	mu sync.RWMutex

	current   map[string]weather.CurrentWeather
	forecast  map[forecastKey]weather.Forecast
	lastFetch map[string]time.Time
}

// NewInMemoryStore creates a new empty in-memory store instance.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		current:   make(map[string]weather.CurrentWeather),
		forecast:  make(map[forecastKey]weather.Forecast),
		lastFetch: make(map[string]time.Time),
	}
}

// SaveCurrent stores latest current weather for a city and updates last fetch time.
func (s *InMemoryStore) SaveCurrent(city string, w weather.CurrentWeather, fetchedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.current[normalizeCity(city)] = w
	s.lastFetch[city] = fetchedAt
}

// GetCurrent returns latest current weather for a city if present.
func (s *InMemoryStore) GetCurrent(city string) (weather.CurrentWeather, bool) {
	s.mu.RLock()
	defer s.mu.Unlock()

	w, ok := s.current[normalizeCity(city)]
	return w, ok
}

// SaveForecast stores latest forecast for a city and number of days.
func (s *InMemoryStore) SaveForecast(city string, days int, f weather.Forecast, fetchedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := forecastKey{
		City: normalizeCity(city),
		Days: days,
	}

	s.forecast[key] = f
	s.lastFetch[normalizeCity(city)] = fetchedAt
}

// GetForecast returns latest forecast for a city and days if present.
func (s *InMemoryStore) GetForecast(city string, days int) (weather.Forecast, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := forecastKey{
		City: normalizeCity(city),
		Days: days,
	}

	f, ok := s.forecast[key]
	return f, ok
}

// LastFetchTimes returns a copy of last successful fetch timestamps per city.
func (s *InMemoryStore) LastFetchTimes() map[string]time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make(map[string]time.Time, len(s.lastFetch))
	for city, t := range s.lastFetch {
		res[city] = t
	}
	return res
}

// normalizeCity makes city key consistent (case-insensitive).
func normalizeCity(city string) string {
	return strings.ToLower(strings.Trim(city, " "))
}
