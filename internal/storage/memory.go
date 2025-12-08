package storage

import (
	"strings"
	"sync"
	"time"

	"github.com/andrqxa/weather-aggregator/internal/weather"
)

const maxHistoryEntries = 50

type forecastKey struct {
	City string
	Days int
}

type CurrentSnapshot struct {
	At   time.Time
	Data weather.CurrentWeather
}

type ForecastSnapshot struct {
	At   time.Time
	Days int
	Data weather.Forecast
}

// InMemoryStore keeps latest and historical weather data in memory.
// It is safe for concurrent use.
type InMemoryStore struct {
	mu sync.RWMutex

	current   map[string]weather.CurrentWeather
	forecast  map[forecastKey]weather.Forecast
	lastFetch map[string]time.Time

	currentHistory  map[string][]CurrentSnapshot
	forecastHistory map[forecastKey][]ForecastSnapshot
}

// NewInMemoryStore creates a new empty in-memory store instance.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		current:         make(map[string]weather.CurrentWeather),
		forecast:        make(map[forecastKey]weather.Forecast),
		lastFetch:       make(map[string]time.Time),
		currentHistory:  make(map[string][]CurrentSnapshot),
		forecastHistory: make(map[forecastKey][]ForecastSnapshot),
	}
}

// SaveCurrent stores latest current weather for a city, updates last fetch time
// and appends entry to the history with a bounded size.
func (s *InMemoryStore) SaveCurrent(city string, w weather.CurrentWeather, fetchedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := normalizeCity(city)

	s.current[key] = w
	s.lastFetch[key] = fetchedAt

	h := s.currentHistory[key]
	h = append(h, CurrentSnapshot{
		At:   fetchedAt,
		Data: w,
	})
	if len(h) > maxHistoryEntries {
		h = h[len(h)-maxHistoryEntries:]
	}
	s.currentHistory[key] = h
}

// GetCurrent returns latest current weather for a city if present.
func (s *InMemoryStore) GetCurrent(city string) (weather.CurrentWeather, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w, ok := s.current[normalizeCity(city)]
	return w, ok
}

// SaveForecast stores latest forecast for a city and number of days,
// updates last fetch time and appends entry to the history
// with a bounded size.
func (s *InMemoryStore) SaveForecast(city string, days int, f weather.Forecast, fetchedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalizedCity := normalizeCity(city)

	key := forecastKey{
		City: normalizedCity,
		Days: days,
	}

	s.forecast[key] = f
	s.lastFetch[normalizedCity] = fetchedAt

	h := s.forecastHistory[key]
	h = append(h, ForecastSnapshot{
		At:   fetchedAt,
		Days: days,
		Data: f,
	})
	if len(h) > maxHistoryEntries {
		h = h[len(h)-maxHistoryEntries:]
	}
	s.forecastHistory[key] = h
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

// CurrentHistory returns up to `limit` recent current weather snapshots
// for the given city. If limit <= 0 or greater than available entries,
// all entries are returned.
func (s *InMemoryStore) CurrentHistory(city string, limit int) []CurrentSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := normalizeCity(city)
	h := s.currentHistory[key]

	if len(h) == 0 {
		return nil
	}

	if limit <= 0 || limit >= len(h) {
		res := make([]CurrentSnapshot, len(h))
		copy(res, h)
		return res
	}

	res := make([]CurrentSnapshot, limit)
	copy(res, h[len(h)-limit:])
	return res
}

// ForecastHistory returns up to `limit` recent forecast snapshots
// for the given (city, days) pair. If limit <= 0 or greater than
// available entries, all entries are returned.
func (s *InMemoryStore) ForecastHistory(city string, days, limit int) []ForecastSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := forecastKey{
		City: normalizeCity(city),
		Days: days,
	}
	h := s.forecastHistory[key]

	if len(h) == 0 {
		return nil
	}

	if limit <= 0 || limit >= len(h) {
		res := make([]ForecastSnapshot, len(h))
		copy(res, h)
		return res
	}

	res := make([]ForecastSnapshot, limit)
	copy(res, h[len(h)-limit:])
	return res
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
	return strings.ToLower(strings.TrimSpace(city))
}
