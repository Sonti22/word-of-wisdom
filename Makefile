.PHONY: help all build build-windows run-server run-client test test-short test-integration \
        test-coverage fmt lint clean docker-build docker-up docker-down up down logs \
        install-deps

# Default target
all: fmt lint test build ## Run fmt, lint, test, and build

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ============================================================================
# Build targets
# ============================================================================

build: ## Build server and client binaries (Linux/macOS)
	@echo "Building binaries..."
	@mkdir -p bin
	go build -ldflags="-w -s" -o bin/server ./cmd/server
	go build -ldflags="-w -s" -o bin/client ./cmd/client
	@echo "✓ Build complete: bin/server, bin/client"

build-windows: ## Build server and client binaries (Windows)
	@echo "Building Windows binaries..."
	@mkdir -p bin
	go build -ldflags="-w -s" -o bin/server.exe ./cmd/server
	go build -ldflags="-w -s" -o bin/client.exe ./cmd/client
	@echo "✓ Build complete: bin/server.exe, bin/client.exe"

install-deps: ## Download Go dependencies
	go mod download
	go mod tidy

# ============================================================================
# Run targets
# ============================================================================

run-server: ## Run the server (set WOW_ADDR, WOW_BITS, WOW_EXPIRES as needed)
	@echo "Starting server..."
	go run ./cmd/server

run-client: ## Run the client (set WOW_ADDR/SERVER_ADDR as needed)
	@echo "Starting client..."
	go run ./cmd/client

# ============================================================================
# Test targets
# ============================================================================

test: ## Run all tests with race detector
	@echo "Running tests..."
	go test -race -v ./...

test-short: ## Run only unit tests (skip integration tests)
	@echo "Running unit tests..."
	go test -short -race -v ./...

test-integration: ## Run only integration tests
	@echo "Running integration tests..."
	go test -v ./tests/

test-coverage: ## Run tests with coverage report
	@echo "Generating coverage report..."
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

# ============================================================================
# Code quality targets
# ============================================================================

fmt: ## Format Go code
	@echo "Formatting code..."
	@gofmt -l -s -w .
	@echo "✓ Code formatted"

lint: ## Run code quality checks (gofmt, go vet)
	@echo "Running linters..."
	@echo "  - gofmt..."
	@test -z "$$(gofmt -l -s .)" || (echo "Code not formatted. Run 'make fmt'" && exit 1)
	@echo "  - go vet..."
	@go vet ./...
	@echo "✓ Lint passed"

# ============================================================================
# Docker targets
# ============================================================================

docker-build: ## Build Docker images for server and client
	@echo "Building Docker images..."
	docker build -f Dockerfile.server -t word-of-wisdom-server:latest .
	docker build -f Dockerfile.client -t word-of-wisdom-client:latest .
	@echo "✓ Docker images built"

up: docker-up ## Alias for docker-up

docker-up: ## Start server and client via docker-compose
	@echo "Starting services via docker-compose..."
	docker-compose up --build

docker-up-d: ## Start services in background
	@echo "Starting services in background..."
	docker-compose up -d --build

down: docker-down ## Alias for docker-down

docker-down: ## Stop docker-compose services
	@echo "Stopping services..."
	docker-compose down

logs: ## Show docker-compose logs
	docker-compose logs -f

# ============================================================================
# Cleanup targets
# ============================================================================

clean: ## Remove build artifacts and coverage reports
	@echo "Cleaning up..."
	rm -rf bin/ coverage.out coverage.html server.log
	@echo "✓ Clean complete"

clean-docker: ## Remove Docker containers, images, and volumes
	@echo "Cleaning Docker resources..."
	docker-compose down -v --rmi all --remove-orphans
	@echo "✓ Docker resources cleaned"

# ============================================================================
# Utility targets
# ============================================================================

check: fmt lint test ## Run all checks (fmt, lint, test)

ci: lint test ## Run CI checks (lint + test)

.DEFAULT_GOAL := help

