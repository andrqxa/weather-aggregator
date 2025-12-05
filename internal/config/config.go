package config

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application configuration values
type Config struct {
	Port          string
	FetchInterval time.Duration
	WeatherAPIKey string
	DefaultCities []string
}

// Load loads configuration from environment variables or .env file.
func Load() *Config {
	// Load .env file if present, ignore error silently
	_ = godotenv.Load()

	return &Config{
		Port:          getEnv("FIBER_PORT", "3000"),
		WeatherAPIKey: getEnv("WEATHER_API_KEY", ""),
		FetchInterval: getDuration("FETCH_INTERVAL", 15*time.Minute),
		DefaultCities: parseCities(getEnv("DEFAULT_CITIES", "London")),
	}
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
		slog.Warn("invalid duration",
			"key", key,
			"value", v,
			"default", defaultValue.String(),
		)
	}
	return defaultValue
}

func getEnv(key string, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultValue
}

func parseCities(raw string) []string {
	parts := strings.Split(raw, ",")
	res := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, p)
		}

	}
	return res
}
