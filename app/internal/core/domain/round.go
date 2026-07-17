package domain

import "time"

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
