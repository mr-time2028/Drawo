# `internal/services`

**Responsibility:** Implement application use cases.

Each file implements one port from `internal/core/ports`:

- `auth_service.go` — login, register, refresh, logout.
- `user_service.go` — profile management and dashboard data.

Services contain business logic but no HTTP or SQL details.
They receive repositories through their constructors.

**Design pattern:** Application Service (DDD).
Services orchestrate repositories to fulfill use cases.

**Testing strategy:** Unit tests with mocked repositories.
