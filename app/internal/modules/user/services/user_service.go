package services

import "drawo/internal/modules/user/repositories"

type UserService struct {
	userRepository repositories.UserRepositoryInterface
}

func New() *UserService {
	return &UserService{
		userRepository: repositories.New(),
	}
}
