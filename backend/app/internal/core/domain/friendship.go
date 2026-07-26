package domain

import "time"

// Friendship represents an accepted friend relationship between two accounts stored in the relational DB.
type Friendship struct {
	UserID    string
	FriendID  string
	CreatedAt time.Time
}
