# `pkg`

**Responsibility:** Provide cross-cutting utilities used by any layer.

Packages here are not allowed to import `internal/` packages. They should be
self-contained and reusable across the whole application.

Sub-packages:

- **`errors/`** — Application error types and HTTP mapping.
- **`logger/`** — Structured logging with request context.
- **`security/`** — Password hashing, token generation, sanitization.
- **`validator/`** — Request validation wrapper.

**Design pattern:** Utility / Helper library.

**Testing strategy:** Pure unit tests; no external dependencies.
