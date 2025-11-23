.PHONY: help build up down restart logs logs-all test clean migrate-up migrate-down ps

# Variables
DOCKER_COMPOSE = docker-compose
APP_SERVICE = app
DB_SERVICE = postgres

## help: Show help for all available commands
help:
	@echo "Available commands:"
	@echo "  make build          - Build Docker images"
	@echo "  make up             - Start all services"
	@echo "  make down           - Stop all services"
	@echo "  make restart        - Restart services"
	@echo "  make logs           - Show application logs"
	@echo "  make logs-all       - Show logs from all services"
	@echo "  make test           - Run unit tests locally"
	@echo "  make clean          - Full cleanup (containers, images, volumes)"
	@echo "  make migrate-up     - Apply migrations manually"
	@echo "  make migrate-down   - Rollback migrations"
	@echo "  make ps             - Show container status"

## build: Build Docker images
build:
	@echo "Building Docker images..."
	$(DOCKER_COMPOSE) build

## up: Start all services (DB, migrations, app, Swagger UI)
up:
	@echo "Starting services..."
	$(DOCKER_COMPOSE) up -d
	@echo "Services started!"
	@echo "API available at: http://localhost:8080"
	@echo "Swagger UI available at: http://localhost:8081"

## down: Stop all services
down:
	@echo "Stopping services..."
	$(DOCKER_COMPOSE) down

## restart: Restart services
restart: down up

## logs: Show application logs (follow new logs)
logs:
	$(DOCKER_COMPOSE) logs -f $(APP_SERVICE)

## logs-all: Show logs from all services
logs-all:
	$(DOCKER_COMPOSE) logs -f

## test: Run unit tests locally
test:
	@echo "Running unit tests..."
	go test ./internal/service/... -v -coverprofile=coverage.out
	@echo "Code coverage:"
	go tool cover -func=coverage.out | findstr total

## clean: Full cleanup (containers, images, volumes)
clean:
	@echo "Full cleanup..."
	$(DOCKER_COMPOSE) down -v --rmi all --remove-orphans
	@echo "Cleanup completed"

## migrate-up: Apply migrations manually
migrate-up:
	@echo "Applying migrations..."
	$(DOCKER_COMPOSE) run --rm migrate -path=/migrations -database=postgres://pr_reviewer:pr_reviewer_pass@postgres:5432/pr_review_assigner?sslmode=disable up

## migrate-down: Rollback last migration
migrate-down:
	@echo "Rolling back migration..."
	$(DOCKER_COMPOSE) run --rm migrate -path=/migrations -database=postgres://pr_reviewer:pr_reviewer_pass@postgres:5432/pr_review_assigner?sslmode=disable down 1

## ps: Show container status
ps:
	$(DOCKER_COMPOSE) ps


