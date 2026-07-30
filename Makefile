# Drawo root Makefile
#
# Use this from the repository root as a convenience wrapper. There is no root
# docker-compose.yml and no root .env. Backend and frontend are separate projects.

.PHONY: backend-up backend-down backend-logs backend-dev-up backend-dev-down backend-dev-logs backend-download backend-tidy backend-migrate backend-serve frontend-up frontend-down frontend-logs frontend-install frontend-dev backend-test frontend-test frontend-test-coverage test

backend-up:
	cd backend && docker compose up -d --build

backend-down:
	cd backend && docker compose down

backend-logs:
	cd backend && docker compose logs -f

backend-dev-up:
	cd backend && docker compose -f docker-compose-dev.yml up -d

backend-dev-down:
	cd backend && docker compose -f docker-compose-dev.yml down

backend-dev-logs:
	cd backend && docker compose -f docker-compose-dev.yml logs -f

# Create backend/.env from the example on first run (idempotent: won't clobber
# an existing .env the developer has customized).
backend-env:
	@test -f backend/.env || (echo "→ Creating backend/.env from .env.example" && cp backend/.env.example backend/.env)

# Create frontend/.env from the example on first run. The Vite dev server reads
# env from frontend/ (see envDir in vite.config.ts), not frontend/app/.
frontend-env:
	@test -f frontend/.env || (echo "→ Creating frontend/.env from .env.example" && cp frontend/.env.example frontend/.env)

backend-download: backend-env
	cd backend/app && go mod download

backend-tidy:
	cd backend/app && go mod tidy

backend-migrate: backend-env
	cd backend/app && go run . migrate

backend-serve: backend-env
	cd backend/app && go run . serve

frontend-up: frontend-env
	cd frontend && docker compose up -d --build

frontend-down:
	cd frontend && docker compose down

frontend-logs:
	cd frontend && docker compose logs -f

frontend-install: frontend-env
	cd frontend/app && npm install

frontend-dev: frontend-env
	cd frontend/app && npm run dev

backend-test:
	cd backend/app && go test ./...

frontend-test:
	cd frontend/app && npm test

frontend-test-coverage:
	cd frontend/app && npm run test:coverage

test: backend-test frontend-test
