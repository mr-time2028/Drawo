package services

import (
	"context"
	"drawo/internal/core/domain"
	"drawo/pkg/errors"
)

type UserService interface {
	GetProfile(ctx context.Context, userID string) (*domain.UserWithProfile, error)
	UpdateProfile(ctx context.Context, userID string, updates domain.Profile) (*domain.Profile, error)
}

type userService struct{}

func NewUserService() UserService {
	return &userService{}
}

func (s *userService) GetProfile(ctx context.Context, userID string) (*domain.UserWithProfile, error) {
	return nil, errors.New(errors.ErrInternalServer, "user service not implemented")
}

func (s *userService) UpdateProfile(ctx context.Context, userID string, updates domain.Profile) (*domain.Profile, error) {
	return nil, errors.New(errors.ErrInternalServer, "user service not implemented")
}
