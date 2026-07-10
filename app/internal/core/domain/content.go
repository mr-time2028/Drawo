package domain

import "time"

// Category groups words for selection.
type Category struct {
	ID        string
	Name      string
	Language  string
	GroupID   string // Links "Animals" (en) and "حیوانات" (fa)
	CreatedAt time.Time
}

// Word is the actual vocabulary item.
type Word struct {
	ID         string
	CategoryID string
	GroupID    string // Links "Apple" (en) and "سیب" (fa)
	Text       string
	Language   string
	Points     int
	CreatedAt  time.Time
}

// BadWord represents a blacklisted term for chat filtering.
type BadWord struct {
	ID        string
	Text      string
	Language  string
	CreatedAt time.Time
}
