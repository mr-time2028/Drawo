// Package domain contains pure business entities.
package domain

import "time"

// Friendship represents an accepted friend relationship between two accounts stored in the relational DB.
type Friendship struct {
	UserID    string
	FriendID  string
	CreatedAt time.Time
}

// FriendRequestStatus defines the status of a friend request.
type FriendRequestStatus string

const (
	FriendRequestPending  FriendRequestStatus = "pending"
	FriendRequestAccepted FriendRequestStatus = "accepted"
	FriendRequestRejected FriendRequestStatus = "rejected"
)

// FriendRequest represents a pending or resolved invitation to connect.
type FriendRequest struct {
	ID        string
	FromID    string
	ToID      string
	Status    FriendRequestStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GameHistory records the persistent historical summary of a completed game session.
// Note: While gameplay occurs entirely in ephemeral room memory, final summaries are persisted here.
type GameHistory struct {
	ID        string
	RoomID    string
	RoomName  string
	Language  string
	WinnerID  string
	StartedAt time.Time
	EndedAt   time.Time
}

// Round records historical details about a single drawing round in a completed game.
type Round struct {
	ID            string
	GameHistoryID string
	RoundNumber   int
	DrawerID      string
	Word          string
	StartedAt     time.Time
	EndedAt       time.Time
}

// Score records the points earned by a player during a completed game.
type Score struct {
	ID            string
	GameHistoryID string
	UserID        string
	Points        int64
	Rank          int
}

// ReportReason represents the category of a moderation report.
type ReportReason string

const (
	ReportReasonInappropriateDrawing ReportReason = "inappropriate_drawing"
	ReportReasonAbusiveChat          ReportReason = "abusive_chat"
	ReportReasonCheating             ReportReason = "cheating"
	ReportReasonGriefing             ReportReason = "griefing"
)

// Report records user-submitted moderation alerts.
type Report struct {
	ID         string
	ReporterID string
	ReportedID string
	RoomID     string
	Reason     ReportReason
	Details    string
	CreatedAt  time.Time
}

// Achievement represents a badge or accomplishment unlocked by a user.
type Achievement struct {
	ID          string
	UserID      string
	Code        string
	Title       string
	Description string
	UnlockedAt  time.Time
}

// PlayerStatistic tracks lifetime competitive numbers for a user.
type PlayerStatistic struct {
	UserID         string
	TotalGames     int64
	TotalWins      int64
	TotalDrawings  int64
	CorrectGuesses int64
	UpdatedAt      time.Time
}

// UserSettings holds persistent user UI/UX customization options.
type UserSettings struct {
	UserID             string
	SoundEnabled       bool
	MusicEnabled       bool
	LanguagePreference string
	Theme              string // "light" or "dark"
	UpdatedAt          time.Time
}
