# `internal/infrastructure`

**Responsibility:** Provide concrete infrastructure adapters.

Sub-packages:

- **`database/`** — Relational database connectivity via GORM with factory abstraction supporting drivers like PostgreSQL, MySQL, SQLite, etc.
- **`cache/`** — Non-relational database / key-value cache layer implementing `ports.CacheRepository` with driver abstraction supporting Redis, in-memory storage, etc.
- **`websocket/`** — WebSocket hub, room goroutines, and message protocol.
- **`di/`** — Dependency injection container that wires everything together.

These packages implement the "driven ports" defined in `internal/core/ports`.
They are allowed to import third-party libraries and framework-specific code.

**Design pattern:** Adapter (Hexagonal Architecture).

**Testing strategy:** Health checks verify connectivity; unit tests verify memory adapters and driver registration; integration tests verify behavior against real infrastructure in containers.
