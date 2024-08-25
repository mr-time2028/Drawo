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

func (userService *UserService) Register(registerRequest *requests.RegisterRequest) (*models.User, *errors.ServiceError) {
	isMatchPasswords := helpers.CompareRequestPasswords(registerRequest.Password, registerRequest.ConfirmPassword)
	if !isMatchPasswords {
		return nil, &errors.ServiceError{
			Error:   errors.BadRequestErr,
			Field:   "password",
			Message: "password and confirm password are not match",
		}
	}

	isExistsUser, err := userService.userRepository.CheckIfUserExists(registerRequest.Username)
	if err != nil {
		return nil, &errors.ServiceError{
			Error:   errors.InternalServerErr,
			Field:   "username",
			Message: fmt.Sprintf("cannot check if user exists: %s", err.Error()),
		}
	}

	if isExistsUser {
		return nil, &errors.ServiceError{
			Error:   errors.BadRequestErr,
			Field:   "username",
			Message: "username already taken",
		}
	}

	hashedPassword, err := helpers.HashPassword(registerRequest.Password)
	if err != nil {
		return nil, &errors.ServiceError{
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
		return nil, &errors.ServiceError{
			Error:   errors.InternalServerErr,
			Message: fmt.Sprintf("cannot insert user: %s", err.Error()),
		}
	}

	return newUser, nil
}

func (userService *UserService) Login(loginRequest *requests.LoginRequest) (*auth.TokenPairs, *errors.ServiceError) {
	user, err := userService.userRepository.GetUserByUsername(loginRequest.Username)
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, &errors.ServiceError{
				Error:   errors.BadRequestErr,
				Field:   "",
				Message: "incorrect username or password",
			}
		default:
			return nil, &errors.ServiceError{
				Error:   errors.InternalServerErr,
				Field:   "username",
				Message: fmt.Sprintf("cannot get the user from the database: %s", err.Error()),
			}
		}
	}

	if err = helpers.CompareRequestAndHashPasswords(loginRequest.Password, user.Password); err != nil {
		switch err {
		case bcrypt.ErrMismatchedHashAndPassword:
			return nil, &errors.ServiceError{
				Error:   errors.BadRequestErr,
				Field:   "",
				Message: "incorrect username or password",
			}
		default:
			return nil, &errors.ServiceError{
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
		return nil, &errors.ServiceError{
			Error:   errors.InternalServerErr,
			Field:   "token",
			Message: fmt.Sprintf("cannot generate token pairs: %s", err.Error()),
		}
	}

	return tokens, nil
}
