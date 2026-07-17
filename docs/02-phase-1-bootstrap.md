# Phase 1 — Project Bootstrap & Clean Skeleton

**Status:** Completed.

## What was delivered

1. **Clean Architecture layout**
   - `internal/core/domain/` — Granular domain entities (User, Friend, GameHistory, etc.) split into modular files.
   - `internal/core/ports/` — Granular repository and service interfaces split by domain.
   - `internal/services/` — Placeholder application services.
   - `internal/repositories/` — Modular GORM repository implementations for each domain port.
   - `internal/delivery/http/` — Gin controllers, middlewares, routes.
   - `internal/infrastructure/` — DB, Redis, DI container.
   - `pkg/` — cross-cutting utilities (errors, logger, security, validator).

2. **Dependency injection**
   - `internal/infrastructure/di/container.go` wires DB, Redis, repositories, and services.
   - Services and repositories receive dependencies through constructors.

3. **Configuration**
   - All settings are loaded from environment variables.
   - A local `.env` file in the `app/` directory is loaded automatically via `godotenv`.
   - `.env.example` at the repository root documents every variable with defaults.
   - The `.config.yml` file was removed; `.env` is now the single local config source.

4. **Structured logging**
   - `pkg/logger` wraps `slog` with request IDs and user IDs.
   - HTTP middleware logs every request with status, latency, and client IP.

5. **HTTP server**
   - Graceful shutdown on SIGINT/SIGTERM.
   - Middleware: request ID, logger, recovery, CORS.
   - Health endpoints: `/health/ping` and `/health`.

6. **Docker support**
   - `docker-compose-dev.yml`: Postgres + Redis only.
   - `docker-compose.yml`: full production stack (nginx, app, postgres, redis).
   - `app/Dockerfile` multi-stage build.
   - `nginx/Dockerfile` + `nginx.conf` with WebSocket support.

7. **Makefile**
   - `make dev-up`, `make prod-up`, `make build`, `make test`, `make test-race`, `make fmt`.

8. **Package READMEs**
   - Every top-level package has a README explaining its responsibility, design pattern,
     and testing strategy.

9. **Unit tests**
   - `config/config_test.go`
   - `pkg/errors/errors_test.go`
   - `pkg/validator/validator_test.go`
   - `pkg/security/password_test.go`

## Verification

Run these commands in the `app/` directory:

```bash
go build -o /tmp/drawo .
go test ./...
```

Both should succeed.

## Known limitations (to be addressed in later phases)

- Auth, user, and room services are placeholders.
- No database migrations yet (tables do not exist).
- No WebSocket room manager yet.
- No frontend integration yet.

## Next step

**Phase 2 — Database Schema & Migrations.**
We will design the full relational schema and create up/down migrations for every table.
