package services

import (
	tokenModel "drawo/internal/modules/auth/models"
	"drawo/pkg/errors"
)

type TokenServiceInterface interface {
	GenerateAccessTokenByRefreshToken(refreshToken string) (string, *errors.TypedError)
	VerifyAccessToken(accessToken string) (*tokenModel.JWTClaims, *errors.TypedError)
}
