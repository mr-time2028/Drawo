package repositories

import (
	"drawo/internal/core/domain"
	"errors"
	"gorm.io/gorm"
)

type UserRepository interface {
	Insert(user *domain.User) error
	GetByID(id string) (*domain.User, error)
	GetByUsername(username string) (*domain.User, error)
	Exists(username string) (bool, error)
	Update(user *domain.User) error
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Insert(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *userRepo) GetByID(id string) (*domain.User, error) {
	var user domain.User
	if err := r.db.First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetByUsername(username string) (*domain.User, error) {
	var user domain.User
	if err := r.db.First(&user, "username = ?", username).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) Exists(username string) (bool, error) {
	var count int64
	if err := r.db.Model(&domain.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *userRepo) Update(user *domain.User) error {
	return r.db.Save(user).Error
}
