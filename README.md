# Drawo Workspace

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

## Backend with Docker

```bash
cd backend
cp .env.example .env
docker compose up -d --build
```

Backend API:

```text
http://localhost:8080
```


## Frontend with Docker

Start the backend first, then:

```bash
cd frontend
cp .env.example .env
docker compose up -d --build
```

Frontend:

```text
http://localhost:3000
```

The frontend Nginx container proxies `/api`, `/api/v1/ws`, and `/uploads` to the backend on the host at port `8080`.

## Local development without Docker app containers

Backend dev infrastructure terminal:

```bash
make backend-dev-up
```

Backend app terminal:

```bash
make backend-serve
```

Frontend terminal:

```bash
cd frontend/app
npm install
npm run dev
```

Frontend dev server:

```text
http://localhost:5173
```

## Convenience commands from repository root

```bash
make backend-up
make backend-down
make backend-dev-up
make backend-dev-down
make backend-serve
make frontend-up
make frontend-down
make frontend-dev
make test
```

Project-specific instructions:

```text
backend/README.md
frontend/README.md
```
