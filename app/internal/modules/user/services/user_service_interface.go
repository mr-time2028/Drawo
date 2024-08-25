package services

import (
	"drawo/internal/modules/user/models"
	"drawo/internal/modules/user/requests"
	"drawo/pkg/auth"
	"drawo/pkg/errors"
)

type UserServiceInterface interface {
	Register(registerRequest *requests.RegisterRequest) (*models.User, *errors.ServiceError)
	Login(loginRequest *requests.LoginRequest) (*auth.TokenPairs, *errors.ServiceError)
}
