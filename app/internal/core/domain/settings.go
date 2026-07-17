package domain

import "time"

// UserSettings holds persistent user UI/UX customization options.
type UserSettings struct {
	UserID             string
	SoundEnabled       bool
	MusicEnabled       bool
	LanguagePreference string
	Theme              string // "light" or "dark"
	UpdatedAt          time.Time
}
