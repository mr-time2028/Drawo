package ports

import (
	"context"

	"drawo/internal/core/domain"
)

// TokenPair is the result of a successful login or refresh.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds
}

// AuthService defines the authentication use cases.
type AuthService interface {
	Register(ctx context.Context, username, password string) (*domain.User, error)
	Login(ctx context.Context, username, password string) (*TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
}

// UserService defines user management use cases.
type UserService interface {
	GetProfile(ctx context.Context, userID string) (*domain.UserWithProfile, error)
	UpdateProfile(ctx context.Context, userID string, updates domain.Profile) (*domain.Profile, error)
}

// RoomService defines application use cases for ephemeral room creation and discovery.
type RoomService interface {
	CreateRoom(ctx context.Context, name, ownerID string, roomType domain.RoomType, language string, maxPlayers, roundTime, maxRounds int) (*domain.Room, error)
	GetRoom(ctx context.Context, roomID string) (*domain.Room, error)
	JoinByInviteCode(ctx context.Context, inviteCode string) (*domain.Room, error)
}

// HealthReporter is implemented by any infrastructure component that can report health.
type HealthReporter interface {
	Health(ctx context.Context) error
}
