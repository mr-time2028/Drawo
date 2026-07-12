package domain

import "time"

// SongType distinguishes between landing page music and in-game music.
type SongType string

const (
	SongTypeLanding SongType = "landing"
	SongTypeGame    SongType = "game"
)

// Song represents a music file managed by the admin.
type Song struct {
	ID        string
	Title     string
	FileKey   string   // The object key in MinIO/S3
	Type      SongType
	IsActive  bool     // Whether this song should be included in the playlist
	CreatedAt time.Time
}

// GlobalSetting represents a key-value configuration pair stored in the database.
type GlobalSetting struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}
