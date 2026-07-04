// Package domain contains pure business entities.
//
// These structs have NO dependencies on Gin, GORM, Redis, or any framework.
// They represent the business rules of Drawo and are used by all layers.
package domain

import "time"

// User is the central account entity.
//
// Why separate User from Profile?
//   Authentication fields rarely change, while profile fields (avatar, settings)
//   change frequently. Splitting them reduces lock contention and keeps the
//   auth path lightweight. They share the same ID (1:1 relationship).
type User struct {
	ID          string
	Username    string
	PasswordHash string
	IsActive    bool
	IsSuperuser bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Profile holds user preferences and statistics.
//
// All numeric scores use int64 to avoid overflow during high-volume play.
type Profile struct {
	UserID           string
	AvatarURL        string
	Email            string
	Phone            string
	EmailVerified    bool
	PhoneVerified    bool
	Locale           string
	Theme            string // "light" or "dark"
	BackgroundSound  bool
	ToolSound        bool
	WordScore        int64
	ReputationScore  int64
	GamesPlayed      int64
	MVPs             int64
	Rank             string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// UserWithProfile joins an account with its profile for dashboard responses.
type UserWithProfile struct {
	User    User
	Profile Profile
}
