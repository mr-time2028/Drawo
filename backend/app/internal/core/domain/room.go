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

// WordSource controls which dictionary the drawing words are drawn from.
type WordSource string

const (
	WordSourceDefault  WordSource = "default"  // All enabled categories, current game logic.
	WordSourceCategory WordSource = "category" // Restricted to a single CategoryID.
	WordSourceCustom   WordSource = "custom"   // Only the room owner's custom words.
)

// Room-level gameplay bounds (used for validation of owner-supplied settings).
const (
	RoomMinPlayers     = 2
	RoomMaxPlayers     = 12
	RoomDefaultPlayers = 8

	RoomMinRounds     = 1
	RoomMaxRounds     = 10
	RoomDefaultRounds = 3

	RoomMinRoundTime     = 30
	RoomMaxRoundTime     = 180
	RoomRoundTimeStep    = 10
	RoomDefaultRoundTime = 80

	RoomMinCustomWords = 5
	RoomMaxCustomWords = 200

	RoomMaxNameLength = 50
	RoomMinNameLength = 3
	RoomInviteCodeLen = 6
)

// Points tiers for custom words (mirrors Word.Points).
const (
	WordPointsEasy   = 1
	WordPointsMedium = 2
	WordPointsHard   = 3
)

// CustomWord is an owner-supplied word with optional point difficulty.
type CustomWord struct {
	Text   string `json:"text"`
	Points int    `json:"points"`
}

// Room represents an ephemeral, in-memory drawing game session.
//
// DESIGN DECISION:
//
//	Rooms (both public and private) are treated strictly as ephemeral runtime objects rather
//	than persistent relational database entities. A room only exists while players are actively
//	using it. It is created when the first player creates or joins it, maintained entirely in
//	memory during gameplay, and automatically destroyed when the game ends or all players leave.
//
//	Private rooms generate a unique InviteCode that remains valid only during the room's active
//	lifecycle and is automatically invalidated upon room destruction.
//
//	When scaled across multiple instances, discovery and InviteCode lookups are coordinated
//	via distributed caching (Redis), while runtime state remains exclusively in memory on the
//	owning game server instance.
type Room struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	InviteCode       string           `json:"invite_code,omitempty"` // Unique invitation code for private rooms.
	OwnerID          string           `json:"owner_id"`
	Type             RoomType         `json:"type"`
	// PasswordHash is persisted to Redis so join-after-restart / cross-instance
	// joins can verify the password. It is scrubbed from HTTP responses by
	// controllers (roomWithMeta omits it) and by the public GetByCode endpoint.
	PasswordHash     string           `json:"password_hash,omitempty"`
	HasPassword      bool             `json:"has_password"`
	Language         string           `json:"language"`
	WordSource       WordSource       `json:"word_source"`
	CategoryID       string           `json:"category_id,omitempty"` // Used when WordSource == WordSourceCategory.
	State            RoomState        `json:"state"`
	MinPlayers       int              `json:"min_players"`
	MaxPlayers       int              `json:"max_players"`
	RoundTime        int              `json:"round_time"` // Seconds per drawing turn.
	MaxRounds        int              `json:"max_rounds"`
	CurrentRound     int              `json:"current_round,omitempty"`
	CurrentDrawerID  string           `json:"current_drawer_id,omitempty"`
	CustomCategories []CustomCategory `json:"custom_categories,omitempty"` // Used when WordSource == WordSourceCustom.
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// CustomCategory is an owner-supplied category for WordSourceCustom rooms.
// Words is keyed by point tier (1=easy, 2=medium, 3=hard), each value is a
// list of words in that difficulty bucket.
type CustomCategory struct {
	Name  string            `json:"name"`
	Words map[int][]string  `json:"words"`
}

// Player represents a user participating in a specific ephemeral room.
// It bridges User and Room and tracks transient in-room state (score, role, online presence).
type Player struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"` // Denormalized to avoid joins during high-frequency gameplay.
	AvatarURL string    `json:"avatar_url"`
	Score     int64     `json:"score"`
	IsDrawer  bool      `json:"is_drawer"`
	IsOnline  bool      `json:"is_online"`
	IsOwner   bool      `json:"is_owner"`
	JoinedAt  time.Time `json:"joined_at"`
}

