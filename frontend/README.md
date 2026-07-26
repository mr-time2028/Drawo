# Drawo Frontend

## Run production-like stack with Docker from repository root

```bash
make prod-up
```

Open:

```text
http://localhost
```

## Run frontend for development

Requires Node.js and npm.

```bash
cp .env.example .env
npm install
npm run dev
```

Open:

```text
http://localhost:5173
```

## Test

```bash
npm test
```

## Typecheck

```bash
npm run typecheck
```

## Build

```bash
npm run build
```

## Current status

The current frontend contains only the login screen. HTTP calls use Axios through `src/api/http.ts`. More UI will be added step by step.
