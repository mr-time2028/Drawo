package domain

import "time"

// Achievement represents a badge or accomplishment unlocked by a user.
type Achievement struct {
	ID          string
	UserID      string
	Code        string
	Title       string
	Description string
	UnlockedAt  time.Time
}
