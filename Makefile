.PHONY: test test-all build run migrate-up migrate-down migrate-create clean \
       converter-install converter-dev converter-test \
       lint-openapi \
       docker-dev docker-prod docker-test docker-down docker-down-prod \
       docker-logs docker-build docker-clean

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=oreader

# Main package
MAIN_PACKAGE=./cmd/server

# Build directory
BUILD_DIR=./build

# Test coverage output
COVERAGE_DIR=./coverage
COVERAGE_FILE=$(COVERAGE_DIR)/coverage.out

# Database migration
MIGRATIONS_DIR=./migrations
DATABASE_URL?=mysql://oreader:oreader@tcp(localhost:3306)/oreader?charset=utf8mb4&parseTime=True&loc=Local

# Default target
all: test build

## test: Run all tests with coverage (requires MySQL)
test:
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) ./internal/... || true
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_DIR)/coverage.html || true

## test-short: Run short tests
test-short:
	$(GOTEST) -v -short ./...

## test-coverage: Show test coverage summary
test-coverage:
	$(GOTEST) -cover ./...

## build-prepare: Copy frontend dist for embedding
build-prepare:
	@echo "Preparing frontend for embedding..."
	@rm -rf $(MAIN_PACKAGE)/dist
	@mkdir -p $(MAIN_PACKAGE)/dist
	@cp -r web/dist/* $(MAIN_PACKAGE)/dist/
	@echo "Frontend copied for embedding"

## build: Build the binary with embedded frontend (alias for build-prod)
build: build-prod

## build-prod: Build production binary with embedded frontend (pure Go, no CGO)
build-prod: build-prepare
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Production binary: $(BUILD_DIR)/$(BINARY_NAME)"

## dev: Run backend in API-only mode (no frontend build required)
dev:
	CGO_ENABLED=0 $(GOCMD) run -tags noembed $(MAIN_PACKAGE)

## dev-full: Run full dev environment via Docker Compose
dev-full:
	docker compose up --build

## run: Run the application (alias for dev)
run: dev

## migrate-up: Run database migrations up (MySQL)
migrate-up:
	@echo "Running migrations..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

## migrate-down: Rollback database migrations (MySQL)
migrate-down:
	@echo "Rolling back migrations..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

## migrate-create: Create a new migration file (usage: make migrate-create name=create_users)
migrate-create:
	@test -n "$(name)" || (echo "Usage: make migrate-create name=migration_name" && exit 1)
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

## migrate-version: Show migration version
migrate-version:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" version

## clean: Clean build artifacts
clean:
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@rm -rf $(MAIN_PACKAGE)/dist
	$(GOCLEAN)

## deps: Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## lint: Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Please install golangci-lint" && exit 1)
	golangci-lint run ./...

## lint-openapi: Lint OpenAPI spec with redocly
lint-openapi:
	npx @redocly/cli lint docs/openapi.yaml --config .redocly.yaml

## vulncheck: Run security vulnerability check
vulncheck:
	@which govulncheck > /dev/null || (echo "Please install govulncheck: go install golang.org/x/vuln/cmd/govulncheck@latest" && exit 1)
	govulncheck ./...

## frontend-install: Install frontend dependencies
frontend-install:
	cd web && npm install

## frontend-dev: Start frontend development server
frontend-dev:
	cd web && npm run dev

## frontend-build: Build frontend for production
frontend-build:
	cd web && npm run build

## converter-install: Install Python converter dependencies
converter-install:
	@echo "Installing converter dependencies..."
	pip install -r converter/requirements.txt
	@echo "Converter dependencies installed"

## converter-dev: Start the paper converter gRPC server
converter-dev:
	@echo "Starting paper converter gRPC server..."
	cd converter && python server.py

## converter-test: Run converter unit tests
converter-test:
	@echo "Running converter tests..."
	cd converter && python -m pytest tests/ -v

## test-all: Run all tests (Go + Frontend + Converter)
test-all:
	@echo "========== Running Go Tests =========="
	$(GOTEST) -v -race ./internal/...
	@echo ""
	@echo "========== Running Frontend Tests =========="
	cd web && npm test -- --run
	@echo ""
	@echo "========== Running Converter Tests =========="
	cd converter && python -m pytest tests/ -v
	@echo ""
	@echo "========== All Tests Complete =========="

# =============================================================================
# Docker Compose targets
# =============================================================================

## docker-dev: Start development environment with Docker Compose
docker-dev:
	docker compose up --build

## docker-prod: Start production environment with Docker Compose
docker-prod:
	docker compose --profile prod up -d --build

## docker-test: Run tests in Docker Compose environment
docker-test:
	docker compose --profile test up --build --abort-on-container-exit

## docker-down: Stop Docker Compose development environment
docker-down:
	docker compose down

## docker-down-prod: Stop Docker Compose production environment
docker-down-prod:
	docker compose --profile prod down

## docker-logs: View Docker Compose logs (follow mode)
docker-logs:
	docker compose logs -f

## docker-build: Build all Docker images
docker-build:
	docker compose build

## docker-clean: Remove all Docker Compose resources (including volumes)
docker-clean:
	docker compose down -v --rmi local

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':'
