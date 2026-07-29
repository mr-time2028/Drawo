# Drawo Frontend

The frontend project owns its own Docker Compose file and env example.

```text
frontend/
  docker-compose.yml
  .env.example
  app/
    package.json
    src/
    Dockerfile
    nginx.conf
```

## Run frontend with Docker

Start the backend first. Then from `frontend/`:

```bash
cp .env.example .env
docker compose up -d --build
```

Open:

```text
http://localhost:3000
```

The Nginx container proxies backend traffic to the backend exposed on the host at port `8080`.

## Run frontend locally with npm

From `frontend/`:

```bash
cp .env.example .env
make install
make dev
```

Or directly:

```bash
cd app
npm install
npm run dev
```

Open:

```text
http://localhost:5173
```

## Test

```bash
make test
```

## Typecheck

```bash
make typecheck
```

## Build

```bash
make build
```

## Environment

Only frontend-safe Vite variables belong in `frontend/.env`:

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/api/v1/ws
VITE_DEFAULT_LANGUAGE=fa
```

No backend secrets, database values, Redis values, or Swagger values belong in frontend env files.
