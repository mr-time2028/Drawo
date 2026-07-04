// Package ports defines interfaces (ports) that the application layer depends on.
//
// In Clean Architecture / Hexagonal terms, these are "driven ports".
// They are implemented by adapters in internal/repositories and internal/infrastructure.
//
// Why put interfaces here?
//   The application layer imports ports, not concrete implementations.
//   This lets us swap relational databases (e.g., PostgreSQL for MySQL/SQLite)
//   or caching/non-relational stores (e.g., Redis for an in-memory solution)
//   or use mock repositories in tests without changing business logic.
package ports

import (
	"context"
	"time"

	"drawo/internal/core/domain"
)

// UserRepository defines persistence operations for accounts.
type UserRepository interface {
	Insert(user *domain.User) error
	GetByID(id string) (*domain.User, error)
	GetByUsername(username string) (*domain.User, error)
	Exists(username string) (bool, error)
	Update(user *domain.User) error
}

// ProfileRepository defines persistence operations for user profiles.
type ProfileRepository interface {
	Insert(profile *domain.Profile) error
	GetByUserID(userID string) (*domain.Profile, error)
	Update(profile *domain.Profile) error
}

// RoomRepository defines persistence operations for rooms.
type RoomRepository interface {
	Insert(room *domain.Room) error
	GetByID(id string) (*domain.Room, error)
	Update(room *domain.Room) error
	ListPublic(language string, paging domain.Paging) (*domain.PageOf[domain.Room], error)
}

// CacheRepository defines operations for non-relational key-value caching and coordination.
// Decoupling caching logic from specific technologies like Redis allows seamless replacement
// with in-memory stores, Memcached, or alternative key-value engines.
type CacheRepository interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (bool, error)
	Close() error
	HealthReporter
}
