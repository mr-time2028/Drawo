package domain

import "time"

// GuestAuth is a short-lived, room-bound token issued to anonymous users who
// join a private room via an invite link. It lets them connect to the
// WebSocket and play without creating an account, while still letting the
// server rate-limit/ban/kick them by ID.
//
// Guests can never:
//   - Create rooms (owner must be a registered user).
//   - Join public/ranked matches.
//   - Earn stats or achievements.
type GuestAuth struct {
	Token     string    `json:"token"`
	GuestID   string    `json:"guest_id"`
	RoomID    string    `json:"room_id"`
	Nickname  string    `json:"nickname"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// IsGuest reports whether a client identity string (UserID from AuthContext)
// refers to a guest (as opposed to a registered user). Guest IDs are
// namespaced with a prefix so they can never collide with UUID user IDs.
const GuestIDPrefix = "guest:"

func IsGuestID(id string) bool {
	return len(id) > len(GuestIDPrefix) && id[:len(GuestIDPrefix)] == GuestIDPrefix
}
