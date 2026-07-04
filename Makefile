# Drawo Makefile
#
# One-command targets for development and production.

.PHONY: dev-up dev-down dev-logs prod-up prod-down prod-logs build test test-race lint fmt

# Development commands (backing services only)
dev-up:
	docker-compose -f docker-compose-dev.yml up -d

dev-down:
	docker-compose -f docker-compose-dev.yml down

dev-logs:
	docker-compose -f docker-compose-dev.yml logs -f

# Production commands (full stack)
prod-up:
	docker-compose up --build -d

prod-down:
	docker-compose down

prod-logs:
	docker-compose logs -f

# Go commands (run from app/)
build:
	cd app && go build -o bin/drawo .

test:
	cd app && go test ./...

test-race:
	cd app && go test -race ./...

fmt:
	cd app && gofmt -w .

lint:
	cd app && golangci-lint run ./...
