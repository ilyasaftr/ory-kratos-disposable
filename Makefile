.PHONY: help build run clean docker-build docker-run docker-stop test fmt lint

# Default target
help:
	@echo "Available targets:"
	@echo "  build         - Build the application binary"
	@echo "  run           - Run the application locally"
	@echo "  clean         - Remove build artifacts"

# Build the application
build:
	@echo "Building application..."
	@go build -o bin/webhook ./cmd/server

# Run the application locally
run:
	@echo "Running application..."
	@go run ./cmd/server/main.go

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Create .env file from example
env:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env file created. Please update with your values."; \
	else \
		echo ".env file already exists."; \
	fi
