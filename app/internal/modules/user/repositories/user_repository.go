package repositories

import (
	"drawo/internal/modules/user/models"
	"drawo/pkg/database"
	"errors"
	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func New() *UserRepository {
	return &UserRepository{
		DB: database.GetDB(),
	}
}

func (userRepository *UserRepository) InsertOneUser(user models.User) (*models.User, error) {
	result := userRepository.DB.Create(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (userRepository *UserRepository) CheckIfUserExists(username string) (bool, error) {
	var user models.User
	condition := models.User{Username: username}
	result := userRepository.DB.Where(condition).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

func (userRepository *UserRepository) GetUserByUsername(username string) (*models.User, error) {
	var user *models.User
	condition := models.User{Username: username}
	result := userRepository.DB.Where(condition).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}
