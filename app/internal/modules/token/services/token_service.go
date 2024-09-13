package services

import (
	tokenHelper "drawo/internal/modules/token/helpers"
	tokenModel "drawo/internal/modules/token/models"
	"drawo/internal/modules/user/repositories"
	"drawo/pkg/errors"
	"fmt"
	"gorm.io/gorm"
)

type TokenService struct {
	UserRepository repositories.UserRepositoryInterface
}

func New() *TokenService {
	return &TokenService{
		UserRepository: repositories.New(),
	}
}

func (tokenService *TokenService) GenerateAccessTokenByRefreshToken(refreshToken string) (string, *errors.TypedError) {
	claims, err := tokenHelper.ParseWithClaims(refreshToken)
	if err != nil || claims.TokenType != "refresh" {
		return "", &errors.TypedError{
			Error:   errors.UnauthorizedErr,
			Field:   "",
			Message: "invalid refresh token",
		}
	}

	userID := claims.Subject
	user, err := tokenService.UserRepository.GetUserByID(userID)
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return "", &errors.TypedError{
				Error:   errors.UnauthorizedErr,
				Field:   "",
				Message: "invalid refresh token",
			}
		default:
			return "", &errors.TypedError{
				Error:   errors.InternalServerErr,
				Field:   "userID",
				Message: fmt.Sprintf("cannot get the user from the database: %s", err.Error()),
			}
		}
	}

	tokens, err := tokenHelper.GenerateTokenPair(&tokenModel.JwtUser{
		ID: user.ID,
	})
	if err != nil {
		return "", &errors.TypedError{
			Error:   errors.InternalServerErr,
			Field:   "token",
			Message: fmt.Sprintf("cannot generate token pairs: %s", err.Error()),
		}
	}

	return tokens.AccessToken, nil
}

func (tokenService *TokenService) VerifyAccessToken(accessToken string) (*tokenModel.JWTClaims, *errors.TypedError) {
	claims, err := tokenHelper.VerifyAccessToken(accessToken)
	if err != nil {
		return nil, &errors.TypedError{
			Error:   errors.UnauthorizedErr,
			Field:   "",
			Message: "invalid access token",
		}
	}
	return claims, nil
}
