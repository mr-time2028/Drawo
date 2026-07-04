# `internal/repositories`

**Responsibility:** Persist and retrieve domain entities.

Each repository implements a port from `internal/core/ports` using GORM:

- `user_repository.go` — account persistence.
- `profile_repository.go` — profile persistence.
- `room_repository.go` — room persistence.

Repositories translate between domain structs and the database.
They return plain domain errors or `gorm.ErrRecordNotFound`;
HTTP status mapping happens in controllers.

**Design pattern:** Repository (DDD / Clean Architecture).

**Testing strategy:** Integration tests with a test PostgreSQL container.
