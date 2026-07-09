// Package domain contains the pure business entities of the Drawo application.
// This package is the "Heart" of the system and has no external dependencies.
package domain

import (
	"time"
)

// Session represents a unique user authentication period.
//
// DESIGN DECISION:
//   While Drawo uses JWTs for credentials, the Session entity lives in Redis to
//   provide a "Source of Truth" for active connections. This allows features like:
//   - Revoking all sessions when a user changes their password.
//   - Tracking "Last Seen" timestamps for friend lists.
//   - Enforcing a maximum number of simultaneous devices.
type Session struct {
	ID        string    // Unique session identifier (typically a cryptographically secure random string)
	UserID    string    // The ID of the owner of this session
	IP        string    // Client IP address at session creation
	UserAgent string    // Client browser/device info
	ExpiresAt time.Time // Exact moment the session is no longer valid
	CreatedAt time.Time // When the session was first established
}

// IsExpired returns true if the current time has passed the session's expiration.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
