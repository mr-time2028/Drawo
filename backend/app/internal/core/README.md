# `internal/core`

**Responsibility:** Define the business heart of Drawo.

This package contains two sub-packages:

- **`domain/`** — Pure business entities (User, Profile, Room, Player, etc.).
  These structs have no framework dependencies.
- **`ports/`** — Interfaces that the application layer depends on.
  They are implemented by adapters (repositories, services).

**Design pattern:** Hexagonal / Clean Architecture.
The core knows nothing about HTTP, GORM, Redis, or WebSockets.
This makes business rules easy to test and protects the codebase from
framework lock-in.

**Trade-off:** More packages and interfaces than a simple CRUD app.
The benefit is that we can swap PostgreSQL for SQLite, or Gin for Echo,
without touching domain logic.
