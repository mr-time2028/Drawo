# Drawo — Repository Analysis & Refined Roadmap

> **Status:** Phase 1 & 2 Completed. Core modular architecture, database migrations, and subjective port implementations are established with high test coverage.

---

## 1. Project Overview & Current State

Drawo is a production-quality multiplayer drawing and guessing game, following Clean Architecture principles in Go.

### 1.1 What has been implemented
| Area | Status |
|------|--------|
| **Core Architecture** | Subjective Modular Ports. Interfaces and implementations reside together in `internal/core/ports/repositories` and `internal/core/ports/services`. |
| **Domain Models** | Granular entities (User, Profile, Room, Friend, etc.) separated into subjective files in `internal/core/domain`. |
| **Database** | 12 modular SQL migrations using UUID primary keys. Integrated migration CLI in the app binary. |
| **Testing** | ~73% total coverage. Foundational packages (errors, validator, security, config) and logic-heavy services/repositories are near 100%. |
| **Features** | Ephemeral room discovery logic, Custom Word List support for private rooms, and MinPlayers configuration. |

---

## 2. Refined Development Roadmap

The project is structured into **15 remaining phases**. Each phase is a complete, testable module based on the original project requirements.

### Phase 1 & 2: Bootstrap & Database Foundation (COMPLETED)
- [x] Clean Architecture setup with Subjective Modular Ports.
- [x] High-coverage test suite for all foundational packages.
- [x] Relational schema for all 11 persistent entities.
- [x] 12 modular migrations with reversible logic.
- [x] Integrated `migrate` and `generate_migration` CLI tools.

### Phase 3: Redis & Session Infrastructure
- [ ] Add Redis service to Docker Compose files.
- [ ] Implement Redis-backed session store for high-performance WebSocket authentication.
- [ ] Build a sliding-window rate limiter helper for API security.
- [ ] Implement Redis discovery health checks.

### Phase 4: Hardened Authentication
- [ ] Implement JWT Access + Refresh token logic with `jwt/v5`.
- [ ] Add refresh-token rotation and reuse detection (stored in Redis).
- [ ] Implement secure password hashing with Bcrypt.
- [ ] Add login brute-force protection (failed-attempt tracking).
- [ ] Create Login, Register, Refresh, and Logout endpoints.

### Phase 5: User Profile & Verification
- [ ] Extend `Profile` logic: Avatar upload/generation, locale, theme, sound settings.
- [ ] Implement Profile CRUD with duplicate username/email checks.
- [ ] Add OTP (One-Time Password) service interface for Email/Phone verification.
- [ ] Implement mock OTP provider for testing.
- [ ] Create dashboard stats endpoint (Word Score, Reputation, MVPs, Games Played).

### Phase 6: Internationalization (i18n) & Words
- [ ] Implement JSON-based localization system (English/Persian).
- [ ] Add Word and Category management (Admin only).
- [ ] Dictionary system: per-language word lists with point tiers (1/2/3).
- [ ] Bad-word filter configuration using Aho–Corasick algorithm for speed.

### Phase 7: Admin Configuration API
- [ ] Site songs management: Landing page song queue (Admin upload).
- [ ] Background game sound management (Admin upload).
- [ ] Global game settings: Suggested words count, timers, max rounds.
- [ ] Account management: Ban/Unban (based on reputation or manual action).

### Phase 8: Scalable WebSocket Architecture
- [ ] Refactor Hub to manage independent room goroutines.
- [ ] Implement WS message envelope protocol (`auth`, `join`, `leave`, `chat`, `draw`, `game`).
- [ ] Client read/write pumps with heartbeat/ping-pong.
- [ ] Room registry with RWMutex for multi-instance scalability.

### Phase 9: Drawing Engine & Protocol
- [ ] Implement operational-sync protocol: Synchronize strokes/shapes instead of images.
- [ ] Protocols for: Pencil, Brush, Marker, Calligraphy, Eraser, Fill, undo/redo, shapes.
- [ ] Add tool-selection sound logic hooks.

### Phase 10: Game Loop & Logic
- [ ] Sequential game states: Lobby → Countdown → Category → Word Selection → Drawing/Guessing → Round End → Leaderboard → End.
- [ ] Word suggestion engine: Pulling random words from chosen category/tier.
- [ ] **Custom Word List:** Priority override for private rooms.
- [ ] Drawer rotation algorithm and Round timer management.

### Phase 11: Chat & Guess Detection
- [ ] Chat handler with correct-guess detection (case/space insensitive).
- [ ] Drawer chat lock (prevent drawer from helping).
- [ ] Hide correct guesses from other players (broadcast "X guessed the word").
- [ ] Rate-limited chat to prevent spam.

### Phase 12: Scoring & Reputation System
- [ ] Fair Scoring: Word point tier + speed bonus + streak bonus.
- [ ] Behavioral Reputation: Default 10000; decrease on reports; auto-ban below 3000.
- [ ] MVP tracking: Highest scorer per game (excluding private rooms).
- [ ] Animated score update pushes via WebSocket.

### Phase 13: Reporting & Moderation
- [ ] Reporting endpoint: NSFW, Abusive, Illegal, Spam categories.
- [ ] Link reports to reputation penalties.
- [ ] Admin dashboard for confirmed report review.

### Phase 14: Security Hardening & Polish
- [ ] Audit log middleware for all administrative actions.
- [ ] Input sanitization (XSS protection) and secure headers (HSTS, CSP).
- [ ] Production-ready Docker Compose with health checks and Nginx SSL.

### Phase 15: Frontend Integration & Final Test
- [ ] Integration with `web/board-prototype`.
- [ ] Responsive Dashboard and Game UI.
- [ ] Dark/Light mode implementation.
- [ ] Audio manager: Site music and tool sounds.
- [ ] Load testing with `k6` for concurrent rooms.

---

## 3. Concurrency Strategy (Reminder)

- **One goroutine per room:** Isolated execution loops.
- **Channels for communication:** Buffered inboxes for each room to prevent blocking.
- **Stateless WebSockets:** Auth via JWT/Redis tokens, allowing horizontal scaling.

---

## 4. Immediate Next Step

**Phase 3 — Redis & Session Infrastructure.**

We will now add Redis to our infrastructure, build the caching adapter, and prepare the foundation for high-performance sessions and rate limiting.
