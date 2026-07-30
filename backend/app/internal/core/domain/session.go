package domain

import (
	"time"
)

// Session represents a unique user authentication period.
type Session struct {
	ID             string    // Unique session identifier
	UserID         string    // The ID of the owner of this session
	RefreshTokenID string    // The ID of the currently valid refresh token for this session
	IP             string    // Client IP address at session creation
	UserAgent      string    // Client browser/device info
	ExpiresAt      time.Time // Exact moment the session is no longer valid
	CreatedAt      time.Time // When the session was first established
}

// TokenPair is the result of a successful login or refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// IsExpired returns true if the current time has passed the session's expiration.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
