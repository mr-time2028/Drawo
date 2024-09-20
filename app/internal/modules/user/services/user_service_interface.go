package services

import (
	tokenModel "drawo/internal/modules/token/models"
	userModel "drawo/internal/modules/user/models"
	"drawo/internal/modules/user/requests"
	"drawo/pkg/errors"
)

type UserServiceInterface interface {
	Register(registerRequest *requests.RegisterRequest) (*userModel.User, *errors.TypedError)
	Login(loginRequest *requests.LoginRequest) (*tokenModel.JWTTokenPairs, *errors.TypedError)
	GetUserFromAuthHeader(authHeader string) (*userModel.User, *errors.TypedError)
}
