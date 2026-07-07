package services

import (
	"context"
	"drawo/internal/core/domain"
	"drawo/pkg/errors"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type AuthService interface {
	Register(ctx context.Context, username, password string) (*domain.User, error)
	Login(ctx context.Context, username, password string) (*TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
}

type authService struct{}

func NewAuthService() AuthService {
	return &authService{}
}

func (s *authService) Register(ctx context.Context, username, password string) (*domain.User, error) {
	return nil, errors.New(errors.ErrInternalServer, "auth service not implemented")
}

func (s *authService) Login(ctx context.Context, username, password string) (*TokenPair, error) {
	return nil, errors.New(errors.ErrInternalServer, "auth service not implemented")
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	return nil, errors.New(errors.ErrInternalServer, "auth service not implemented")
}

func (s *authService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	return errors.New(errors.ErrInternalServer, "auth service not implemented")
}
