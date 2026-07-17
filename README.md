# Drawo

A production-quality multiplayer drawing and guessing game.

## Quick start (development)

1. Copy the environment file and adjust it:
   ```bash
   cp .env.example .env
   ```

2. Start the backing services:
   ```bash
   make dev-up
   ```

3. Run database migrations (Required on first start):
   ```bash
   cd app && go run . migrate
   ```

4. Run the Go server:
   ```bash
   cd app && go run . serve
   ```

5. Visit `http://localhost:8080/health/ping`.

## Configuration

Drawo is configured entirely through environment variables. For local development,
copy `.env.example` to a single `.env` file in the repository root. Docker Compose
reads this file automatically, and the Go app also loads it when started from the
`app/` directory.

`.env.example` at the repository root contains every available variable with sensible defaults.

In production, set the same variables directly in your container orchestrator.

## Architecture

The backend follows **Clean Architecture / Hexagonal Architecture**:

- `internal/core` — pure domain entities and interfaces (no framework deps).
- `internal/services` — application use cases.
- `internal/repositories` — GORM persistence.
- `internal/delivery/http` — Gin controllers, middlewares, routes.
- `internal/infrastructure` — DB, Redis/cache, file storage, DI container.
- `internal/realtime` — WebSocket protocol, secure socket auth/re-auth, hub, and per-room goroutines.
- `internal/delivery/websocket` — WebSocket delivery adapter and route registration.
- `pkg` — cross-cutting utilities.

See `docs/01-analysis-and-roadmap.md` for the full development roadmap.

## Commands

| Command | Description |
|---------|-------------|
| `make dev-up` | Start Postgres + Redis for development |
| `make prod-up` | Start full production stack |
| `make build` | Build the Go binary |
| `make test` | Run unit tests |
| `make test-race` | Run tests with race detector |
| `make test-coverage` | Generate HTML test coverage report |

