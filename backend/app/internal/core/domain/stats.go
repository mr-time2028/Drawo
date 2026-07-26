package domain

import "time"

// PlayerStatistic tracks lifetime competitive numbers for a user.
// Ranked/public dictionary games should update these global stats. Private or
// custom-word games can remain in history without changing these counters.
type PlayerStatistic struct {
	UserID             string
	TotalGames         int64
	RankedGames        int64
	PrivateGames       int64
	TotalWins          int64
	TotalScore         int64
	BestGameScore      int64
	MVPs               int64
	TotalDrawings      int64
	SuccessfulDrawings int64
	CorrectGuesses     int64
	GamesAbandoned     int64
	ReportsReceived    int64
	ReportsConfirmed   int64
	UpdatedAt          time.Time
}
