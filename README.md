# **Weather Data Aggregator Service (Go + Fiber)**

*A concurrent weather aggregation service with scheduling, in-memory storage, multi-provider support, and graceful shutdown.*

---

## **📄 Table of Contents**

* [Overview](#overview)
* [Features](#features)
* [Architecture](#architecture)

    * [High-level Diagram](#high-level-diagram)
    * [Package Structure](#package-structure)
* [Configuration](#configuration)
* [Running the Application](#running-the-application)
* [HTTP API](#http-api)

    * [/health](#get-apiv1health)
    * [/weather/current](#get-apiv1weathercurrentcitycity)
    * [/weather/forecast](#get-apiv1weatherforecastcitycitydays1-7)
* [Implementation Notes](#implementation-notes)
* [Possible Extensions](#possible-extensions)

---

# **Overview**

This service periodically fetches weather information from multiple providers, aggregates results, stores them in memory, and exposes a clean HTTP API using Fiber.

It demonstrates:

* concurrent provider requests,
* background scheduling,
* normalization and aggregation of heterogeneous external data,
* structured logging (`log/slog`),
* graceful shutdown,
* multi-module architecture (`internal/...`).

The implementation corresponds to a real test assignment.

---

# **Features**

### ✔ Multi-provider architecture

* OpenMeteo (real HTTP client, no API key required)
* OpenWeatherMap (stub)
* WeatherAPI.com (stub)

### ✔ Concurrent fetching

Providers are queried in parallel for:

* current weather,
* multi-day forecast.

### ✔ Aggregation

* combines successful results,
* averages numeric data (temperature, humidity, wind speed),
* unifies timestamps.

### ✔ In-memory storage

* stores **latest current weather** per city,
* stores **latest forecast** for `{city, days}`,
* keeps limited **historical snapshots**,
* exposes **last fetch times**.

### ✔ Background scheduler

* runs every `FETCH_INTERVAL`,
* fetches weather for all default cities,
* avoids overlapping runs,
* logs each tick.

### ✔ JSON Logging (`log/slog`)

Structured logs for:

* configuration loading,
* provider calls,
* scheduler ticks,
* shutdown sequence.

### ✔ Graceful shutdown

Stops:

* HTTP server,
* scheduler goroutine(s),

after receiving OS signals (`SIGINT`, `SIGTERM`).

---

# **Architecture**

## **High-level diagram**

```
                 ┌─────────────────────────────────────────┐
                 │                 main.go                  │
                 │  - load config                          │
                 │  - init logger                          │
                 │  - init providers/storage/service       │
                 │  - start scheduler + Fiber server       │
                 └─────────────────────────────────────────┘
                                      │
                                      ▼
         ┌──────────────────────────────────────────┐
         │              Scheduler                   │
         │  time.Ticker → for each city →           │
         │      Service.GetCurrentWeather()         │
         │      Service.GetForecast()               │
         │  store results in InMemoryStore          │
         └──────────────────────────────────────────┘
                                      │
                                      ▼
           ┌────────────────────────────────────────┐
           │                Service                  │
           │  Providers queried concurrently:        │
           │      go p.FetchCurrent()                │
           │      go p.FetchForecast()               │
           │  collect results via channel            │
           │  aggregation logic                      │
           └────────────────────────────────────────┘
                                      │
                                      ▼
            ┌──────────────────────────────────────┐
            │              Providers               │
            │  OpenMeteoProvider (real client)     │
            │  OpenWeatherMapProvider (stub)       │
            │  WeatherAPIComProvider (stub)        │
            └──────────────────────────────────────┘
                                      │
                                      ▼
         ┌──────────────────────────────────────────┐
         │            InMemoryStore                 │
         │  - current[city] → CurrentWeather        │
         │  - forecast[{city,days}] → Forecast      │
         │  - history (snapshots)                   │
         │  - lastFetch[city]                       │
         └──────────────────────────────────────────┘
                                      │
                                      ▼
                    ┌──────────────────────┐
                    │        Fiber API     │
                    │ /health              │
                    │ /weather/current     │
                    │ /weather/forecast    │
                    └──────────────────────┘
```

---

# **Package Structure**

```
cmd/weather/
    main.go

internal/
    api/
        handlers.go
        routes.go

    config/
        config.go

    weather/
        models.go
        provider.go
        openmeteo.go
        openweathermap.go
        weatherapicom.go
        service.go
        aggregator.go
        normalizer.go

    storage/
        store.go

    scheduler/
        scheduler.go
```

---

# **Configuration**

All config is environment-based.
Example (`.env.example`):

```env
FIBER_PORT=3000
FETCH_INTERVAL=30s

OPENWEATHERMAP_API_KEY=
WEATHERAPI_API_KEY=

REQUEST_TIMEOUT=5s

DEFAULT_CITIES=London, Paris, Warsaw
```

Usage:

```bash
cp .env.example .env
```

---

# **Running the Application**

### **With Makefile (recommended)**

```bash
make build
make run
```

Press `Ctrl+C` to trigger graceful shutdown.

---

### **Direct Go run**

```bash
go run ./cmd/weather
```

---

# **HTTP API**

## **GET `/api/v1/health`**

Returns service status and configuration summary.

Example:

```json
{
  "status": "ok",
  "default_cities": ["London","Paris","Warsaw"],
  "fetch_interval": "30s",
  "request_timeout": "5s",
  "openweathermap_key": true,
  "weatherapi_key": true,
  "last_fetch": {
    "london": "2025-12-09T10:18:51Z"
  }
}
```

---

## **GET `/api/v1/weather/current?city={city}`**

### Responses

* `200` — aggregated current weather
* `400` — missing `city`
* `404` — no providers returned city
* `503` — provider failure

Example:

```bash
curl "http://localhost:3000/api/v1/weather/current?city=London"
```

---

## **GET `/api/v1/weather/forecast?city={city}&days=1-7`**

### Parameters

* `city` — required
* `days` — integer `1..7`

Example:

```bash
curl "http://localhost:3000/api/v1/weather/forecast?city=London&days=3"
```

---

# **Implementation Notes**

* Providers run concurrently per request using goroutines + buffered channels.
* Aggregator merges multiple provider responses into a unified domain model.
* Scheduler uses:

    * `time.Ticker`,
    * per-tick `sync.WaitGroup`,
    * context for cancellation.
* Storage uses `sync.RWMutex` and keeps:

    * live data,
    * history snapshots,
    * timestamps of last successful fetch.
* Logging is fully structured JSON using `log/slog`.

---

# **Possible Extensions**

* Implement real HTTP clients for:

    * OpenWeatherMap,
    * WeatherAPI.com
* Add Prometheus metrics.
* Add integration tests using httptest.
* Implement caching at provider level.
* Add rate-limiters, circuit breakers, retries with exponential backoff.
* Package the service with Docker/Compose.

---
