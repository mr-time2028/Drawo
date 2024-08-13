package services

import (
	"drawo/internal/modules/user/helpers"
	"drawo/internal/modules/user/models"
	"drawo/internal/modules/user/repositories"
	"drawo/internal/modules/user/requests"
	"drawo/pkg/errors"
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
			Error:   errors.InternalServerErr,
			Field:   "password",
			Message: "password and confirm password are not match",
		}
	}

	isExistsUser, err := userService.userRepository.CheckIfUserExists(registerRequest.Username)
	if err != nil {
		return nil, &errors.ServiceError{
			Error:   errors.InternalServerErr,
			Field:   "username",
			Message: "cannot check if user exists",
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
			Message: "cannot hash password",
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
			Message: "cannot insert user",
		}
	}

	return newUser, nil
}
