package domain

import "time"

// ReputationEvent is an immutable audit record explaining why a user's
// reputation changed. Keeping these records makes moderation decisions
// traceable instead of silently changing a score.
type ReputationEvent struct {
	ID        string
	UserID    string
	Delta     int64
	Reason    string
	RoomID    string
	Round     int
	CreatedAt time.Time
}
