# `internal/delivery/http`

**Responsibility:** Expose the application over HTTP.

Sub-packages:

- **`controllers/`** — Gin handlers. They are thin: parse input, call services,
  return JSON. No business logic.
- **`middlewares/`** — Cross-cutting HTTP concerns: request IDs, logging,
  recovery, CORS, authentication, rate limiting.
- **`routes/`** — Central route registration.

**Design pattern:** Adapter (Hexagonal Architecture).
The HTTP layer is an adapter that drives the application layer.

**Testing strategy:** HTTP integration tests using `httptest.NewServer`.
