# `internal/infrastructure`

**Responsibility:** Provide concrete infrastructure adapters.

Sub-packages:

- **`database/`** — Relational database connectivity via GORM with factory abstraction supporting drivers like PostgreSQL, MySQL, SQLite, etc.
- **`cache/`** — Non-relational database / key-value cache layer implementing `ports.CacheRepository` with driver abstraction supporting Redis, in-memory storage, etc.
- **`storage/`** — File/object storage adapters such as local filesystem and MinIO/S3-compatible storage.
- **`di/`** — Dependency injection container that wires everything together.

Realtime WebSocket runtime code lives in `internal/realtime`, and its delivery adapter lives in `internal/delivery/websocket`. They were intentionally moved out of infrastructure because they represent Drawo's realtime application runtime, not an external infrastructure adapter.

These packages implement the "driven ports" defined in `internal/core/ports`.
They are allowed to import third-party libraries and framework-specific code.

**Design pattern:** Adapter (Hexagonal Architecture).

**Testing strategy:** Health checks verify connectivity; unit tests verify memory adapters and driver registration; integration tests verify behavior against real infrastructure in containers.
