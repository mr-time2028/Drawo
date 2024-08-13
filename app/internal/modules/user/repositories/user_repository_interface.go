package repositories

import "drawo/internal/modules/user/models"

type UserRepositoryInterface interface {
	InsertOneUser(user models.User) (*models.User, error)
	CheckIfUserExists(username string) (bool, error)
}
