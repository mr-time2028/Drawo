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

// HealthReporter is implemented by any infrastructure component that can report health.
type HealthReporter interface {
	Health(ctx context.Context) error
}
