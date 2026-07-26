# Drawo backend Makefile
#
# Use this from the backend/ directory for backend-only development and Go tasks.

.PHONY: dev-up dev-down dev-logs deps tidy migrate serve build test test-race test-coverage lint fmt

# Development commands (backing services only: Postgres + Redis)
dev-up:
	docker compose -f docker-compose-dev.yml up -d

dev-down:
	docker compose -f docker-compose-dev.yml down

dev-logs:
	docker compose -f docker-compose-dev.yml logs -f

# Go dependency commands
deps:
	cd app && go mod download

tidy:
	cd app && go mod tidy

# Go application commands
migrate:
	cd app && go run . migrate

serve:
	cd app && go run . serve

build:
	cd app && go build -o bin/drawo .

test:
	cd app && go test ./...

test-race:
	cd app && go test -race ./...

test-coverage:
	cd app && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html && rm coverage.out

fmt:
	cd app && gofmt -w .

lint:
	cd app && golangci-lint run ./...
