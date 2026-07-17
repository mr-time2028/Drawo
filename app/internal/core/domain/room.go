package domain

import "time"

// RoomType distinguishes public matchmaking rooms from private friend rooms.
type RoomType string

const (
	RoomTypePublic  RoomType = "public"
	RoomTypePrivate RoomType = "private"
)

// RoomState represents the runtime lifecycle state of an ephemeral room.
type RoomState string

const (
	RoomStateLobby    RoomState = "lobby"
	RoomStatePlaying  RoomState = "playing"
	RoomStateFinished RoomState = "finished"
	RoomStateClosed   RoomState = "closed"
)

// Room represents an ephemeral, in-memory drawing game session.
//
// DESIGN DECISION:
//   Rooms (both public and private) are treated strictly as ephemeral runtime objects rather
//   than persistent relational database entities. A room only exists while players are actively
//   using it. It is created when the first player creates or joins it, maintained entirely in
//   memory during gameplay, and automatically destroyed when the game ends or all players leave.
//
//   Private rooms generate a unique InviteCode that remains valid only during the room's active
//   lifecycle and is automatically invalidated upon room destruction.
//
//   When scaled across multiple instances, discovery and InviteCode lookups are coordinated
//   via distributed caching (Redis), while runtime state remains exclusively in memory on the
//   owning game server instance.
type Room struct {
	ID           string
	Name         string
	InviteCode   string // Unique invitation code for private rooms; invalidated when room closes
	OwnerID      string
	Type         RoomType
	PasswordHash string // Empty for public rooms
	Language     string
	State        RoomState
	MinPlayers   int
	MaxPlayers   int
	RoundTime    int // Seconds per drawing turn
	MaxRounds    int
	CurrentRound int
	CustomWords  []string // Custom word list for private rooms; overrides default dictionaries
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Player represents a user participating in a specific ephemeral room.
// It bridges User and Room and tracks transient in-room state (score, role, online presence).
type Player struct {
	ID        string
	RoomID    string
	UserID    string
	Username  string // Denormalized to avoid joins during high-frequency gameplay
	AvatarURL string
	Score     int64
	IsDrawer  bool
	IsOnline  bool
	JoinedAt  time.Time
}
