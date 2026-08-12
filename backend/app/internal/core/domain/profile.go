package domain

import "time"

// Profile holds user preferences and statistics.
//
// All numeric scores use int64 to avoid overflow during high-volume play.
type Profile struct {
	UserID          string    `json:"user_id"`
	AvatarURL       string    `json:"avatar_url"`
	Email           string    `json:"email"`
	Phone           string    `json:"phone"`
	EmailVerified   bool      `json:"email_verified"`
	PhoneVerified   bool      `json:"phone_verified"`
	Locale          string    `json:"locale"` // Preferred UI / matchmaking language ("en" or "fa").
	BackgroundSound bool      `json:"background_sound"`
	ToolSound       bool      `json:"tool_sound"`
	WordScore       int64     `json:"word_score"`
	ReputationScore int64     `json:"reputation_score"`
	GamesPlayed     int64     `json:"games_played"`
	MVPs            int64     `json:"mvps" gorm:"column:mvps"`
	Rank            string    `json:"rank"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UserWithProfile joins an account with its profile for dashboard responses.
type UserWithProfile struct {
	User    User    `json:"user"`
	Profile Profile `json:"profile"`
}
