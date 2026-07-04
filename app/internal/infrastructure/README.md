# `internal/infrastructure`

**Responsibility:** Provide concrete infrastructure adapters.

Sub-packages:

- **`database/`** — PostgreSQL connection via GORM.
- **`redis/`** — Redis client for sessions, cache, and rate limits.
- **`websocket/`** — WebSocket hub, room goroutines, and message protocol.
- **`di/`** — Dependency injection container that wires everything together.

These packages implement the "driven ports" defined in `internal/core/ports`.
They are allowed to import third-party libraries and framework-specific code.

**Design pattern:** Adapter (Hexagonal Architecture).

**Testing strategy:** Health checks verify connectivity; integration tests
verify behavior against real infrastructure in containers.
