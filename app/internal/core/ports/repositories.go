// Package ports defines interfaces (ports) that the application layer depends on.
//
// In Clean Architecture / Hexagonal terms, these are "driven ports".
// They are implemented by adapters in internal/repositories and internal/infrastructure.
package ports

import (
	"context"
	"time"

	"drawo/internal/core/domain"
)

// UserRepository defines persistence operations for accounts stored in the relational database.
type UserRepository interface {
	Insert(user *domain.User) error
	GetByID(id string) (*domain.User, error)
	GetByUsername(username string) (*domain.User, error)
	Exists(username string) (bool, error)
	Update(user *domain.User) error
}

// ProfileRepository defines persistence operations for user profiles stored in the relational database.
type ProfileRepository interface {
	Insert(profile *domain.Profile) error
	GetByUserID(userID string) (*domain.Profile, error)
	Update(profile *domain.Profile) error
}

// RoomRepository defines operations for ephemeral room discovery and invite code lookup.
// As per architectural rules, rooms are NOT stored in relational databases. They are ephemeral
// runtime objects coordinated across server instances via non-relational distributed cache (Redis/memory).
type RoomRepository interface {
	Save(ctx context.Context, room *domain.Room) error
	GetByID(ctx context.Context, id string) (*domain.Room, error)
	GetByInviteCode(ctx context.Context, inviteCode string) (*domain.Room, error)
	Delete(ctx context.Context, id string, inviteCode string) error
	ListPublic(ctx context.Context, language string, paging domain.Paging) (*domain.PageOf[domain.Room], error)
}

// FriendshipRepository defines persistence operations for friendships in the relational database.
type FriendshipRepository interface {
	AddFriend(ctx context.Context, friendship *domain.Friendship) error
	ListFriends(ctx context.Context, userID string) ([]domain.Friendship, error)
	RemoveFriend(ctx context.Context, userID, friendID string) error
}

// FriendRequestRepository defines persistence operations for friend requests in the relational database.
type FriendRequestRepository interface {
	CreateRequest(ctx context.Context, req *domain.FriendRequest) error
	GetByID(ctx context.Context, id string) (*domain.FriendRequest, error)
	ListPending(ctx context.Context, userID string) ([]domain.FriendRequest, error)
	UpdateRequest(ctx context.Context, req *domain.FriendRequest) error
}

// GameHistoryRepository defines persistence operations for historical game summaries, rounds, and scores.
type GameHistoryRepository interface {
	SaveGameSummary(ctx context.Context, summary *domain.GameHistory, rounds []domain.Round, scores []domain.Score) error
	GetGameSummary(ctx context.Context, gameID string) (*domain.GameHistory, []domain.Round, []domain.Score, error)
	ListUserGames(ctx context.Context, userID string, paging domain.Paging) (*domain.PageOf[domain.GameHistory], error)
}

// ReportRepository defines persistence operations for moderation reports in the relational database.
type ReportRepository interface {
	InsertReport(ctx context.Context, report *domain.Report) error
	ListReports(ctx context.Context, paging domain.Paging) (*domain.PageOf[domain.Report], error)
}

// AchievementRepository defines persistence operations for user achievements in the relational database.
type AchievementRepository interface {
	UnlockAchievement(ctx context.Context, achievement *domain.Achievement) error
	ListUserAchievements(ctx context.Context, userID string) ([]domain.Achievement, error)
}

// PlayerStatsRepository defines persistence operations for lifetime player statistics in the relational database.
type PlayerStatsRepository interface {
	GetStats(ctx context.Context, userID string) (*domain.PlayerStatistic, error)
	UpsertStats(ctx context.Context, stats *domain.PlayerStatistic) error
}

// UserSettingsRepository defines persistence operations for UI/UX preferences in the relational database.
type UserSettingsRepository interface {
	GetSettings(ctx context.Context, userID string) (*domain.UserSettings, error)
	SaveSettings(ctx context.Context, settings *domain.UserSettings) error
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
