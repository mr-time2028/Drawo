# Drawo Backend

## Requirements

- Go
- Docker
- Docker Compose

## Fresh clone setup

From the repository root:

```bash
cp .env.example .env
make backend-infra-up
make backend-download
make backend-migrate
make backend-dev
```

Backend API:

```text
http://localhost:8080
```

Health check:

```text
http://localhost:8080/health/ping
```

## Run backend directly from this folder

```bash
cp .env.example .env
make dev-up
make deps
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

## Stop infrastructure

```bash
make dev-down
```
