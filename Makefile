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
DOCKER_COMPOSE_FILE := docker-compose.yml

# Docker Configuration
DOCKER := docker
DOCKER_COMPOSE := docker-compose
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
# Development
# ==============================================================================

.PHONY: dev
dev: clean deps env ## Start development server with hot reload
	@echo "Starting development server..."
	@if [ -x "$$(command -v air)" ]; then \
		air; \
	else \
		echo "Air not found, installing..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

.PHONY: run
run: clean build ## Run the application locally
	@echo "Running $(PROJECT_NAME)..."
	@./$(BINARY)

.PHONY: watch
watch: ## Watch for changes and rebuild
	@echo "Watching for changes..."
	@if [ -x "$$(command -v reflex)" ]; then \
		reflex -r '\.go$$' -- go run main.go; \
	else \
		echo "Reflex not found, installing..."; \
		go install github.com/cespare/reflex@latest; \
		reflex -r '\.go$$' -- go run main.go; \
	fi

# ==============================================================================
# Build
# ==============================================================================

.PHONY: build
build: clean deps ## Build the application
	@echo "Building $(PROJECT_NAME) v$(VERSION)..."
	@mkdir -p $(BIN_DIR)
	@$(GO) build $(GO_BUILD_FLAGS) -o $(BINARY) main.go
	@echo "Build complete: $(BINARY)"
	@$(BINARY) --version

.PHONY: build-all
build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	@$(GO) mod tidy
	@for os in darwin linux windows; do \
		for arch in amd64 arm64; do \
			echo "Building for $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch $(GO) build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(PROJECT_NAME)-$$os-$$arch main.go; \
			if [ "$$os" = "windows" ]; then \
				mv $(DIST_DIR)/$(PROJECT_NAME)-$$os-$$arch $(DIST_DIR)/$(PROJECT_NAME)-$$os-$$arch.exe; \
			fi; \
		done; \
	done
	@echo "Build complete. Output in $(DIST_DIR)"

.PHONY: install
install: build ## Install the application
	@echo "Installing $(PROJECT_NAME)..."
	@sudo cp $(BINARY) /usr/local/bin/$(PROJECT_NAME)
	@echo "Installation complete. Run '$(PROJECT_NAME)' to start."

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

# ==============================================================================
# Testing
# ==============================================================================

.PHONY: test
test: deps ## Run all tests
	@echo "Running tests..."
	@mkdir -p $(COVERAGE_DIR)
	@$(GO) test $(GO_TEST_FLAGS) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out

.PHONY: test-unit
test-unit: deps ## Run unit tests only
	@echo "Running unit tests..."
	@$(GO) test $(GO_TEST_FLAGS) ./pkg/...

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@$(GO) test $(GO_TEST_FLAGS) -tags=integration ./...

.PHONY: test-coverage
test-coverage: test ## Generate test coverage report
	@echo "Generating coverage report..."
	@$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

.PHONY: test-benchmark
test-benchmark: deps ## Run benchmarks
	@echo "Running benchmarks..."
	@$(GO) test -bench=. -benchmem ./...

.PHONY: test-race
test-race: deps ## Run tests with race detector
	@echo "Running tests with race detector..."
	@$(GO) test -race ./...

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

.PHONY: tidy
tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	@$(GO) mod tidy
	@echo "go.mod tidied."

# ==============================================================================
# Database
# ==============================================================================

.PHONY: db-start
db-start: ## Start PostgreSQL and Redis
	@echo "Starting databases..."
	@docker-compose up -d postgres redis
	@sleep 5
	@echo "Databases started."

.PHONY: db-stop
db-stop: ## Stop PostgreSQL and Redis
	@echo "Stopping databases..."
	@docker-compose stop postgres redis
	@echo "Databases stopped."

.PHONY: db-reset
db-reset: db-stop db-start ## Reset databases
	@echo "Databases reset."

.PHONY: db-psql
db-psql: ## Connect to PostgreSQL
	@echo "Connecting to PostgreSQL..."
	@PGPASSWORD=$(POSTGRES_PASSWORD) psql -h $(POSTGRES_HOST) -p $(POSTGRES_PORT) -U $(POSTGRES_USER) -d $(POSTGRES_DB)

.PHONY: db-redis
db-redis: ## Connect to Redis
	@echo "Connecting to Redis..."
	@redis-cli -h $(REDIS_HOST) -p $(REDIS_PORT)

.PHONY: db-migrate
db-migrate: ## Run database migrations
	@echo "Running database migrations..."
	@$(GO) run main.go
	@echo "Migrations completed."

.PHONY: db-seed
db-seed: ## Seed database with sample data
	@echo "Seeding database..."
	@# Add seed commands here
	@echo "Database seeded."

# ==============================================================================
# Docker
# ==============================================================================

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@$(DOCKER) build -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .
	@echo "Docker image built: $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)"

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@$(DOCKER) run -p 8080:8080 --name $(PROJECT_NAME) $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)

.PHONY: docker-compose-up
docker-compose-up: ## Start all services with Docker Compose
	@echo "Starting services with Docker Compose..."
	@$(DOCKER_COMPOSE) up -d
	@echo "Services started. Visit http://localhost:8080"

.PHONY: docker-compose-down
docker-compose-down: ## Stop all services with Docker Compose
	@echo "Stopping services..."
	@$(DOCKER_COMPOSE) down
	@echo "Services stopped."

.PHONY: docker-compose-logs
docker-compose-logs: ## View Docker Compose logs
	@$(DOCKER_COMPOSE) logs -f

.PHONY: docker-compose-restart
docker-compose-restart: docker-compose-down docker-compose-up ## Restart all services

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

# ==============================================================================
# Environment
# ==============================================================================

.PHONY: env
env: ## Setup environment variables
	@echo "Setting up environment..."
	@if [ ! -f $(ENV_FILE) ]; then \
		echo "Creating $(ENV_FILE) from $(ENV_EXAMPLE)..."; \
		cp $(ENV_EXAMPLE) $(ENV_FILE); \
		echo "Please edit $(ENV_FILE) with your configuration."; \
	else \
		echo "$(ENV_FILE) already exists."; \
	fi

.PHONY: env-check
env-check: ## Check environment variables
	@echo "Checking environment variables..."
	@if [ -f $(ENV_FILE) ]; then \
		echo "Loading $(ENV_FILE)..."; \
		export $$(grep -v '^#' $(ENV_FILE) | xargs); \
		echo "Environment variables loaded."; \
	else \
		echo "$(ENV_FILE) not found. Run 'make env' to create it."; \
	fi

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
	@rm -rf vendor
	@$(DOCKER) system prune -f
	@echo "Complete cleanup done."

# ==============================================================================
# Monitoring
# ==============================================================================

.PHONY: logs
logs: ## View application logs
	@echo "Viewing logs..."
	@mkdir -p $(LOGS_DIR)
	@if [ -f $(LOGS_DIR)/app.log ]; then \
		tail -f $(LOGS_DIR)/app.log; \
	else \
		echo "No log file found. Starting logging..."; \
		$(GO) run main.go 2>&1 | tee $(LOGS_DIR)/app.log; \
	fi

.PHONY: monitor
monitor: ## Monitor system resources
	@echo "Monitoring system resources..."
	@echo "=== ChitChat Processes ==="
	@ps aux | grep -E "$(PROJECT_NAME)|chitchat" | grep -v grep
	@echo ""
	@echo "=== Memory Usage ==="
	@free -h
	@echo ""
	@echo "=== Disk Usage ==="
	@df -h .
	@echo ""
	@echo "=== Network Connections ==="
	@netstat -tulpn | grep -E "8080|5432|6379" || true

.PHONY: health
health: ## Check system health
	@echo "Checking system health..."
	@echo "1. Checking PostgreSQL..."
	@if pg_isready -h $(POSTGRES_HOST) -p $(POSTGRES_PORT) >/dev/null 2>&1; then \
		echo "   ✓ PostgreSQL is running"; \
	else \
		echo "   ✗ PostgreSQL is not running"; \
	fi
	@echo "2. Checking Redis..."
	@if redis-cli -h $(REDIS_HOST) -p $(REDIS_PORT) ping >/dev/null 2>&1; then \
		echo "   ✓ Redis is running"; \
	else \
		echo "   ✗ Redis is not running"; \
	fi
	@echo "3. Checking application..."
	@if curl -s http://localhost:8080/health >/dev/null 2>&1; then \
		echo "   ✓ Application is running"; \
	else \
		echo "   ✗ Application is not running"; \
	fi

# ==============================================================================
# Release
# ==============================================================================

.PHONY: release
release: clean test lint build-all ## Create a release
	@echo "Creating release v$(VERSION)..."
	@mkdir -p release
	@cp $(DIST_DIR)/* release/
	@cp README.md LICENSE release/
	@tar -czf release/$(PROJECT_NAME)-v$(VERSION).tar.gz -C release .
	@echo "Release created: release/$(PROJECT_NAME)-v$(VERSION).tar.gz"

.PHONY: version
version: ## Show version information
	@echo "ChitChat v$(VERSION)"
	@echo "Git Branch: $(GIT_BRANCH)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(GO_VERSION)"

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
	@$(GO) list -m all | $(GO) mod why

# ==============================================================================
# Documentation
# ==============================================================================

.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	@if [ -x "$$(command -v swag)" ]; then \
		swag init -g main.go -o docs; \
	else \
		echo "swag not found, installing..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
		swag init -g main.go -o docs; \
	fi
	@echo "Documentation generated in docs/"

.PHONY: docs-serve
docs-serve: ## Serve documentation
	@echo "Serving documentation..."
	@if [ -x "$$(command -v godoc)" ]; then \
		godoc -http=:6060; \
	else \
		echo "Starting documentation server..."; \
		$(GO) tool godoc -http=:6060; \
	fi

# ==============================================================================
# Quality Assurance
# ==============================================================================

.PHONY: qa
qa: test lint vet security-scan ## Run all quality checks
	@echo "All quality checks passed! ✓"

.PHONY: ci
ci: deps test lint vet ## Run CI pipeline
	@echo "CI pipeline completed successfully!"

# ==============================================================================
# Development Tools
# ==============================================================================

.PHONY: tools
tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/cespare/reflex@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Tools installed."

# ==============================================================================
# Database Migration Tools (Optional)
# ==============================================================================

.PHONY: migrate-create
migrate-create: ## Create a new migration file
	@echo "Creating new migration..."
	@if [ -x "$$(command -v migrate)" ]; then \
		read -p "Enter migration name: " name; \
		migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $${name}; \
	else \
		echo "migrate not found, installing..."; \
		go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
		read -p "Enter migration name: " name; \
		migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $${name}; \
	fi

.PHONY: migrate-up
migrate-up: ## Apply all migrations
	@echo "Applying migrations..."
	@if [ -x "$$(command -v migrate)" ]; then \
		migrate -path $(MIGRATIONS_DIR) -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" up; \
	else \
		echo "migrate tool not found"; \
	fi

.PHONY: migrate-down
migrate-down: ## Rollback last migration
	@echo "Rolling back migration..."
	@if [ -x "$$(command -v migrate)" ]; then \
		migrate -path $(MIGRATIONS_DIR) -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" down 1; \
	else \
		echo "migrate tool not found"; \
	fi

# ==============================================================================
# Main Targets
# ==============================================================================

.PHONY: all
all: qa build ## Run all checks and build

.PHONY: setup
setup: env deps tools db-start ## Complete setup for development
	@echo "Setup complete! Run 'make dev' to start development."

.PHONY: deploy
deploy: qa docker-build docker-push ## Deploy to production
	@echo "Deployment complete!"

# ==============================================================================
# Debugging
# ==============================================================================

.PHONY: debug
debug: ## Debug the application
	@echo "Starting debug mode..."
	@$(GO) build -gcflags="all=-N -l" -o $(BINARY)-debug
	@dlv exec $(BINARY)-debug

.PHONY: pprof
pprof: ## Start pprof server
	@echo "Starting pprof server on :6060..."
	@$(GO) tool pprof -http=:6060 http://localhost:8080/debug/pprof/profile

.PHONY: trace
trace: ## Generate execution trace
	@echo "Generating trace..."
	@curl -o trace.out http://localhost:8080/debug/pprof/trace?seconds=5
	@$(GO) tool trace trace.out

# ==============================================================================
# Backup & Restore
# ==============================================================================

.PHONY: backup
backup: ## Backup database
	@echo "Backing up database..."
	@mkdir -p backups
	@PGPASSWORD=$(POSTGRES_PASSWORD) pg_dump -h $(POSTGRES_HOST) -p $(POSTGRES_PORT) -U $(POSTGRES_USER) $(POSTGRES_DB) > backups/backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "Backup complete: backups/backup_*.sql"

.PHONY: restore
restore: ## Restore database from backup
	@echo "Restoring database..."
	@if [ -z "$(BACKUP_FILE)" ]; then \
		echo "Usage: make restore BACKUP_FILE=backups/backup_file.sql"; \
		exit 1; \
	fi
	@PGPASSWORD=$(POSTGRES_PASSWORD) psql -h $(POSTGRES_HOST) -p $(POSTGRES_PORT) -U $(POSTGRES_USER) -d $(POSTGRES_DB) < $(BACKUP_FILE)
	@echo "Restore complete."

# ==============================================================================
# Performance Testing
# ==============================================================================

.PHONY: bench
bench: ## Run performance benchmarks
	@echo "Running performance benchmarks..."
	@if [ -x "$$(command -v wrk)" ]; then \
		echo "Testing API endpoints..."; \
		wrk -t4 -c100 -d30s http://localhost:8080/health; \
	else \
		echo "wrk not found, installing..."; \
		sudo apt-get install -y wrk || brew install wrk; \
		wrk -t4 -c100 -d30s http://localhost:8080/health; \
	fi

.PHONY: load-test
load-test: ## Run load testing
	@echo "Running load test..."
	@if [ -x "$$(command -v vegeta)" ]; then \
		echo "GET http://localhost:8080/health" | vegeta attack -duration=30s -rate=100 | vegeta report; \
	else \
		echo "vegeta not found, installing..."; \
		go install github.com/tsenart/vegeta/v12@latest; \
		echo "GET http://localhost:8080/health" | vegeta attack -duration=30s -rate=100 | vegeta report; \
	fi

# ==============================================================================
# Git Operations
# ==============================================================================

.PHONY: git-hooks
git-hooks: ## Install git hooks
	@echo "Installing git hooks..."
	@cp scripts/git-hooks/* .git/hooks/
	@chmod +x .git/hooks/*
	@echo "Git hooks installed."

.PHONY: pre-commit
pre-commit: fmt lint vet test ## Run pre-commit checks
	@echo "Pre-commit checks passed! ✓"

# ==============================================================================
# API Testing
# ==============================================================================

.PHONY: api-test
api-test: ## Test API endpoints
	@echo "Testing API endpoints..."
	@if [ -x "$$(command -v newman)" ]; then \
		newman run tests/api-collection.json; \
	else \
		echo "newman not found, installing..."; \
		npm install -g newman; \
		newman run tests/api-collection.json; \
	fi

.PHONY: swagger
swagger: ## Generate Swagger/OpenAPI documentation
	@echo "Generating Swagger documentation..."
	@if [ -x "$$(command -v swag)" ]; then \
		swag init; \
	else \
		echo "swag not found, installing..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
		swag init; \
	fi

# ==============================================================================
# Default target
# ==============================================================================

.DEFAULT_GOAL := help
