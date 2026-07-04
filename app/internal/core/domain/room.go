package domain

import "time"

// RoomType distinguishes public matchmaking rooms from private friend rooms.
type RoomType string

const (
	RoomTypePublic  RoomType = "public"
	RoomTypePrivate RoomType = "private"
)

// RoomState represents the lifecycle of a room.
type RoomState string

const (
	RoomStateLobby    RoomState = "lobby"
	RoomStatePlaying  RoomState = "playing"
	RoomStateFinished RoomState = "finished"
	RoomStateClosed   RoomState = "closed"
)

// Room is a drawing session container.
//
// Why does Room not contain Clients?
//   Clients are a runtime/transport concern. The domain Room only holds the data
//   that needs to be persisted. The WebSocket adapter maps Clients to Rooms.
type Room struct {
	ID           string
	Name         string
	OwnerID      string
	Type         RoomType
	PasswordHash string // empty for public rooms
	Language     string
	State        RoomState
	MaxPlayers   int
	RoundTime    int // seconds
	MaxRounds    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Player is a user participating in a specific room.
//
// It bridges User and Room and tracks in-room state (score, role, connection status).
type Player struct {
	ID        string
	RoomID    string
	UserID    string
	Username  string // denormalized to avoid joins in the hot path
	AvatarURL string
	Score     int64
	IsDrawer  bool
	IsOnline  bool
	JoinedAt  time.Time
}
