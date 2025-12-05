package weather

import "time"

// Source represents a weather data provider.
type Source string

const (
	SourceOpenWeather Source = "openweather"
	SourceOpenMeteo   Source = "openmeteo"
	SourceWeatherAPI  Source = "weatherapi"
)

// CurrentWeather represents normalized current weather data.
type CurrentWeather struct {
	City        string    `json:"city"`
	Temperature float64   `json:"temperature"` // Celsius
	Humidity    int       `json:"humidity"`    // %
	WindSpeed   float64   `json:"wind_speed"`  // m/s
	Description string    `json:"description"`
	Source      Source    `json:"source"`
	ObservedAt  time.Time `json:"observed_at"`
}

// ForecastItem represents a single forecast point.
type ForecastItem struct {
	TimeStamp   time.Time `json:"timestamp"`
	Temperature float64   `json:"temperature"` // Celsius
	Description string    `json:"description"`
	Source      Source    `json:"source"`
}

// Forecast represents normalized forecast for a city.
type Forecast struct {
	City      string         `json:"city"`
	Items     []ForecastItem `json:"items"`
	From      time.Time      `json:"from"`
	To        time.Time      `json:"to"`
	Source    Source         `json:"source"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// AggregatedWeather is what we will store and serve via API.
type AggregatedWeather struct {
	Current  CurrentWeather `json:"current"`
	Forecast Forecast       `json:"forecast"`
}
