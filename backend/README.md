# Drawo Backend

The backend project owns its own Docker Compose file and env example.

```text
backend/
  docker-compose.yml
  .env.example
  app/
    Dockerfile
    go.mod
    migrations/
    internal/
```

## Requirements

- Go
- Docker
- Docker Compose

## Run full backend stack with Docker

From `backend/`:

```bash
cp .env.example .env
docker compose up -d --build
```

Backend API:

```text
http://localhost:8080
```

Health check:

```text
http://localhost:8080/health/ping
```

## Run only backend development infrastructure

Use this when you want Docker to run only Postgres + Redis, while you run the Go app manually:

```bash
docker compose -f docker-compose-dev.yml up -d
```

or:

```bash
make dev-up
```

## Run backend locally with Go

Start Postgres and Redis first. You can use the backend compose stack or your own local services.

```bash
cd app
go run . migrate
go run . serve
```

Or from `backend/`:

```bash
make migrate
make serve
```

## Go dependency commands

Download modules for a fresh clone:

```bash
make deps
```

Tidy modules after adding/removing imports:

```bash
make tidy
```

Use `make deps` for normal setup. Use `make tidy` only when you intentionally want to update `go.mod` / `go.sum`.

## Test

```bash
make test
```

## Build

```bash
make build
```

## Stop Docker stack

```bash
make down
```
