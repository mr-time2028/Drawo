package services

import (
	"drawo/internal/modules/user/models"
	"drawo/internal/modules/user/requests"
	"drawo/pkg/auth"
	"drawo/pkg/errors"
)

type UserServiceInterface interface {
	Register(registerRequest *requests.RegisterRequest) (*models.User, *errors.TypedError)
	Login(loginRequest *requests.LoginRequest) (*auth.TokenPairs, *errors.TypedError)
	GenerateAccessTokenByRefreshToken(refreshToken string) (string, *errors.TypedError)
	VerifyAccessToken(accessToken string) (*auth.Claims, *errors.TypedError)
}
