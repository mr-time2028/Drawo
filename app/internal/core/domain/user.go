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
