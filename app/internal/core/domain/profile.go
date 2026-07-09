package domain

import "time"

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
