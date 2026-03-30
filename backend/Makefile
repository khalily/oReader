COVERAGE_DIR = ./coverage

.PHONY: install test lint fmt migrate-up migrate-down migrate-create

install:
	go mod download

test:
	@mkdir -p $(COVERAGE_DIR)
	go test -v -race -p 1 -coverprofile=$(COVERAGE_DIR)/coverage.out ./internal/...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@test -n "$(name)" || (echo "Usage: make migrate-create name=xxx" && exit 1)
	migrate create -ext sql -dir ./migrations -seq $(name)
