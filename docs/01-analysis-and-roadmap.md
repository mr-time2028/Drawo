# Drawo — Repository Analysis & Development Roadmap

> **Status:** Analysis complete. No production code has been written yet. This document is the foundation for incremental implementation.

---

## 1. Current Repository Snapshot

I inspected the existing code at `https://github.com/mr-time2028/Drawo` and the three uploaded files (`index.html`, `style.css`, `script.js`).

### 1.1 What already exists

| Area | Existing state |
|------|----------------|
| **HTTP framework** | Gin + Cobra CLI (`serve`, `migrate`, etc.) |
| **Config** | Viper-based `.config.yml` with defaults |
| **Database** | PostgreSQL via GORM + golang-migrate SQL migrations |
| **Auth** | JWT access + refresh tokens, bcrypt password hashing, login/register |
| **Users** | Basic `User` model (id, username, password, is_active, is_superuser) |
| **Rooms** | Partial private-room creation endpoint, no join logic |
| **WebSocket** | A central `Hub` with `Register/Unregister/Broadcast` channels and per-room client maps |
| **Docker** | Production (`nginx`, `app`, `postgres`) and dev (`postgres` only) Compose files |
| **Frontend** | Standalone HTML/CSS/JS drawing board with brush, eraser, rectangle, circle, triangle, color picker, save |

### 1.2 Notable gaps / bugs found

1. **Room model mismatch.** `internal/modules/room/services/room_service.go` sets `Type: roomModel.Private`, but `room_model.go` only has `IsPrivate bool`. The code currently does **not compile** in the room module.
2. **No Redis integration yet.** The required Redis stack (sessions, rate limiting, refresh-token rotation, presence) is absent from both code and Compose.
3. **No refresh-token persistence / rotation.** `GenerateAccessTokenByRefreshToken` verifies the refresh token and issues a new access token, but the old refresh token remains valid forever and there is no DB/Redis deny-list.
4. **WebSocket hub is a single bottleneck.** All rooms share one `select` loop. Under high concurrency this becomes a global lock and will not scale to thousands of players.
5. **No drawing synchronization protocol.** The WS layer only echoes raw strings; there is no message type system for `draw_stroke`, `clear`, `chat`, `guess`, etc.
6. **No game loop, scoring, words, categories, i18n, admin, or reporting.** These are entirely missing.
7. **Frontend is not integrated.** The uploaded board is a tutorial demo; it has no network layer, no game UI, no dashboard, no auth, no dark mode, and no audio.
8. **No structured logging, no request IDs, no rate limiting, no brute-force protection.**
9. **No package READMEs.** Per your requirements, every package needs a README explaining responsibility.
10. **No tests.** There are zero unit, integration, or WebSocket tests.

### 1.3 What we will preserve

- Gin + Cobra foundation (familiar to you).
- Module-based folder layout (`user`, `auth`, `room`, `game`, etc.).
- GORM + PostgreSQL + golang-migrate.
- JWT concept (we will harden it).
- The drawing-board HTML/CSS/JS as the visual starting point (we will extend it into a full SPA).
- The existing Docker Compose pattern (we will add Redis and health checks).

---

## 2. Proposed Clean Architecture

We will keep the code easy to read while applying Clean Architecture / SOLID / dependency injection. The folder layout will be:

```
Drawo/
├── app/
│   ├── cmd/                        # Cobra commands (serve, migrate, testdata)
│   ├── config/                     # Viper config + env defaults
│   ├── internal/
│   │   ├── core/                   # Domain entities + repository/usecase interfaces (no external deps)
│   │   │   ├── domain/             # Pure structs: User, Room, Game, Round, Word, ...
│   │   │   └── ports/              # Interfaces: UserRepository, TokenService, GameEngine, ...
│   │   ├── application/            # Business logic / use cases (services)
│   │   │   ├── auth/
│   │   │   ├── user/
│   │   │   ├── room/
│   │   │   ├── game/               # game loop, scoring, reports
│   │   │   └── admin/
│   │   ├── adapters/
│   │   │   ├── delivery/http/      # Gin controllers, middlewares, routes
│   │   │   ├── persistence/        # GORM repositories + Redis cache/sessions
│   │   │   ├── ws/                 # WebSocket hub, room goroutines, message protocol
│   │   │   └── storage/            # File uploads (songs, avatars)
│   │   └── infrastructure/         # DB/Redis connection, migration helpers, i18n loader
│   ├── pkg/                        # Cross-cutting utilities (errors, json, validator, logger, security)
│   ├── migrations/                 # SQL migrations (golang-migrate)
│   └── tests/                      # Integration & ws tests
├── web/                            # Frontend (vanilla ES modules, mobile-first, i18n ready)
├── nginx/
├── docker-compose.yml              # Production: nginx, app, postgres, redis
├── docker-compose-dev.yml          # Dev: app (host), postgres, redis
└── Makefile
```

### 2.1 Design decisions explained

- **Domain-first (`internal/core`).** Entities and interfaces live here with **no imports** from Gin, GORM, Redis, etc. This makes business rules testable in isolation and prevents framework lock-in.
- **Dependency injection via a `Provider` / container in `cmd/serve.go`.** Repositories and services are wired once at startup and passed downward. No `repository.New()` inside services.
- **One goroutine per room.** The global hub will only manage registration/unregistration. Each `Room` will run its own event loop on a dedicated goroutine with a buffered channel. This removes the global lock and lets rooms scale independently.
- **WebSocket message envelope.** Every WS frame will be JSON with `type`, `payload`, `seq`, and `timestamp`. The server synchronizes drawing **operations** (e.g., `stroke`, `line`, `shape`, `clear`), not full canvas images, minimizing bandwidth.
- **Redis as the real-time coordination layer.** Sessions, refresh-token families, rate-limit counters, presence, and matchmaking queues will live in Redis. PostgreSQL remains the source of truth for long-lived data.
- **i18n from day one.** All user-facing strings (including word dictionaries) are loaded from JSON files. Adding a language means adding one folder.

---

## 3. Development Roadmap

We will build the project in **18 phases**. Each phase is a complete, testable module.

### Phase 1 — Project Bootstrap & Clean Skeleton
- Refactor Go module layout to `internal/core`, `application`, `adapters`, `infrastructure`.
- Introduce dependency-injection container.
- Add structured logger (`slog` or `uber/zap`) with request IDs.
- Add package READMEs for every top-level package.
- Add `.env` support and environment-based config defaults.
- Create a minimal working HTTP server that compiles and passes a smoke test.

### Phase 2 — Database Schema & Migrations
- Design full relational schema for persistent entities (users, profiles, friendships, friend requests, game history, rounds, scores, reports, achievements, player statistics, user settings). Note: Public and private rooms are ephemeral runtime objects maintained in memory and coordinated via Redis/cache; they are NOT stored in the relational database.
- Write up/down migrations for every table.
- Add indexes, foreign keys, and `ON DELETE` rules.
- Document every relation in `docs/database-schema.md`.

### Phase 3 — Domain Entities & Repository Interfaces
- Define pure domain structs (no GORM tags).
- Define repository interfaces in `internal/core/ports`.
- Add value objects (Language, RoomType, GameState, Role, ReportReason).

### Phase 4 — Redis & Session Foundation
- Add Redis service to both Compose files.
- Implement Redis client wrapper, connection health check.
- Build session store for WebSocket auth.
- Implement sliding-window rate limiter helper.

### Phase 5 — Hardened Authentication
- Refactor JWT helpers to use `jwt/v5` typed claims.
- Add refresh-token families stored in Redis (rotation + reuse detection).
- Add login brute-force protection (failed-attempt counter per IP/username).
- Add logout / revoke session endpoints.
- Add secure password validation rules.
- Add CSRF-safe cookie option for web clients.

### Phase 6 — User Profile & Dashboard API
- Extend user model with avatar, email, phone, locale, theme, sound flags.
- CRUD profile endpoints with duplicate checks.
- Default avatar generation / upload.
- Email & phone verification placeholders (OTP service interface + mock provider).
- Dashboard stats endpoint (score, rank, games played, MVPs).

### Phase 7 — Internationalization (i18n)
- JSON-based translation files for English and Persian.
- Locale middleware + accept-language detection.
- i18n helper used in errors, validation, and notifications.
- RTL support foundation for Persian.

### Phase 8 — Admin Configuration API
- Admin middleware (`is_superuser`).
- Endpoints: upload site/background songs, add words/categories/bad-words, configure suggested-word count, round time, max rounds, min/max players, language matching.
- Global settings cache in Redis.
- Word dictionaries per language with point tiers (1/2/3).

### Phase 9 — WebSocket Architecture (Scalable Room Manager)
- Refactor hub to spawn per-room goroutines.
- Define WS message protocol (`auth`, `join`, `leave`, `chat`, `draw`, `game_event`).
- Implement client read/write pumps, heartbeat/ping-pong.
- Add room registry with RWMutex + channel-based event dispatch.
- Concurrency tests with `-race`.

### Phase 10 — Drawing Engine (Frontend + Protocol)
- Extend uploaded board: pencil, brush, marker, calligraphy, eraser, fill, eyedropper, undo/redo, zoom/pan, shapes, text, selection.
- Implement operational transform-like stroke messages (low bandwidth).
- Add tool-select sound toggle.
- Smooth drawing with `requestAnimationFrame` + pointer events + pressure support.

### Phase 11 — Game Loop
- Lobby → countdown → category selection → word point selection → word selection → drawing → guessing → round end → leaderboard → next round → game end.
- Word suggestion engine (N random words from chosen category & point tier).
- Timer service per room.
- Hint reveal logic (dashed placeholders, occasional letter reveal).
- Drawer rotation algorithm.

### Phase 12 — Chat, Guessing & Filtering
- Chat message handler with correct-guess detection (case/space insensitive, language aware).
- Drawer chat lock.
- Bad-word filter using admin-configured list + Aho–Corasick for speed.
- Rate-limited chat (per player, per room).
- Correct guess is hidden from others; only "X guessed the word" is broadcast.

### Phase 13 — Scoring & Reporting
- Two score types: **word score** (game) and **reputation score** (account behavior, default 10000, ban below 3000).
- Fair scoring: base points from chosen word tier + speed bonus + streak bonus.
- Animated score updates pushed via WebSocket.
- Reporting endpoint with reasons; reputation penalty on confirmed reports.
- MVP tracking (excluding private rooms).

### Phase 14 — Security Hardening
- Input sanitization (XSS): escape all dynamic HTML; use textContent only.
- GORM parameterized queries (already used) + continue review.
- CORS tightened per environment.
- Rate limiting on all public endpoints.
- Audit log middleware for sensitive actions.
- Secure headers (HSTS, CSP, X-Frame-Options).

### Phase 15 — Testing
- Unit tests for services with mocked repositories.
- Integration tests for HTTP endpoints (test DB in a container).
- WebSocket tests using Gorilla test helpers.
- Concurrency/race tests for room manager.
- Makefile targets: `test`, `test-integration`, `test-race`.

### Phase 16 — Docker Compose & Deployment
- Production Compose: nginx (SSL-ready), Go app (multi-stage build), Postgres, Redis, shared volumes, health checks.
- Dev Compose: Postgres + Redis only; app runs locally for fast iteration.
- Environment variable templates (`.env.example`).
- One-command Makefile: `make dev` and `make prod`.

### Phase 17 — Frontend UI
- Landing page with music toggle, login/register modals.
- Dashboard: profile, settings (theme, sound, language).
- Game room: players list, drawing board, chat, word placeholders, timer, scoreboard.
- Admin panel.
- Dark/light theme, mobile-first responsive CSS.
- Audio manager for site songs and game sounds.

### Phase 18 — Final Integration, Load Testing & Documentation
- End-to-end smoke test: register → create room → draw → guess.
- Load-testing guide using `k6` or `vegeta` for WS rooms.
- Complete `README.md`, `TESTING.md`, `DEPLOYMENT.md`.
- Wrap-up downloadable archive with all code.

---

## 4. Concurrency Strategy for WebSocket Rooms

This is the hardest part of a drawing game, so it deserves its own explanation.

### 4.1 Current problem
The existing `Hub.Run()` has one `select` over three channels and iterates all clients inside the same goroutine. With 1,000 rooms × 10 players, every broadcast must wait for every other broadcast. This is a global lock.

### 4.2 Proposed design

```
Hub (singleton)
  ├── Register   chan *Client
  ├── Unregister chan *Client
  └── rooms      map[string]*Room  (RWMutex for lookups)

Room (one per active room, one goroutine)
  ├── inbox      chan RoomEvent (buffered)
  ├── clients    map[string]*Client (mutex only when client set changes)
  ├── state      RoomState (mutex only when state changes)
  └── gameLoop   *GameLoop
```

- **Hub goroutine** only: adds/removes clients, creates room goroutines on demand, routes messages to the correct room's `inbox`.
- **Room goroutine** only: processes its own `inbox` serially, so state mutations need **no mutex** inside the event loop. Mutexes are used only for the client map (read-heavy) and cross-room queries from HTTP handlers.
- **Client goroutines** (read/write pumps) never touch shared state; they send events to the room's `inbox`.

### 4.3 Why this scales
- No global lock on the hot path (drawing/chat).
- Rooms are independent; one lagging room cannot block others.
- We can later shard rooms across multiple server instances by moving the registry to Redis.

---

## 5. Testing Strategy

| Type | Tooling | What it covers |
|------|---------|----------------|
| Unit | `go test` + testify + mock repositories | Services, scoring, helpers |
| Integration | `go test` + testcontainers-go (Postgres/Redis) | HTTP handlers end-to-end |
| WebSocket | Gorilla `httptest.NewServer` + `websocket.Dial` | Join/leave, draw, chat, game events |
| Concurrency | `go test -race` + custom stress tests | Room manager, hub registry |
| Load | `k6` (JavaScript) | 100+ concurrent rooms, latency distribution |
| Frontend | Manual + browser DevTools | UI flows on mobile/desktop |

At the end of the project I will provide a `TESTING.md` with copy-paste commands and sample `k6` scripts.

---

## 6. Immediate Next Step

**Phase 1 — Project Bootstrap & Clean Skeleton.**

I will:
1. Create the new directory layout under `/home/user/Drawo/app`.
2. Set up the dependency-injection container.
3. Add structured logging and environment config.
4. Write package READMEs.
5. Make sure `go build`, `go test ./...`, and `docker-compose -f docker-compose-dev.yml up` succeed.

After Phase 1 we will have a clean, compiling foundation. Then we move to the database schema in Phase 2.

If you want to adjust priorities (e.g., start with the drawing board first, or skip admin panel until later), let me know. Otherwise I will proceed with Phase 1.
