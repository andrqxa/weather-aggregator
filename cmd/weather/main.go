package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/andrqxa/weather-aggregator/internal/config"
	"github.com/andrqxa/weather-aggregator/internal/scheduler"
	"github.com/andrqxa/weather-aggregator/internal/storage"
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

	// Init storage
	store := storage.NewInMemoryStore()

	log.Info("configuration loaded",
		"port", cfg.Port,
		"fetch_interval", cfg.FetchInterval.String(),
		"openweathermap_key_set", cfg.OpenWeatherMapAPIKey != "",
		"weatherapi_key_set", cfg.WeatherAPIKey != "",
		"request_timeout", cfg.RequestTimeout.String(),
		"default_cities", cfg.DefaultCities,
	)

	// Root context with OS signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	// Initialize weather providers and service
	providers := initProviders(cfg)
	svc := weather.NewService(providers)

	// Initialize scheduler (e.g. 1-day forecast by default).
	const defaultForecastDays = 1

	sched := scheduler.NewScheduler(
		svc,
		store,
		cfg.DefaultCities,
		cfg.FetchInterval,
		cfg.RequestTimeout,
		defaultForecastDays,
		log,
	)

	// Start scheduler in background.
	go sched.Start(ctx)

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
			"last_fetch":         store.LastFetchTimes(),
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

		// Try cache first
		if cw, ok := store.GetCurrent(city); ok {
			return c.JSON(cw)
		}

		ctxReq, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		defer cancel()

		w, err := svc.GetCurrentWeather(ctxReq, city)
		if err != nil {
			return mapServiceError(c, err)
		}

		// Save to storage with current time as fetch timestamp
		store.SaveCurrent(city, w, time.Now().UTC())

		return c.JSON(w)
	})

	// GET /api/v1/weather/forecast?city=London&days=1
	weatherGroup.Get("/forecast", func(c *fiber.Ctx) error {
		city := c.Query("city")
		if city == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "city query parameter is required",
			})
		}

		rawDays := c.Query("days")

		if rawDays == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "days query parameter is required",
			})
		}

		days, err := strconv.Atoi(rawDays)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid days parameter, expected integer",
			})
		}
		if days < 1 || days > 7 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "days parameter must be in the 1 - 7 limit",
			})
		}

		// Try cache first
		if fc, ok := store.GetForecast(city, days); ok {
			return c.JSON(fc)
		}

		ctxReq, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		defer cancel()

		fc, err := svc.GetForecast(ctxReq, city, days)
		if err != nil {
			return mapServiceError(c, err)
		}

		store.SaveForecast(city, days, fc, time.Now().UTC())

		return c.JSON(fc)
	})

	// Run Fiber server in background
	go func() {
		log.Info("starting server", "port", cfg.Port)
		if err := app.Listen(":" + cfg.Port); err != nil {
			if ctx.Err() == nil {
				log.Error("server failed", "error", err)
			} else {
				log.Info("server stopped", "reason", "context canceled")
			}
		}
	}()

	// Wait for termination signal
	<-ctx.Done()
	log.Info("shutdown signal received")

	// Stop Fiber gracefully
	if err := app.Shutdown(); err != nil {
		log.Error("failed to shutdown server", "error", err)
	} else {
		log.Info("server gracefully stopped")
	}

	// Scheduler сам завершится по ctx.Done()
	log.Info("scheduler stopped")
}

func initProviders(cfg *config.Config) []weather.Provider {
	httpClient := &http.Client{
		Timeout: cfg.RequestTimeout,
	}

	providers := []weather.Provider{
		weather.NewOpenMeteoProvider(httpClient),
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
