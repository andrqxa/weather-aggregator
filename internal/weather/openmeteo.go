package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// / OpenMeteoProvider implements Provider using https://api.open-meteo.com.
// It does not require an API key and works with a fixed set of city → coordinates
// mappings that is sufficient for this test task.
type OpenMeteoProvider struct {
	client *http.Client
}

// NewOpenMeteoProvider creates a new OpenMeteoProvider with the given HTTP client.
// If client is nil, http.DefaultClient is used.
func NewOpenMeteoProvider(client *http.Client) *OpenMeteoProvider {
	if client == nil {
		client = http.DefaultClient
	}

	return &OpenMeteoProvider{
		client: client,
	}
}

// Name returns provider identifier.
func (p *OpenMeteoProvider) Name() string {
	return string(SourceOpenMeteo)
}

// coordinates holds a small, hard-coded city → lat/lon map for the test task.
type coordinates struct {
	Lat float64
	Lon float64
}

var openMeteoCityCoords = map[string]coordinates{
	"london": {
		Lat: 51.5074,
		Lon: -0.1278,
	},
	"paris": {
		Lat: 48.8566,
		Lon: 2.3522,
	},
	"warsaw": {
		Lat: 52.2297,
		Lon: 21.0122,
	},
}

// ---- OpenMeteo DTO ----

type openMeteoCurrentResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	CurrentWeather struct {
		Temperature float64 `json:"temperature"` // °C
		Humidity    int     `json:"humidity"`    // %
		WindSpeed   float64 `json:"windspeed"`   // km/h
		WeatherCode int     `json:"weathercode"`
		Time        string  `json:"time"` // ISO8601
	} `json:"current_weather"`
}

// For forecast take the hourly-data and fold them into the plain list.
type openMeteoForecastResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	Hourly struct {
		Time        []string  `json:"time"`
		Temperature []float64 `json:"temperature_2m"`
		Humidity    []int     `json:"humidity_2m"`
		WindSpeed   []float64 `json:"windspeed_10m"`
		WeatherCode []int     `json:"weathercode"`
	} `json:"hourly"`
}

// FetchCurrent returns normalized current weather for a given city using OpenMeteo.
func (p *OpenMeteoProvider) FetchCurrent(ctx context.Context, city string) (CurrentWeather, error) {
	coords, ok := openMeteoCityCoords[normalizeCity(city)]
	if !ok {
		return CurrentWeather{}, ErrCityNotFound
	}

	endpoint := "https://api.open-meteo.com/v1/forecast"

	q := url.Values{}
	q.Set("latitude", fmt.Sprintf("%f", coords.Lat))
	q.Set("longitude", fmt.Sprintf("%f", coords.Lon))
	q.Set("current_weather", "true")

	u := endpoint + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		slog.Error("failed to create OpenMeteo request",
			"city", city,
			"error", err,
		)
		return CurrentWeather{}, ErrProviderUnavailable
	}

	resp, err := p.client.Do(req)
	if err != nil {
		// ctx cancellation / timeout will be here too
		slog.Warn("OpenMeteo request failed",
			"city", city,
			"error", err,
		)
		return CurrentWeather{}, ErrProviderUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("OpenMeteo returned non-200 status",
			"city", city,
			"status", resp.StatusCode,
		)
		return CurrentWeather{}, ErrProviderUnavailable
	}

	var omResp openMeteoCurrentResponse
	if err := json.NewDecoder(resp.Body).Decode(&omResp); err != nil {
		slog.Warn("failed to decode OpenMeteo current response",
			"city", city,
			"error", err,
		)
		return CurrentWeather{}, ErrProviderUnavailable
	}

	observedAt := time.Now().UTC()
	if omResp.CurrentWeather.Time != "" {
		if t, err := time.Parse(time.RFC3339, omResp.CurrentWeather.Time); err == nil {
			observedAt = t
		}
	}

	cw := CurrentWeather{
		City:        city,
		Temperature: omResp.CurrentWeather.Temperature,
		Humidity:    omResp.CurrentWeather.Humidity,
		WindSpeed:   omResp.CurrentWeather.WindSpeed,
		//Description: omResp.CurrentWeather.WeatherCode,
		Source:     SourceOpenMeteo,
		ObservedAt: observedAt,
	}

	return cw, nil
}

// FetchForecast returns normalized forecast for the given city and days
// using OpenMeteo hourly forecast. Implementation is intentionally minimal
// but demonstrates real HTTP integration.
func (p *OpenMeteoProvider) FetchForecast(ctx context.Context, city string, days int) (Forecast, error) {
	coords, ok := openMeteoCityCoords[normalizeCity(city)]
	if !ok {
		return Forecast{}, ErrCityNotFound
	}

	endpoint := "https://api.open-meteo.com/v1/forecast"

	q := url.Values{}
	q.Set("latitude", fmt.Sprintf("%f", coords.Lat))
	q.Set("longitude", fmt.Sprintf("%f", coords.Lon))
	q.Set("hourly", "temperature_2m,weathercode,windspeed_10m,relativehumidity_2m")
	q.Set("forecast_days", fmt.Sprintf("%d", days))
	q.Set("timezone", "UTC")

	u := endpoint + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		slog.Error("failed to create OpenMeteo forecast request",
			"city", city,
			"days", days,
			"error", err,
		)
		return Forecast{}, ErrProviderUnavailable
	}

	resp, err := p.client.Do(req)
	if err != nil {
		slog.Warn("OpenMeteo forecast request failed",
			"city", city,
			"days", days,
			"error", err,
		)
		return Forecast{}, ErrProviderUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("OpenMeteo forecast returned non-200 status",
			"city", city,
			"days", days,
			"status", resp.StatusCode,
		)
		return Forecast{}, ErrProviderUnavailable
	}

	var omResp openMeteoForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&omResp); err != nil {
		slog.Warn("failed to decode OpenMeteo forecast response",
			"city", city,
			"days", days,
			"error", err,
		)
		return Forecast{}, ErrProviderUnavailable
	}

	items := make([]ForecastItem, 0, len(omResp.Hourly.Time))

	for i := range omResp.Hourly.Time {
		tStr := omResp.Hourly.Time[i]
		t, err := time.Parse(time.RFC3339, tStr)
		if err != nil {
			continue
		}

		item := ForecastItem{
			TimeStamp:   t,
			Temperature: safeIndexFloat(omResp.Hourly.Temperature, i),
			//WindSpeed:   safeIndexFloat(omResp.Hourly.WindSpeed, i),
			Source: SourceOpenMeteo,
		}

		items = append(items, item)
	}

	fc := Forecast{
		City:  city,
		Days:  days,
		Items: items,
	}

	return fc, nil
}

func safeIndexFloat(xs []float64, i int) float64 {
	if i < 0 || i >= len(xs) {
		return 0
	}
	return xs[i]
}

func normalizeCity(city string) string {
	return strings.ToLower(strings.TrimSpace(city))
}
