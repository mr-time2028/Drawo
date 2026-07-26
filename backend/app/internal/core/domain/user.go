// Package domain contains pure business entities.
//
// These structs have NO dependencies on Gin, GORM, Redis, or any framework.
// They represent the business rules of Drawo and are used by all layers.
package domain

import "time"

// AccountStatus explains why an account can or cannot authenticate. IsActive is
// kept for fast/legacy checks, while Status gives the precise moderation state.
type AccountStatus string

const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusBanned    AccountStatus = "banned"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusDeleted   AccountStatus = "deleted"
)

// User is the central account entity.
//
// Why separate User from Profile?
//
//	Authentication fields rarely change, while profile fields (avatar, settings)
//	change frequently. Splitting them reduces lock contention and keeps the
//	auth path lightweight. They share the same ID (1:1 relationship).
type User struct {
	ID           string
	Username     string
	PasswordHash string
	IsActive     bool
	Status       AccountStatus
	IsSuperuser  bool
	BanCount     int        // How many times this user has been banned
	BannedAt     *time.Time // Timestamp of the most recent ban
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
