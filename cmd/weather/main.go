package main

import (
	"log/slog"
	"os"

	"github.com/andrqxa/weather-aggregator/internal/config"
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
		"default_cities", cfg.DefaultCities,
		"api_key_set", cfg.WeatherAPIKey != "",
	)

	// Fiber init
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
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
			"status":         "ok",
			"default_cities": cfg.DefaultCities,
			"fetch_interval": cfg.FetchInterval.String(),
			"api_key_set":    cfg.WeatherAPIKey != "",
		})
	})

	log.Info("starting server", "port", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Error("server failed", "error", err)
	}
}
