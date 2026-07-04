package services

import (
	"context"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports"
	"drawo/pkg/errors"
)

// UserService is a placeholder implementation of ports.UserService.
type UserService struct{}

// NewUserService creates a new placeholder user service.
func NewUserService() ports.UserService {
	return &UserService{}
}

// GetProfile is not implemented yet.
func (s *UserService) GetProfile(ctx context.Context, userID string) (*domain.UserWithProfile, error) {
	return nil, errors.New(errors.ErrInternalServer, "user service not implemented in Phase 1")
}

// UpdateProfile is not implemented yet.
func (s *UserService) UpdateProfile(ctx context.Context, userID string, updates domain.Profile) (*domain.Profile, error) {
	return nil, errors.New(errors.ErrInternalServer, "user service not implemented in Phase 1")
}

// Compile-time check.
var _ ports.UserService = (*UserService)(nil)
