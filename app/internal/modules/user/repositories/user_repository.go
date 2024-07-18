package repositories

import (
	"drawo/pkg/database"
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
