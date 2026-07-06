// Package websocket manages real-time multiplayer connections and event-driven ephemeral rooms.
package websocket

import "time"

// EventType defines the category of a real-time message inside an ephemeral room.
type EventType string

const (
	EventJoin        EventType = "join"
	EventLeave       EventType = "leave"
	EventChat        EventType = "chat"
	EventDraw        EventType = "draw"
	EventClearCanvas EventType = "clear_canvas"
	EventGameState   EventType = "game_state"
)

// MessageEnvelope represents the standard JSON frame sent over WebSocket connections.
type MessageEnvelope struct {
	Type      EventType   `json:"type"`
	Payload   interface{} `json:"payload"`
	Seq       int64       `json:"seq"`
	Timestamp int64       `json:"timestamp"`
}

// Client represents a connected WebSocket player connection.
// It acts as a lightweight transport adapter bridging network frames to the room inbox.
type Client struct {
	ID       string
	UserID   string
	Username string
	Send     chan []byte
}

// RoomEvent is a structured message delivered to an active room's inbox channel.
//
// DESIGN DECISION:
//   By passing all events via channels into the room's single goroutine event loop,
//   we eliminate mutex contention on room state during high-frequency drawing streams.
type RoomEvent struct {
	Type      EventType
	Client    *Client
	Payload   []byte
	Timestamp time.Time
}
