# `internal/delivery/websocket`

**Responsibility:** Expose Drawo's realtime protocol over WebSocket.

WebSocket starts as an HTTP Upgrade request, but after the upgrade it is not a REST/HTTP controller anymore. This package keeps realtime delivery separate from `internal/delivery/http` so routes and controllers are easier to find.

Contents:

- `controller.go` — Thin Gin adapter that upgrades the request by delegating to `internal/realtime.Handler`.
- `routes.go` — Registers `GET /api/v1/ws`.

The actual realtime protocol, authentication/re-authentication, room goroutines, read/write pumps, and message validation live in:

```text
internal/realtime
```

This package should stay thin and should not contain game logic.
