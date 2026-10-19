.PHONY: help build run stop test clean migrate-up migrate-down docker-up docker-down docker-logs seed docker-seed test-race

# Variables
# Prefer Compose v2 ("docker compose"); the old Python docker-compose 1.29 crashes with
# KeyError: 'ContainerConfig' on current Docker Engine versions when recreating containers.
DOCKER_COMPOSE ?= $(shell docker compose version >/dev/null 2>&1 && echo "docker compose" || echo docker-compose)
GO = go
MIGRATE = migrate

help:
	@echo "School Management System - Available Commands"
	@echo ""
	@echo "Development:"
	@echo "  make build              - Build the application"
	@echo "  make run                - Run the application (requires PostgreSQL)"
	@echo "  make test               - Run tests"
	@echo "  make test-race          - Run tests with the race detector"
	@echo "  make seed               - Fill an empty database with a demo school"
	@echo "  make clean              - Clean build artifacts"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up         - Run migrations up"
	@echo "  make migrate-down       - Run migrations down"
	@echo "  make migrate-create     - Create a new migration"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up          - Start Docker containers"
	@echo "  make docker-down        - Stop Docker containers"
	@echo "  make docker-logs        - View Docker logs"
	@echo "  make docker-clean       - Remove Docker containers and volumes"
	@echo "  make docker-seed        - Load the demo school into the Docker database"
	@echo ""

build:
	@echo "Building application..."
	$(GO) build -o bin/api cmd/api/main.go
	@echo "Build complete!"

run: build
	@echo "Starting application..."
	./bin/api

test:
	@echo "Running tests..."
	$(GO) test -v -cover ./...

test-race:
	$(GO) test -race ./...

# Demo data (only runs on an empty database)
seed:
	$(GO) run ./cmd/seed

clean:
	@echo "Cleaning up..."
	rm -rf bin/
	$(GO) clean

# Database migrations
migrate-up:
	@echo "Running migrations up..."
	$(MIGRATE) -path ./migrations -database "postgres://postgres:password@localhost:5432/school_management?sslmode=disable" up

migrate-down:
	@echo "Rolling back migrations..."
	$(MIGRATE) -path ./migrations -database "postgres://postgres:password@localhost:5432/school_management?sslmode=disable" down

migrate-create:
	@read -p "Enter migration name: " name; \
	$(MIGRATE) create -ext sql -dir ./migrations -seq $$name

# Docker commands
docker-up:
	@echo "Starting Docker containers..."
	$(DOCKER_COMPOSE) up -d
	@echo "Containers started!"
	@echo "App: http://localhost:8080  (API under /api)"
	@echo "PostgreSQL: localhost:5432"

docker-down:
	@echo "Stopping Docker containers..."
	$(DOCKER_COMPOSE) down

docker-logs:
	@echo "Docker logs:"
	$(DOCKER_COMPOSE) logs -f

docker-seed:
	$(DOCKER_COMPOSE) run --rm api ./seed

docker-clean:
	@echo "Removing Docker containers and volumes..."
	$(DOCKER_COMPOSE) down -v
	@echo "Clean complete!"

# Restart containers
docker-restart: docker-down docker-up

# Format and lint code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Default target
.DEFAULT_GOAL := help
