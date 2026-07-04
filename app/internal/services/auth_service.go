// Package services implements the application use cases defined in internal/core/ports.
//
// Services contain business logic. They orchestrate repositories and infrastructure
// but never import HTTP frameworks. Each service implements one port interface.
package services

import (
	"context"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports"
	"drawo/pkg/errors"
)

// AuthService is a placeholder implementation of ports.AuthService.
//
// It will be replaced by real JWT-based authentication in Phase 5.
// Returning an AppError lets controllers return a proper HTTP response even now.
type AuthService struct{}

// NewAuthService creates a new placeholder auth service.
func NewAuthService() ports.AuthService {
	return &AuthService{}
}

// Register is not implemented yet.
func (s *AuthService) Register(ctx context.Context, username, password string) (*domain.User, error) {
	return nil, errors.New(errors.ErrInternalServer, "auth service not implemented in Phase 1")
}

// Login is not implemented yet.
func (s *AuthService) Login(ctx context.Context, username, password string) (*ports.TokenPair, error) {
	return nil, errors.New(errors.ErrInternalServer, "auth service not implemented in Phase 1")
}

// Refresh is not implemented yet.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*ports.TokenPair, error) {
	return nil, errors.New(errors.ErrInternalServer, "auth service not implemented in Phase 1")
}

// Logout is not implemented yet.
func (s *AuthService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	return errors.New(errors.ErrInternalServer, "auth service not implemented in Phase 1")
}

// Compile-time check.
var _ ports.AuthService = (*AuthService)(nil)
