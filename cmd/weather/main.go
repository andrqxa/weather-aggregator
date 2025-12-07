package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/andrqxa/weather-aggregator/internal/config"
	"github.com/andrqxa/weather-aggregator/internal/weather"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func initLogger() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logg := slog.New(handler)
	slog.SetDefault(logg)
	return logg
}

func main() {

	// Init logger
	log := initLogger()

	//Init config
	cfg := config.Load()

	log.Info("configuration loaded",
		"port", cfg.Port,
		"fetch_interval", cfg.FetchInterval.String(),
		"openweathermap_key_set", cfg.OpenWeatherMapAPIKey != "",
		"weatherapi_key_set", cfg.WeatherAPIKey != "",
		"request_timeout", cfg.RequestTimeout.String(),
		"default_cities", cfg.DefaultCities,
	)

	// Initialize weather providers.
	providers := initProviders(cfg)

	svc := weather.NewService(providers)

	// Fiber init
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Log unexpected/unhandled error
			slog.Error("unhandled fiber error", "error", err)

			// Do not leak internal details to the client
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal server error",
			})
		},
	})

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New())

	// API routing
	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Health check
	v1.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":             "ok",
			"default_cities":     cfg.DefaultCities,
			"fetch_interval":     cfg.FetchInterval.String(),
			"openweathermap_key": cfg.OpenWeatherMapAPIKey != "",
			"weatherapi_key":     cfg.WeatherAPIKey != "",
			"request_timeout":    cfg.RequestTimeout.String(),
		})
	})

	weatherGroup := v1.Group("/weather")

	// GET /api/v1/weather/current?city=London
	weatherGroup.Get("/current", func(c *fiber.Ctx) error {
		city := c.Query("city")
		if city == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "city query parameter is required",
			})
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		defer cancel()

		w, err := svc.GetCurrentWeather(ctx, city)
		if err != nil {
			return mapServiceError(c, err)
		}
		return c.JSON(w)
	})

	// GET /api/v1/weather/forecast?city=London&from=2025-12-05T00:00:00Z&to=2025-12-06T00:00:00Z
	weatherGroup.Get("/forecast", func(c *fiber.Ctx) error {
		city := c.Query("city")
		if city == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "city query parameter is required",
			})
		}

		fromStr := c.Query("from")
		toStr := c.Query("to")

		var (
			from time.Time
			to   time.Time
			err  error
		)

		if fromStr == "" && toStr == "" {
			// Default: forecast for the next 24 hours
			from = time.Now().UTC()
			to = from.Add(24 * time.Hour)
		} else {
			if fromStr == "" || toStr == "" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "both from and to query parameters must be provided or both omitted",
				})
			}

			from, err = time.Parse(time.RFC3339, fromStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "invalid from parameter, expected RFC3339",
					"example": time.Now().UTC().Format(time.RFC3339),
				})
			}

			to, err = time.Parse(time.RFC3339, toStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "invalid to parameter, expected RFC3339",
					"example": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
				})
			}

			if !to.After(from) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "`to` must be after `from`",
				})
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		defer cancel()

		fc, err := svc.GetForecast(ctx, city, from, to)
		if err != nil {
			return mapServiceError(c, err)
		}

		return c.JSON(fc)
	})

	log.Info("starting server", "port", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Error("server failed", "error", err)
	}
}

func initProviders(cfg *config.Config) []weather.Provider {
	providers := []weather.Provider{
		weather.NewOpenMeteoProvider(),
	}

	if cfg.OpenWeatherMapAPIKey != "" {
		providers = append(providers,
			weather.NewOpenWeatherMapProvider(cfg.OpenWeatherMapAPIKey),
		)
	}

	if cfg.WeatherAPIKey != "" {
		providers = append(providers,
			weather.NewWeatherAPIComProvider(cfg.WeatherAPIKey),
		)
	}

	return providers
}

// mapServiceError converts domain/service errors to HTTP responses.
func mapServiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, weather.ErrCityNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "city not found",
		})
	case errors.Is(err, weather.ErrProviderUnavailable):
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "weather providers are unavailable",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})

	}
}
