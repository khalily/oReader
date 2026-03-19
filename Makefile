.PHONY: test build run migrate-up migrate-down migrate-create clean

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=oreader
BINARY_UNIX=$(BINARY_NAME)_unix

# Main package
MAIN_PACKAGE=./cmd/server

# Build directory
BUILD_DIR=./build

# Test coverage output
COVERAGE_DIR=./coverage
COVERAGE_FILE=$(COVERAGE_DIR)/coverage.out

# Database migration
MIGRATIONS_DIR=./migrations
DATABASE_URL?=oreader.db

# Default target
all: test build

## test: Run all tests with coverage
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

## build-prod: Build production binary with embedded frontend
build-prod: build-prepare
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Production binary: $(BUILD_DIR)/$(BINARY_NAME)"

## dev: Run backend in API-only mode (no frontend build required)
dev:
	$(GOCMD) run -tags noembed $(MAIN_PACKAGE)

## dev-full: Run both backend and frontend development servers
dev-full:
	@./dev.sh

## run: Run the application (alias for dev)
run: dev

## migrate-up: Run database migrations up
migrate-up:
	@echo "Running migrations..."
	migrate -path $(MIGRATIONS_DIR) -database sqlite3://$(DATABASE_URL) up

## migrate-down: Rollback database migrations
migrate-down:
	@echo "Rolling back migrations..."
	migrate -path $(MIGRATIONS_DIR) -database sqlite3://$(DATABASE_URL) down

## migrate-create: Create a new migration file (usage: make migrate-create name=create_users)
migrate-create:
	@test -n "$(name)" || (echo "Usage: make migrate-create name=migration_name" && exit 1)
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

## migrate-version: Show migration version
migrate-version:
	migrate -path $(MIGRATIONS_DIR) -database sqlite3://$(DATABASE_URL) version

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

## docker-build: Build Docker image
docker-build:
	docker build -t oreader:latest .

## docker-run: Run Docker container
docker-run:
	docker run -p 8080:8080 oreader:latest

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':'
