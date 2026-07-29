# Drawo frontend Makefile
#
# Use this from the frontend/ directory. The React/Vite application lives in
# frontend/app, while frontend/docker-compose.yml belongs to the frontend project.

.PHONY: up down logs install dev test test-coverage typecheck build preview

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

install:
	cd app && npm install

dev:
	cd app && npm run dev

test:
	cd app && npm test

test-coverage:
	cd app && npm run test:coverage

typecheck:
	cd app && npm run typecheck

build:
	cd app && npm run build

preview:
	cd app && npm run preview
