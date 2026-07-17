package domain

import "time"

// PlayerStatistic tracks lifetime competitive numbers for a user.
type PlayerStatistic struct {
	UserID         string
	TotalGames     int64
	TotalWins      int64
	TotalDrawings  int64
	CorrectGuesses int64
	UpdatedAt      time.Time
}
