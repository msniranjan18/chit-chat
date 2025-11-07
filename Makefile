# Version: 1.0.0

# ==============================================================================
# Configuration
# ==============================================================================

# Project Information
PROJECT_NAME := chitchat
VERSION := 1.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

# Go Configuration
GO := go
GO_VERSION := $(shell go version | awk '{print $$3}')
GO_MODULE := github.com/msniranjan18/chit-chat
GO_PACKAGES := $(shell go list ./...)
GO_TEST_FLAGS := -v -race -cover -timeout 2m
GO_BUILD_FLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT) -X main.date=$(BUILD_TIME)"
GO_LDFLAGS := -ldflags="-s -w"

# Directories
BIN_DIR := bin
DIST_DIR := dist
COVERAGE_DIR := coverage
MIGRATIONS_DIR := migrations
LOGS_DIR := logs
DOCKER_DIR := .

# Files
BINARY := $(BIN_DIR)/$(PROJECT_NAME)
ENV_FILE := .env
ENV_EXAMPLE := .env.example

# Docker Configuration
DOCKER := docker
DOCKER_COMPOSE := docker-compose -f hack/docker-compose.yaml
DOCKER_IMAGE_NAME := $(PROJECT_NAME)
DOCKER_TAG := latest
DOCKER_REGISTRY :=

# PostgreSQL Configuration
POSTGRES_DB := chitchat
POSTGRES_USER := postgres
POSTGRES_PASSWORD := password
POSTGRES_HOST := localhost
POSTGRES_PORT := 5432

# Redis Configuration
REDIS_HOST := localhost
REDIS_PORT := 6379

# ==============================================================================
# Help
# ==============================================================================

.PHONY: help
help: ## Display this help message
	@echo "ChitChat - Messaging Application"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ==============================================================================
# Dependencies
# ==============================================================================

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@$(GO) mod download
	@$(GO) mod verify
	@echo "Dependencies downloaded."

.PHONY: deps-update
deps-update: ## Update all dependencies
	@echo "Updating dependencies..."
	@$(GO) get -u ./...
	@$(GO) mod tidy
	@echo "Dependencies updated."

.PHONY: deps-clean
deps-clean: ## Clean dependencies cache
	@echo "Cleaning dependencies cache..."
	@$(GO) clean -modcache
	@echo "Dependencies cache cleaned."

.PHONY: tidy
tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	@$(GO) mod tidy
	@echo "go.mod tidied."

# ==============================================================================
# Build
# ==============================================================================

.PHONY: build
build: clean deps deps-update  ## Build the application
	@echo "Building $(PROJECT_NAME) v$(VERSION)..."
	@mkdir -p $(BIN_DIR)
	@$(GO) build $(GO_BUILD_FLAGS) -o $(BINARY) ./cmd/main.go
	@echo "Build complete: $(BINARY)"

# ==============================================================================
# Testing
# ==============================================================================

.PHONY: test
test: ## Run unit tests
	@echo "Running tests..."
	@$(GO) test $(GO_TEST_FLAGS) ./...
	@echo "Tests passed."

# ==============================================================================
# Docker
# ==============================================================================

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@$(DOCKER) build -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .
	@echo "Docker image built: $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)"

.PHONY: docker-push
docker-push: docker-build ## Push Docker image to registry
ifndef DOCKER_REGISTRY
	@echo "Error: DOCKER_REGISTRY not set"
	@exit 1
endif
	@echo "Pushing Docker image..."
	@$(DOCKER) tag $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG)
	@$(DOCKER) push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG)
	@echo "Docker image pushed."

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@$(DOCKER) run -p 8080:8080 --name $(PROJECT_NAME) $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)

.PHONY: docker-compose-build
docker-compose-build: ## Build app service with Docker Compose
	@echo "Building app service without Docker cache..."
	@$(DOCKER_COMPOSE) build app
	@echo "Docker Compose build completed for app."

.PHONY: docker-compose-build-nocache
docker-compose-build-nocache: ## Build app service with Docker Compose without cache
	@echo "Building app service without Docker cache..."
	@$(DOCKER_COMPOSE) build --no-cache app
	@echo "Docker Compose build (no cache) completed for app."

.PHONY: docker-compose-up
docker-compose-up: docker-compose-build-nocache ## Start all services with Docker Compose
	@echo "Starting services with Docker Compose..."
	@$(DOCKER_COMPOSE) up -d
	@echo "Services started. Visit http://localhost:8080"

.PHONY: docker-compose-down
docker-compose-down: ## Stop all services with Docker Compose
	@echo "Stopping services..."
	@$(DOCKER_COMPOSE) down
	@echo "Services stopped."

.PHONY: docker-compose-docker-volume-delete
docker-compose-docker-volume-delete: ## Stop all services with Docker Compose
	@echo "Stopping services..."
	@$(DOCKER_COMPOSE) down -v
	@echo "Services stopped."

.PHONY: docker-compose-logs
docker-compose-logs: ## View Docker Compose logs
	@$(DOCKER_COMPOSE) logs -f

.PHONY: docker-compose-logs-app
docker-compose-logs-app: ## View Docker Compose logs
	@$(DOCKER_COMPOSE) logs -f app

.PHONY: docker-compose-logs-postgres
docker-compose-logs-postgres: ## View Docker Compose logs
	@$(DOCKER_COMPOSE) logs -f postgres

.PHONY: docker-compose-logs-redis
docker-compose-logs-redis: ## View Docker Compose logs
	@$(DOCKER_COMPOSE) logs -f redis

.PHONY: docker-compose-ps
docker-ps:
	@$(DOCKER_COMPOSE) ps

.PHONY: docker-compose-restart
docker-compose-restart: docker-compose-down docker-compose-build-nocache docker-compose-up ## Restart all services

# ==============================================================================
# Linting & Formatting
# ==============================================================================

.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	@if [ -x "$$(command -v golangci-lint)" ]; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting Go code..."
	@$(GO) fmt ./...
	@echo "Code formatted."

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@$(GO) vet ./...
	@echo "Vet completed."

# ==============================================================================
# Cleanup
# ==============================================================================

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(DIST_DIR) $(COVERAGE_DIR) $(LOGS_DIR)
	@$(GO) clean
	@echo "Cleanup complete."

.PHONY: clean-all
clean-all: clean deps-clean ## Clean everything including dependencies
	@echo "Cleaning everything..."
	@$(DOCKER) system prune -f
	@echo "Complete cleanup done."

# ==============================================================================
# Security
# ==============================================================================

.PHONY: security-scan
security-scan: ## Run security scan
	@echo "Running security scan..."
	@if [ -x "$$(command -v gosec)" ]; then \
		gosec ./...; \
	else \
		echo "gosec not found, installing..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

.PHONY: audit
audit: ## Audit dependencies
	@echo "Auditing dependencies..."
	@$(GO) list -m all | tail -n +2 | awk '{print $$1}' | xargs -n 1 $(GO) mod why

# ==============================================================================
# Documentation
# ==============================================================================

.PHONY: swagger
swagger: ## Generate Swagger / OpenAPI documentation
	@echo "Generating Swagger documentation in container..."
	@docker run --rm \
		-v $$(pwd):/app \
		-w /app \
		golang:1.24 \
		sh -c "\
			go install github.com/swaggo/swag/cmd/swag@latest && \
			swag init -g cmd/main.go -o docs && \
			go mod tidy \
		"
	@echo "Swagger documentation generated in docs/"
	@echo "Access it at: http://localhost:8080/swagger/index.html"

# ==============================================================================
# Development Tools
# ==============================================================================

.PHONY: tools
tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/cespare/reflex@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Tools installed."

# ==============================================================================
# Debugging
# ==============================================================================

.PHONY: pprof
pprof:
	@echo "Opening pprof UI at http://localhost:7070"
	@docker run --rm -it \
		-p 7070:7070 \
		golang:1.24 \
		go tool pprof -http=:7070 http://host.docker.internal:8080/debug/pprof/profile



.PHONY: trace
trace:
	@echo "Collecting trace..."
	@curl -o trace.out http://localhost:8080/debug/pprof/trace?seconds=5
	@docker run --rm -it \
		-p 8081:8081 \
		-v $$(pwd):/app \
		-w /app \
		golang:1.24 \
		go tool trace -http=:8081 trace.out

.PHONY: jwt-secret
jwt-secret: ## Generate a secure JWT secret and copy to clipboard
	@echo "Generating JWT secret..."
	@SECRET=$$(openssl rand -base64 32); \
	if command -v pbcopy >/dev/null 2>&1; then \
		echo "$$SECRET" | pbcopy; \
		echo "Generated JWT_SECRET is copied to clipboard using pbcopy"; \
	elif command -v xclip >/dev/null 2>&1; then \
		echo "$$SECRET" | xclip -selection clipboard; \
		echo "Generated JWT_SECRET is copied to clipboard using xclip"; \
	else \
		echo "Clipboard tool not found. Please copy manually."; \
	fi; \
	echo ""; \
	echo "Add this to your .env file by using paste command"; \
	echo "JWT_SECRET=$$SECRET"

# ==============================================================================
# Redis Utilities
# ==============================================================================

.PHONY: redis-cli
redis-cli: ## Open Redis CLI inside the Redis container
	@echo "Opening Redis CLI..."
	@$(DOCKER_COMPOSE) exec redis redis-cli

.PHONY: redis-ping
redis-ping: ## Ping Redis to check if it is alive
	@echo "Pinging Redis..."
	@$(DOCKER_COMPOSE) exec redis redis-cli ping

.PHONY: redis-info
redis-info: ## Show Redis server info
	@echo "Fetching Redis info..."
	@$(DOCKER_COMPOSE) exec redis redis-cli info

.PHONY: redis-keys
redis-keys: ## List all Redis keys (use carefully in production)
	@echo "Listing Redis keys..."
	@$(DOCKER_COMPOSE) exec redis redis-cli keys '*'

.PHONY: redis-flush
redis-flush: ## Flush all Redis data (DANGEROUS)
	@echo "Flushing ALL Redis data..."
	@$(DOCKER_COMPOSE) exec redis redis-cli FLUSHALL
	@echo "Redis data cleared."

.PHONY: redis-memory
redis-memory: ## Show Redis memory usage
	@echo "Redis memory usage:"
	@$(DOCKER_COMPOSE) exec redis redis-cli info memory

.PHONY: redis-stats
redis-stats: ## Show Redis stats
	@echo "Redis statistics:"
	@$(DOCKER_COMPOSE) exec redis redis-cli info stats

.PHONY: redis-monitor
redis-monitor: ## Monitor Redis commands in real-time (DEBUG use only)
	@echo "Starting Redis MONITOR (Ctrl+C to stop)..."
	@$(DOCKER_COMPOSE) exec redis redis-cli monitor

# ==============================================================================
# PostgreSQL Utilities
# ==============================================================================

.PHONY: pg-psql
pg-psql: ## Open psql shell inside PostgreSQL container
	@echo "Opening PostgreSQL shell..."
	@$(DOCKER_COMPOSE) exec postgres psql -U usr-chitchat -d chitchat

.PHONY: pg-status
pg-status: ## Check PostgreSQL container health
	@echo "Checking PostgreSQL health..."
	@docker inspect --format='{{.State.Health.Status}}' \
		$$(docker compose ps -q postgres)

.PHONY: pg-tables
pg-tables: ## List all tables in the database
	@echo "Listing tables in chitchat DB..."
	@$(DOCKER_COMPOSE) exec postgres \
		psql -U usr-chitchat -d chitchat -c "\dt"

.PHONY: pg-describe
pg-describe: ## Describe a table (usage: make pg-describe TABLE=table_name)
	@if [ -z "$(TABLE)" ]; then \
		echo "Usage: make pg-describe TABLE=table_name"; \
		exit 1; \
	fi
	@$(DOCKER_COMPOSE) exec postgres \
		psql -U usr-chitchat -d chitchat -c "\d $(TABLE)"

.PHONY: pg-count
pg-count: ## Count rows in a table (usage: make pg-count TABLE=table_name)
	@if [ -z "$(TABLE)" ]; then \
		echo "Usage: make pg-count TABLE=table_name"; \
		exit 1; \
	fi
	@$(DOCKER_COMPOSE) exec postgres \
		psql -U usr-chitchat -d chitchat -c "SELECT COUNT(*) FROM $(TABLE);"

.PHONY: pg-select
pg-select: ## View data from a table (usage: make pg-select TABLE=table_name LIMIT=10)
	@if [ -z "$(TABLE)" ]; then \
		echo "Usage: make pg-select TABLE=table_name [LIMIT=10]"; \
		exit 1; \
	fi
	@$(DOCKER_COMPOSE) exec postgres \
		psql -U usr-chitchat -d chitchat -c "SELECT * FROM $(TABLE) LIMIT $${LIMIT:-10};"

.PHONY: pg-size
pg-size: ## Show database size
	@$(DOCKER_COMPOSE) exec postgres \
		psql -U usr-chitchat -d chitchat -c \
		"SELECT pg_size_pretty(pg_database_size('chitchat'));"

.PHONY: pg-connections
pg-connections: ## Show active connections
	@$(DOCKER_COMPOSE) exec postgres \
		psql -U usr-chitchat -d chitchat -c \
		"SELECT pid, usename, state, query FROM pg_stat_activity WHERE datname='chitchat';"

.PHONY: pg-flush
pg-flush: ## Drop and recreate database (DANGEROUS)
	@echo "Dropping and recreating database chitchat..."
	@$(DOCKER_COMPOSE) exec postgres \
		psql -U usr-chitchat -c "DROP DATABASE IF EXISTS chitchat;"
	@$(DOCKER_COMPOSE) exec postgres \
		psql -U usr-chitchat -c "CREATE DATABASE chitchat;"
	@echo "Database reset completed."


# ==============================================================================
# Default target
# ==============================================================================

.DEFAULT_GOAL := help
