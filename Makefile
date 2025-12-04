# ────────────────────────────────────────────────────────────────
# Go Weather Aggregator — Makefile (Kubernetes/Go style)
# ────────────────────────────────────────────────────────────────

BINARY_NAME=weather
CMD=cmd/weather
GOFILES=$(shell find . -name '*.go' -not -path "./vendor/*")

.PHONY: all build run clean test fmt lint deps

## Default target
all: build

## Build binary
build:
	@echo "▶ Building..."
	@go build -o $(BINARY_NAME) ./$(CMD)

## Run application
run:
	@echo "▶ Running..."
	@go run ./$(CMD)

## Run tests
test:
	@echo "▶ Running tests..."
	@go test ./... -cover

## Format code
fmt:
	@echo "▶ Formatting..."
	@go fmt ./...

## Lint (requires golangci-lint)
lint:
	@echo "▶ Linting..."
	@golangci-lint run

## Download dependencies
deps:
	@echo "▶ Downloading dependencies..."
	@go mod tidy
	@go mod vendor

## Clean build artifacts
clean:
	@echo "▶ Cleaning..."
	@rm -f $(BINARY_NAME)

