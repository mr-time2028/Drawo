# Drawo

Multiplayer drawing & guessing game. React/Vite/TypeScript frontend + Go/Gin backend, with Redis + Postgres and a WebSocket realtime layer.

## Repository layout

Drawo is organized as two separate projects inside one repository:

```text
Drawo/
  backend/        Go API, realtime server, migrations, backend docker-compose
    app/
  frontend/       React/Vite browser app, frontend docker-compose
    app/
```

There is intentionally no root `.env` and no root `docker-compose.yml`.

Each project owns its own environment and Docker Compose file:

```text
backend/.env.example
backend/docker-compose.yml
frontend/.env.example
frontend/docker-compose.yml
```

The first time you run a `make` target it auto-copies `.env.example` → `.env`
(only if `.env` does not exist yet), so your local secrets/settings are never
clobbered.

## Prerequisites

- **Node.js 20.x** (pinned in `frontend/app/.nvmrc`; if you use nvm run `nvm use`
  inside `frontend/app/`).
- **Go 1.23+** for running the backend outside Docker.
- **Docker + Docker Compose** for containerised runs.

## Backend with Docker

```bash
make backend-up
```

Backend API:

```text
http://localhost:8080
```

## Frontend with Docker

Start the backend first, then:

```bash
make frontend-up
```

Frontend (Nginx, reverse-proxies API/WS/uploads to host:8080):

```text
http://localhost:3000
```

## Local development without Docker app containers

Backend dev infrastructure (Postgres + Redis):

```bash
make backend-dev-up
```

Backend app (terminal 1):

```bash
make backend-serve
```

Frontend (terminal 2):

```bash
make frontend-install  # npm install + creates frontend/.env from .env.example if missing
make frontend-dev      # vite dev server on http://localhost:5173
```

Frontend dev server:

```text
http://localhost:5173
```

## Convenience commands from repository root

```bash
make backend-up            # start backend stack (docker)
make backend-down
make backend-dev-up        # start only Postgres + Redis for local Go dev
make backend-dev-down
make backend-serve         # go run . serve
make backend-migrate       # go run . migrate
make frontend-up           # start frontend (docker + nginx)
make frontend-down
make frontend-install      # npm install in frontend/app (creates .env if missing)
make frontend-dev          # vite dev server
make backend-test
make frontend-test
make test                  # both
```

Project-specific instructions:

```text
backend/README.md
frontend/README.md
```
