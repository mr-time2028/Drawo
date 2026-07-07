package domain

import "time"

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
