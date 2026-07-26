# Drawo root Makefile
#
# Use this from the repository root.

.PHONY: prod-up prod-down prod-logs backend-infra-up backend-infra-down backend-infra-logs backend-download backend-tidy backend-migrate backend-dev frontend-install frontend-dev backend-test frontend-test frontend-test-coverage test migrate

# Production-like stack: Nginx serves built frontend and proxies API/WebSocket.
prod-up:
	docker compose up -d --build

prod-down:
	docker compose down

prod-logs:
	docker compose logs -f

# Backend infrastructure only: Postgres + Redis.
# This delegates to backend/docker-compose-dev.yml.
backend-infra-up:
	cd backend && docker compose -f docker-compose-dev.yml up -d

backend-infra-down:
	cd backend && docker compose -f docker-compose-dev.yml down

backend-infra-logs:
	cd backend && docker compose -f docker-compose-dev.yml logs -f

# Download Go modules for a fresh clone without changing go.mod/go.sum.
backend-download:
	cd backend/app && go mod download

# Tidy Go modules after adding/removing backend imports. This can modify go.mod/go.sum.
backend-tidy:
	cd backend/app && go mod tidy

# Run database migrations while developing backend locally.
backend-migrate:
	cd backend/app && go run . migrate

# Run the Go backend directly on your machine.
# Start Postgres + Redis first with backend-infra-up.
backend-dev:
	cd backend/app && go run . serve

frontend-install:
	cd frontend && npm install

# Run the frontend directly on your machine.
frontend-dev:
	cd frontend && npm run dev

backend-test:
	cd backend/app && go test ./...

frontend-test:
	cd frontend && npm test

frontend-test-coverage:
	cd frontend && npm run test:coverage

test: backend-test frontend-test

# Run migrations against the production-like Docker network.
# For local/manual backend development, use backend-migrate.
migrate:
	docker compose run --rm app ./drawo migrate
