# `internal/realtime`

**Responsibility:** Manage real-time WebSocket connections and event-driven ephemeral rooms.

## Phase 8 architecture

Drawo now uses a **per-room goroutine model**:

- `Hub` is a thread-safe registry of active room goroutines.
- `Room.Run` is the only goroutine allowed to mutate that room's client map.
- Each connected socket uses exactly two long-lived goroutines: `readPump` and `writePump`.
- `readPump` handles incoming auth/re-auth/chat/draw/game frames.
- `writePump` serializes all outgoing messages, heartbeat pings, and periodic auth/session checks.
- Room events move through buffered channels instead of shared mutable state.

This prevents the most common WebSocket/game bugs:

- concurrent writes to a WebSocket connection
- concurrent map writes in room state
- slow clients blocking the entire room
- unbounded message memory growth

## WebSocket endpoint

HTTP route:

```text
GET /api/v1/ws
```

The route upgrades to WebSocket. Authentication is performed inside the WebSocket protocol instead of query parameters so access tokens are not exposed in proxy access logs.

## Required client handshake

The client must send exactly this sequence after the socket opens:

### 1. Auth frame

```json
{
  "type": "auth",
  "payload": {
    "access_token": "<short-lived access JWT>"
  }
}
```

Rules:

- Must be the first WebSocket message.
- Must arrive within 5 seconds.
- Must be an **access token**, not a refresh token.
- The token signature, issuer, expiry, and active Redis/cache session are validated.

### 2. Join frame

```json
{
  "type": "join",
  "payload": {
    "room_id": "<room id>"
  }
}
```

Rules:

- Must be the second WebSocket message.
- Must arrive within 10 seconds after auth.
- The room must exist in the Hub or distributed room repository.

After this, normal realtime frames may be sent.

## Message envelope

Every client/server application frame uses:

```json
{
  "type": "chat|draw|game|leave|...",
  "payload": {},
  "seq": 123,
  "timestamp": 1720000000
}
```

`payload` is `json.RawMessage` in Go so drawing/chat/game payloads are forwarded without double encoding.

## Authentication and access-token expiry during gameplay

The access token is validated at WebSocket authentication time and its expiry is tracked by the socket auth context. The active socket is also tied to the Redis/cache session ID.

Important UX + security rule:

> The player is not kicked instantly at the exact access-token expiry moment, but the socket must be re-authenticated with a fresh access token within a short grace period.

Flow:

1. The socket authenticates with a short-lived access token.
2. Before expiry, the server sends `auth_required` with `expires_at` and `grace_until`.
3. The frontend refreshes tokens over normal HTTP using the refresh token.
4. The frontend sends another WebSocket `auth` frame with the new access token.
5. The server verifies the new access token belongs to the same user and same session, then updates the socket auth expiry.
6. If the socket passes `expires_at + grace` without re-auth, the server sends `auth_expired` and closes the connection.

Session revocation is still immediate:

- logout, ban, single-device replacement, or refresh-token reuse detection deletes the session;
- the WebSocket checks the session during incoming traffic and on `writePump`'s auth ticker;
- if the session is gone, the socket disconnects even if the access token has not expired yet.

This prevents a stolen access token from keeping a WebSocket alive indefinitely while still giving real players time to refresh in the background during gameplay.

## Security controls

Implemented in `handler.go` and `auth.go`:

- access tokens are sent in the first WebSocket body frame, not query strings;
- refresh tokens are rejected for WebSocket auth;
- token type confusion is prevented by JWT `typ` claim (`access` vs `refresh`);
- Origin checks block cross-site browser WebSocket abuse;
- max message size is enforced;
- auth and join deadlines prevent idle unauthenticated sockets;
- message rate limiting blocks abusive clients;
- only known event types are accepted;
- room membership is required before chat/draw/game events are dispatched;
- slow clients are disconnected instead of blocking the room;
- access-token expiry is tracked after connection;
- clients can re-authenticate an existing socket with a fresh access token;
- re-auth tokens must match the same user and same session;
- sockets are closed after access expiry plus a short grace period;
- session revocation is detected during incoming traffic and by `writePump`'s auth ticker;
- all writes to a connection are serialized through `writePump`;
- no third per-socket monitor goroutine is needed.

## Supported client event types

- `auth` — first frame for login, later frames for socket re-auth with a fresh access token
- `join` — second frame only
- `leave`
- `chat`
- `draw`
- `game`
- `clear_canvas`

Later phases can expand `chat`, `draw`, and `game` payload schemas without changing the transport envelope.
