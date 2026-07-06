# `internal/infrastructure/websocket`

**Responsibility:** Manage real-time WebSocket connections and event-driven ephemeral rooms.

### Architectural Rules
1. **Ephemeral Rooms:** Public and private game rooms are treated strictly as ephemeral runtime objects in memory. They exist only while active players are participating and are never persisted in relational databases (`GORM`).
2. **Invite Codes:** Private rooms generate unique invite codes valid solely during the room's active lifecycle. When a room terminates or all players leave, the invite code is automatically invalidated.
3. **Distributed Discovery:** When running across multiple instances, `ports.RoomRepository` coordinates discovery and invite code lookups via non-relational cache (`Redis`). However, runtime game state (canvas drawing operations, chat, round timers) lives exclusively inside the goroutine memory of the instance owning the room.

### Concurrency Strategy
- Each room runs a dedicated goroutine (`Room.Run`) reading serially from a buffered channel (`inbox chan *RoomEvent`).
- Because all state mutations occur serially inside the room's event loop, hot-path operations like drawing strokes operate with **zero mutex lock contention**.
