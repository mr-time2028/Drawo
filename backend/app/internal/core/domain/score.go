package domain

// Score records the points earned by a player during a completed game.
type Score struct {
	ID            string
	GameHistoryID string
	UserID        string
	Points        int64
	Rank          int
}
