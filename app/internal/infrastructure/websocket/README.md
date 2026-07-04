# internal/infrastructure/websocket

**Responsibility:** Manage WebSocket connections and room coordination.

This package will be implemented in Phase 9. It will contain:

- A scalable room registry.
- One goroutine per room with a serial event loop.
- A typed WebSocket message protocol for drawing, chat, and game events.

**Concurrency model:** Hub routes clients to rooms; each room processes its
own inbox serially. This avoids a global lock and lets rooms scale independently.
