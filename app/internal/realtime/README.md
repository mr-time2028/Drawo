# `internal/realtime`

**Responsibility:** Manage Drawo's realtime WebSocket runtime: secure socket authentication, per-room goroutines, room events, and the drawing protocol.

## Realtime architecture recap

Drawo uses a **per-room goroutine model**:

- `Hub` is a thread-safe registry of active room goroutines.
- `Room.Run` is the only goroutine allowed to mutate that room's client map and canvas history.
- Each connected socket uses exactly two long-lived goroutines: `readPump` and `writePump`.
- `readPump` handles incoming auth/re-auth/chat/draw/game frames.
- `writePump` serializes all outgoing messages, heartbeat pings, and periodic auth/session checks.
- Room events move through buffered channels instead of shared mutable state.

This prevents common WebSocket/game bugs: concurrent writes, concurrent map writes, slow clients blocking a room, and unbounded message growth.

## WebSocket endpoint

Registered by `internal/delivery/websocket`:

```text
GET /api/v1/ws
```

The route starts as HTTP Upgrade, then becomes Drawo's realtime protocol. Access tokens are sent in message bodies, not query params, to avoid token leakage in logs.

## Required client handshake

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
- Must be an access token, not a refresh token.
- Signature, issuer, expiry, token type, and active session are validated.

### 2. Join frame

Public matchmaking is backend-owned. The frontend does **not** need to know a public `room_id`.

For normal public matchmaking, send:

```json
{
  "type": "join",
  "payload": {
    "mode": "public",
    "language": "en"
  }
}
```

or an empty payload:

```json
{
  "type": "join",
  "payload": {}
}
```

The backend finds a non-full public room or creates one.

For private rooms, send an invite code instead of an internal room ID:

```json
{
  "type": "join",
  "payload": {
    "mode": "private",
    "invite_code": "AB12CD"
  }
}
```

`room_id` is still supported only for reconnect/admin/debug flows:

```json
{
  "type": "join",
  "payload": {
    "room_id": "existing-room-id"
  }
}
```

Rules:

- Must be the second WebSocket message.
- Must arrive within 10 seconds after auth.
- Public clients should use backend matchmaking, not choose room IDs themselves.

## WebSocket re-authentication during games

The socket stores the access token expiry in `AuthContext.AccessExpiresAt`.

Before expiry, server sends:

```json
{
  "type": "auth_required",
  "payload": {
    "expires_at": 1710000000
  }
}
```

Frontend should refresh over HTTP, then re-auth the same socket:

```json
{
  "type": "auth",
  "payload": {
    "access_token": "<new access token>"
  }
}
```

The server accepts re-auth only when the new token belongs to the same `user_id` and same `session_id`. If the access token reaches `expires_at` without re-auth, it closes with `auth_expired`. There is no post-expiry grace window; the two-minute warning window is the time reserved for the frontend to refresh in the background.

## Drawing protocol

The drawing protocol uses **operations**, not images. Clients send compact drawing commands and every frontend reconstructs the same canvas by applying the ordered operation stream.

All drawing frames use:

```json
{
  "type": "draw",
  "seq": 123,
  "payload": {
    "op": "stroke"
  }
}
```

The room adds server metadata before broadcast:

```json
{
  "id": "room-1-42",
  "user_id": "drawer-user-id",
  "server_seq": 42,
  "timestamp": 1710000000000
}
```

### Stroke

```json
{
  "type": "draw",
  "payload": {
    "op": "stroke",
    "tool": "pencil",
    "color": "#000000",
    "size": 4,
    "points": [
      {"x": 10, "y": 20},
      {"x": 12, "y": 22}
    ]
  }
}
```

Supported stroke tools:

```text
pencil, brush, marker, calligraphy
```

### Erase

```json
{
  "type": "draw",
  "payload": {
    "op": "erase",
    "size": 20,
    "points": [
      {"x": 10, "y": 20}
    ]
  }
}
```

### Shape

```json
{
  "type": "draw",
  "payload": {
    "op": "shape",
    "shape": "rectangle",
    "color": "#ff0000",
    "x": 10,
    "y": 20,
    "width": 100,
    "height": 80,
    "filled": false
  }
}
```

Supported shapes:

```text
line, rectangle, ellipse, triangle
```

### Fill

```json
{
  "type": "draw",
  "payload": {
    "op": "fill",
    "color": "#00ff00",
    "x": 50,
    "y": 50
  }
}
```

### Clear

```json
{
  "type": "draw",
  "payload": {
    "op": "clear"
  }
}
```

### Undo / redo

```json
{"type":"draw","payload":{"op":"undo"}}
```

```json
{"type":"draw","payload":{"op":"redo"}}
```

Undo removes the latest canvas operation from the same user and stores it in that user's redo stack. Redo restores the latest undone operation. Each client has at most 100 redo entries.

## Canvas sync for late joiners

When a client joins, the room first sends:

```json
{
  "type": "canvas_sync",
  "payload": {
    "operations": [],
    "server_seq": 42
  }
}
```

The client applies `operations` in order to reconstruct the current canvas.

## Drawing validation and anti-abuse limits

Validation happens before a draw operation is accepted:

- max canvas coordinate: `4096`
- max points per stroke/erase op: `256`
- max brush/eraser size: `64`
- max canvas history operations kept in memory: `2000`
- max redo entries per client: `100`
- max draw operations per second per client: `30`
- max fill operations per second per client: `8`
- max clear operations per minute per client: `3`
- colors must be `#RRGGBB`
- unknown tools, shapes, and operations are rejected

Invalid drawing operations send:

```json
{
  "type": "error",
  "payload": {
    "code": "invalid_draw",
    "message": "..."
  }
}
```

Rate-limited drawing sends:

```json
{
  "type": "error",
  "payload": {
    "code": "draw_rate_limited",
    "message": "..."
  }
}
```

## Drawer-only hook for the game loop

`domain.Room` now has:

```go
CurrentDrawerID string
```

When it is empty, drawing is allowed for prototype/lobby behavior. The game loop will set this field each turn. Once set, only that user can draw; others receive:

```json
{
  "type": "error",
  "payload": {
    "code": "draw_forbidden",
    "message": "only the current drawer can draw"
  }
}
```

## Security controls

- Access tokens are sent in WebSocket body frames, not query params.
- Refresh tokens are rejected for WebSocket auth.
- JWT `typ` claim prevents token confusion.
- Origin checks block cross-site WebSocket abuse.
- Message size and message rate are limited.
- Auth and join deadlines prevent idle unauthenticated sockets.
- Room membership is required before draw/chat/game events.
- Access-token expiry is tracked after connection.
- Re-auth tokens must match the same user and session.
- Session revocation is detected during incoming traffic and by `writePump`'s auth ticker.
- All writes are serialized through `writePump`.
- No third per-socket monitor goroutine is used.

## Game loop

Rooms now contain a lightweight game state machine. The room goroutine owns all
state transitions, so timers, drawer rotation, score updates, and canvas changes
happen serially without locks.

Runtime states:

```text
waiting_for_players
countdown
word_selection
drawing
round_end
leaderboard
game_end
```

Flow:

1. Players join a backend-matched room.
2. The room waits until `MinPlayers` are online.
3. The room starts a countdown.
4. The room rotates to the next online drawer.
5. The drawer receives private word suggestions.
6. If the drawer does not choose in time, the room auto-selects the first word.
7. The room enters drawing mode.
8. Non-drawers chat to guess; drawer chat is blocked.
9. Correct guesses are hidden and broadcast as system messages.
10. The round ends when time expires or all active guessers are correct.
11. The room shows round results, leaderboard, then starts next round or ends.

## Word language and translation model

Rooms carry a `Language` field, normally `en` or `fa`.

Public matchmaking uses the requested language:

```json
{"type":"join","payload":{"mode":"public","language":"fa"}}
```

Word suggestions are requested in the room language. Content words are grouped by
`group_id`, so the same concept can exist in multiple languages:

```text
group_id: apple
en: apple
fa: سیب
```

The room stores the chosen word text in the active room language. Guess matching
uses `NormalizeGuess(text, language)` so English and Persian guesses are compared
fairly.

English normalization:

- lowercases text
- removes whitespace
- removes punctuation/symbols

Persian normalization:

- normalizes Arabic ي/ك variants to Persian ی/ک
- removes Arabic diacritics
- removes zero-width non-joiner
- removes whitespace and punctuation

## Chat and guessing

During drawing:

- the drawer cannot use normal chat;
- non-drawers can chat/guess;
- a correct guess is not broadcast as raw text;
- instead, the room sends a system message such as `Bob guessed the word!`;
- a player who already guessed correctly cannot keep guessing for more points.

## Reputation behavior

The room records reputation events during play and flushes them to profiles when
the game ends or the empty room closes.

Positive behavior:

- completing a game increases reputation;
- a game without reports gives a small bonus;
- correct guesses give a small capped bonus;
- useful drawing that gets guessed gives the drawer a small capped bonus.

Bad behavior:

- abandoning an active game decreases reputation;
- abandoning while drawer has a larger penalty;
- disconnects are marked with a reconnect deadline so reconnect support can keep
  a player's slot temporarily instead of punishing instantly.

Positive reputation gain is capped per game to reduce farming.

## Player list events

The room now sends frontend-friendly player events in addition to full
`game_state` snapshots:

```text
player_joined
player_left
player_reconnected
```

Each event uses:

```json
{
  "user_id": "user-id",
  "username": "Alice",
  "reconnect_deadline": 1710000000
}
```

The frontend should use these events for small UI updates, and use `game_state`
as the source of truth after reconnects or state changes.

## Reconnect flow

The frontend can reconnect without knowing a room ID:

```json
{
  "type": "join",
  "payload": {
    "mode": "reconnect"
  }
}
```

The Hub stores an active room mapping for each user while the room exists. If the
room still exists and the reconnect window has not expired, the socket is
reattached to the same player state.

Disconnect behavior:

- player is marked offline;
- a reconnect deadline is recorded;
- the player slot and score remain available during the reconnect window;
- if the user does not return, reputation penalties are applied;
- if the drawer disconnects during drawing, the round is paused briefly and then
  ends if the drawer does not return.

## Drawer disconnect handling

If the current drawer disconnects during drawing:

1. the room enters `drawer_disconnected`;
2. the drawing timer pauses;
3. the room waits for the drawer reconnect window;
4. if the drawer reconnects, drawing resumes with the remaining time;
5. if the drawer does not reconnect, the round ends and reputation penalty is recorded.

This avoids instantly punishing short network drops while still discouraging
players from abandoning active games.

## Reputation audit events

The current profile score is stored on `profiles.reputation_score`. Each change
is also written as a `reputation_events` audit row when a repository is available.

Examples of event reasons:

```text
completed_game
no_reports
correct_guess
successful_drawing
abandoned_active_game
```

This makes reputation changes explainable for moderation/admin tooling.
