.PHONY: help build up down restart logs logs-all test clean migrate-up migrate-down ps

# Variables
DOCKER_COMPOSE = docker-compose
APP_CONTAINER = pr-review-assigner-app
DB_CONTAINER = pr-review-assigner-db

# Colors for output
GREEN = \033[0;32m
NC = \033[0m # No Color

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
	@echo "$(GREEN)Building Docker images...$(NC)"
	$(DOCKER_COMPOSE) build

## up: Start all services (DB, migrations, app, Swagger UI)
up:
	@echo "$(GREEN)Starting services...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)Services started!$(NC)"
	@echo "API available at: http://localhost:8080"
	@echo "Swagger UI available at: http://localhost:8081"

## down: Stop all services
down:
	@echo "$(GREEN)Stopping services...$(NC)"
	$(DOCKER_COMPOSE) down

## restart: Restart services
restart: down up

## logs: Show application logs (follow new logs)
logs:
	$(DOCKER_COMPOSE) logs -f $(APP_CONTAINER)

## logs-all: Show logs from all services
logs-all:
	$(DOCKER_COMPOSE) logs -f

## test: Run unit tests locally
test:
	@echo "$(GREEN)Running unit tests...$(NC)"
	go test ./internal/service/... -v -race -coverprofile=coverage.out
	@echo "$(GREEN)Code coverage:$(NC)"
	go tool cover -func=coverage.out | grep total

## clean: Full cleanup (containers, images, volumes)
clean:
	@echo "$(GREEN)Full cleanup...$(NC)"
	$(DOCKER_COMPOSE) down -v --rmi all --remove-orphans
	@echo "$(GREEN)Cleanup completed$(NC)"

## migrate-up: Apply migrations manually
migrate-up:
	@echo "$(GREEN)Applying migrations...$(NC)"
	$(DOCKER_COMPOSE) run --rm migrate -path=/migrations -database=postgres://pr_reviewer:pr_reviewer_pass@postgres:5432/pr_review_assigner?sslmode=disable up

## migrate-down: Rollback last migration
migrate-down:
	@echo "$(GREEN)Rolling back migration...$(NC)"
	$(DOCKER_COMPOSE) run --rm migrate -path=/migrations -database=postgres://pr_reviewer:pr_reviewer_pass@postgres:5432/pr_review_assigner?sslmode=disable down 1

## ps: Show container status
ps:
	$(DOCKER_COMPOSE) ps


