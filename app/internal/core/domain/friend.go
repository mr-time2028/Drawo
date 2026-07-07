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
