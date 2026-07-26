# Drawo Workspace

Drawo is organized as a monorepo:

```text
Drawo/
  backend/   Go API, realtime game server, migrations
  frontend/  React/Vite browser app
```

## Run production-like stack with Docker

Copy the root Docker environment file:

```bash
cp .env.example .env
```

Builds the frontend, serves it with Nginx, and proxies `/api` plus `/api/v1/ws` to the Go backend:

```bash
make prod-up
```

Run migrations after containers are up:

```bash
make migrate
```

Open:

```text
http://localhost
```

Stop production stack:

```bash
make prod-down
```

## Development workflow

For development, run only infrastructure in Docker and run backend/frontend manually in terminals.

First-time setup:

```bash
cp .env.example .env
make backend-infra-up
make backend-download
make backend-migrate
```

Terminal 1 — run backend:

```bash
make backend-dev
```

Terminal 2 — install and run frontend:

```bash
make frontend-install
make frontend-dev
```

Backend API:

```text
http://localhost:8080
```

Frontend dev server:

```text
http://localhost:5173
```

Backend instructions:

```text
backend/README.md
```

Frontend instructions:

```text
frontend/README.md
```
