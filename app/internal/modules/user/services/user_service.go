package services

import (
	"drawo/internal/modules/user/helpers"
	"drawo/internal/modules/user/models"
	"drawo/internal/modules/user/repositories"
	"drawo/internal/modules/user/requests"
	"drawo/pkg/auth"
	"drawo/pkg/errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	userRepository repositories.UserRepositoryInterface
}

func New() *UserService {
	return &UserService{
		userRepository: repositories.New(),
	}
}

func (userService *UserService) Register(registerRequest *requests.RegisterRequest) (*models.User, *errors.TypedError) {
	if registerRequest.Password != registerRequest.ConfirmPassword {
		return nil, &errors.TypedError{
			Error:   errors.BadRequestErr,
			Field:   "password",
			Message: "password and confirm password are not match",
		}
	}

	isExistsUser, err := userService.userRepository.CheckIfUserExists(registerRequest.Username)
	if err != nil {
		return nil, &errors.TypedError{
			Error:   errors.InternalServerErr,
			Field:   "username",
			Message: fmt.Sprintf("cannot check if user exists: %s", err.Error()),
		}
	}

	if isExistsUser {
		return nil, &errors.TypedError{
			Error:   errors.BadRequestErr,
			Field:   "username",
			Message: "username already taken",
		}
	}

	hashedPassword, err := helpers.HashPassword(registerRequest.Password)
	if err != nil {
		return nil, &errors.TypedError{
			Error:   errors.InternalServerErr,
			Field:   "password",
			Message: fmt.Sprintf("cannot hash password: %s", err.Error()),
		}
	}

	user := models.User{
		Username: registerRequest.Username,
		Password: string(hashedPassword),
	}

	newUser, err := userService.userRepository.InsertOneUser(user)
	if err != nil {
		return nil, &errors.TypedError{
			Error:   errors.InternalServerErr,
			Message: fmt.Sprintf("cannot insert user: %s", err.Error()),
		}
	}

	return newUser, nil
}

func (userService *UserService) Login(loginRequest *requests.LoginRequest) (*auth.TokenPairs, *errors.TypedError) {
	user, err := userService.userRepository.GetUserByUsername(loginRequest.Username)
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, &errors.TypedError{
				Error:   errors.UnauthorizedErr,
				Field:   "",
				Message: "incorrect username or password",
			}
		default:
			return nil, &errors.TypedError{
				Error:   errors.InternalServerErr,
				Field:   "username",
				Message: fmt.Sprintf("cannot get the user from the database: %s", err.Error()),
			}
		}
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginRequest.Password)); err != nil {
		switch err {
		case bcrypt.ErrMismatchedHashAndPassword:
			return nil, &errors.TypedError{
				Error:   errors.UnauthorizedErr,
				Field:   "",
				Message: "incorrect username or password",
			}
		default:
			return nil, &errors.TypedError{
				Error:   errors.InternalServerErr,
				Field:   "password",
				Message: fmt.Sprintf("cannot compare user password with request password: %s", err.Error()),
			}
		}
	}

	tokens, err := auth.GenerateTokenPair(&auth.JwtUser{
		ID: user.ID,
	})
	if err != nil {
		return nil, &errors.TypedError{
			Error:   errors.InternalServerErr,
			Field:   "token",
			Message: fmt.Sprintf("cannot generate token pairs: %s", err.Error()),
		}
	}

	return tokens, nil
}

func (userService *UserService) GenerateAccessTokenByRefreshToken(refreshToken string) (string, *errors.TypedError) {
	claims, err := auth.ParseWithClaims(refreshToken)
	if err != nil || claims.TokenType != "refresh" {
		return "", &errors.TypedError{
			Error:   errors.UnauthorizedErr,
			Field:   "",
			Message: "invalid refresh token",
		}
	}

	userID := claims.Subject
	user, err := userService.userRepository.GetUserByID(userID)
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

	tokens, err := auth.GenerateTokenPair(&auth.JwtUser{
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

func (userService *UserService) VerifyAccessToken(accessToken string) (*auth.Claims, *errors.TypedError) {
	claims, err := auth.VerifyAccessToken(accessToken)
	if err != nil {
		return nil, &errors.TypedError{
			Error:   errors.UnauthorizedErr,
			Field:   "",
			Message: "invalid access token",
		}
	}
	return claims, nil
}
