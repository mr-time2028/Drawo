// Package websocket manages real-time multiplayer connections and event-driven ephemeral rooms.
package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// EventType defines the category of a real-time message inside an ephemeral room.
type EventType string

const (
	// Client-to-server protocol events.
	EventAuth  EventType = "auth"  // First frame authenticates; later frames re-auth/update access token.
	EventJoin  EventType = "join"  // Join an ephemeral room after authentication.
	EventLeave EventType = "leave" // Leave current room gracefully.
	EventChat  EventType = "chat"  // Chat/guess message. Guess rules are handled by game logic.
	EventDraw  EventType = "draw"  // Drawing operation payload.
	EventGame  EventType = "game"  // Generic game-state command/event namespace.

	// Server-to-client protocol events.
	EventAuthOK            EventType = "auth_ok"
	EventAuthRequired      EventType = "auth_required" // Server asks client to refresh access token in background.
	EventCanvasSync        EventType = "canvas_sync"   // Server sends current round drawing history to a joiner.
	EventJoined            EventType = "joined"
	EventPlayerJoined      EventType = "player_joined"
	EventPlayerLeft        EventType = "player_left"
	EventPlayerReconnected EventType = "player_reconnected"
	EventError             EventType = "error"
	EventGameState         EventType = "game_state"
	EventClearCanvas       EventType = "clear_canvas"
)

// MessageEnvelope is the standard JSON frame sent in both directions.
//
// Security/design notes:
//   - All application frames must use this envelope. We never accept arbitrary
//     untyped JSON because it makes validation and rate-limiting ambiguous.
//   - Payload is json.RawMessage so the room loop can forward draw/chat/game
//     payloads without double-encoding or base64-encoding []byte.
//   - Seq is client-controlled and echoed for client-side ordering/debugging;
//     the server does not trust it for authorization.
type MessageEnvelope struct {
	Type      EventType       `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Seq       int64           `json:"seq,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
}

// AuthPayload is used for both the first auth frame and later re-auth frames.
// Tokens are sent inside the WebSocket body instead of query params to avoid
// leaking credentials in access logs.
//
// A client can authenticate either as a registered user (AccessToken) or as an
// anonymous guest (GuestToken, issued by POST /rooms/by-code/:code/join). A
// guest token is short-lived and room-bound: the server will reject JoinPayload
// that targets a different room.
type AuthPayload struct {
	AccessToken string `json:"access_token"`
	GuestToken  string `json:"guest_token"`
}

// AuthOKPayload confirms initial authentication or socket re-authentication.
type AuthOKPayload struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	ExpiresAt int64  `json:"expires_at"`
}

// JoinPayload asks the backend to place this socket into a room.
//
// Public matchmaking is backend-owned: clients normally send an empty payload or
// {"mode":"public","language":"en"}, and the Hub chooses/creates the room.
// room_id is optional and exists for reconnect/admin/debug flows. Private joins
// should use invite_code instead of exposing internal room IDs to the frontend.
type JoinPayload struct {
	Mode       string `json:"mode,omitempty"`        // "public" (default) or "private".
	Language   string `json:"language,omitempty"`    // Matchmaking language, e.g. "en" or "fa".
	CategoryID string `json:"category_id,omitempty"` // Optional dictionary category preference.
	RoomID     string `json:"room_id,omitempty"`     // Optional direct join/reconnect/debug path.
	InviteCode string `json:"invite_code,omitempty"` // Private room lookup path.
}

// ErrorPayload is sent before closing, or as a non-fatal validation error.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// AuthRequiredPayload tells the client to refresh its HTTP tokens in the
// background and then send a new WebSocket auth frame before ExpiresAt.
type AuthRequiredPayload struct {
	ExpiresAt int64 `json:"expires_at"`
}

// AuthContext is the trusted identity attached to a WebSocket after auth.
//
// It is mutable only for re-authentication (registered users can refresh HTTP
// tokens while still in the game). The mutex prevents races between the read
// pump updating expiry and the session monitor checking expiry.
//
// IsGuest distinguishes anonymous players (joined via invite link without an
// account) from registered users. Guests have no session and their identity
// expires with the guest token.
type AuthContext struct {
	mu              sync.RWMutex
	UserID          string // registered user ID OR "guest:<uuid>" for guests
	SessionID       string // empty for guests
	TokenID         string
	AccessExpiresAt time.Time // JWT expiry for registered users; guest token expiry for guests
	IsGuest         bool
	Nickname        string // guest display name (empty for registered users)
	RoomID          string // guests are bound to one room
}

func (a *AuthContext) Snapshot() (userID, sessionID, tokenID string, accessExpiresAt time.Time) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.UserID, a.SessionID, a.TokenID, a.AccessExpiresAt
}

func (a *AuthContext) AccessExpiresAtValue() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.AccessExpiresAt
}

func (a *AuthContext) AccessValid(now time.Time) bool {
	return now.Before(a.AccessExpiresAtValue())
}

func (a *AuthContext) UpdateFrom(next *AuthContext) {
	if next == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	// Registered users can only re-auth as themselves; guests cannot re-auth
	// (they just reconnect if needed).
	if next.IsGuest || a.IsGuest {
		return
	}
	a.TokenID = next.TokenID
	a.AccessExpiresAt = next.AccessExpiresAt
}

// GuestSnapshot returns the guest-specific fields (only meaningful when IsGuest).
func (a *AuthContext) GuestSnapshot() (guestID, nickname, roomID string, expiresAt time.Time) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.UserID, a.Nickname, a.RoomID, a.AccessExpiresAt
}

// Client represents a connected WebSocket player connection.
//
// All writes to Conn happen only in writePump. All room membership mutations are
// serialized by Room.Run. This avoids the common WebSocket race bugs: concurrent
// writes, map races, and room-state mutations from many goroutines.
type Client struct {
	ID        string
	UserID    string
	SessionID string
	Username  string
	RoomID    string
	Conn      *websocket.Conn
	Send      chan []byte
	Done      chan struct{} // closed when the client should be considered disconnected; signals broadcast loops to drop it.
}

// RoomEvent is delivered to an active room's single goroutine inbox.
//
// DESIGN DECISION:
// Passing all events via channels into the room's single goroutine event loop
// eliminates mutex contention and prevents concurrent map/state mutation bugs on
// the hot path for drawing/chat/game traffic.
type RoomEvent struct {
	Type      EventType
	Client    *Client
	Payload   json.RawMessage
	Seq       int64
	Timestamp time.Time
}
