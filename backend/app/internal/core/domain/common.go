package domain

import "time"

// Paging is a reusable pagination request value object.
type Paging struct {
	Limit  int
	Offset int
}

// PageOf wraps a paginated result with total count.
type PageOf[T any] struct {
	Items  []T
	Total  int64
	Limit  int
	Offset int
}

// AuditLog records security-relevant actions.
type AuditLog struct {
	ID        string
	UserID    *string
	Action    string
	Details   string
	IP        string
	UserAgent string
	CreatedAt time.Time
}
