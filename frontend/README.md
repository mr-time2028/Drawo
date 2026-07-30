# Drawo Frontend

React 18 + TypeScript + Vite single-page application for the Drawo drawing &
guessing game. Uses `@tanstack/react-router`, Zustand, `react-i18next` (EN + FA
with full RTL support), and a token-driven light/dark theme.

```text
frontend/
  docker-compose.yml   # Nginx container build
  .env.example         # copy to .env (auto-done by `make install`)
  Makefile
  app/
    package.json
    .nvmrc             # Node 20
    .npmrc
    vite.config.ts     # reads env from ../ (i.e. frontend/.env)
    src/
```

## First-time setup

Requires **Node.js 20.x**. If you use `nvm`:

```bash
cd frontend/app
nvm use            # reads .nvmrc
```

From the repository root:

```bash
make frontend-install     # installs deps + creates frontend/.env from .env.example
make frontend-dev         # starts vite at http://localhost:5173
```

Or from inside `frontend/`:

```bash
make install
make dev
```

The Vite dev server reads environment variables from `frontend/.env` (**not**
`frontend/app/.env`). `VITE_API_BASE_URL` and `VITE_WS_URL` default to
`http://localhost:8080` which matches the backend defaults.

## Scripts

From `frontend/app/`:

| Command                | What it does                                     |
| ---------------------- | ------------------------------------------------ |
| `npm run dev`          | Start Vite dev server on :5173                   |
| `npm run build`        | Typecheck + production build to `app/dist/`      |
| `npm run preview`      | Preview the production build                     |
| `npm run typecheck`    | Run `tsc -b`                                     |
| `npm test`             | Run Vitest suite once                            |
| `npm run test:watch`   | Run Vitest in watch mode                         |
| `npm run lint`         | ESLint (flat config, TS/TSX)                     |
| `npm run lint:fix`     | ESLint with `--fix`                              |
| `npm run format`       | Prettier write                                   |
| `npm run format:check` | Prettier check (used in CI)                      |

## Running with Docker

```bash
make up     # from frontend/, or `make frontend-up` from repo root
```

The frontend container runs Nginx on `FRONTEND_HTTP_PORT` (default 3000) and
reverse-proxies `/api`, `/api/v1/ws`, and `/uploads/` to the backend on
`http://host.docker.internal:8080`.

## Dev helpers

- Visit `http://localhost:5173/__dev__` in development for a design-system
  preview page (button variants, colors, typography, form controls).
- Append `?mock=1` to dashboard pages during Phase 1–3 to force mock data.
