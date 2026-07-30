# Drawo frontend Makefile
#
# Use this from the frontend/ directory. The React/Vite application lives in
# frontend/app, while frontend/docker-compose.yml belongs to the frontend project.

.PHONY: up down logs install dev test test-coverage typecheck build preview lint lint:fix format env

# First-run helper: copy .env.example -> .env only if .env does not exist yet.
env:
	@test -f .env || (echo "→ Creating frontend/.env from .env.example" && cp .env.example .env)

up: env
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

install: env
	cd app && npm install

dev: env
	cd app && npm run dev

test: env
	cd app && npm test

test-coverage: env
	cd app && npm run test:coverage

typecheck: env
	cd app && npm run typecheck

build: env
	cd app && npm run build

preview: env
	cd app && npm run preview

lint: env
	cd app && npm run lint

lint:fix: env
	cd app && npm run lint:fix

format:
	cd app && npm run format
