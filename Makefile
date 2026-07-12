# Variables
APP_NAME = auth-service
BUILD_DIR = bin
MAIN_PATH = cmd/server
GO = go
GOFLAGS = -ldflags="-w -s"
DOCKER_COMPOSE = docker-compose

.PHONY: all build run clean test lint docker-build docker-up docker-down help

all: build

## Build the binary
build:
	@echo "Building $(APP_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

## Run the application locally
run:
	@echo "Running $(APP_NAME)..."
	$(GO) run $(MAIN_PATH)

## Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

## Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Tests complete"

## Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -func=coverage.out

## Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

## Vendor dependencies
vendor:
	@echo "Vendoring dependencies..."
	$(GO) mod vendor

## Build Docker images
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest .

## Start all services with Docker Compose
docker-up:
	@echo "Starting services..."
	$(DOCKER_COMPOSE) up -d --build
	@echo "Services started. API available at http://localhost:8080"

## Stop all services
docker-down:
	@echo "Stopping services..."
	$(DOCKER_COMPOSE) down
	@echo "Services stopped"

## View service logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f api

## Run database migrations
migrate:
	@echo "Running migrations..."
	$(GO) run $(MAIN_PATH)

## Generate swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	swag init -g cmd/server/main.go -o docs

## Help
help:
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'