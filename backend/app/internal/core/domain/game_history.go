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
