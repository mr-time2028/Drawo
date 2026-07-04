// Package repositories contains concrete persistence implementations.
//
// Each repository implements one port from internal/core/ports using a specific
// technology (GORM for relational databases, Redis/memory adapters for cache, etc.).
package repositories

import (
	"drawo/internal/core/domain"
	"drawo/internal/core/ports"
	"errors"

	"gorm.io/gorm"
)

// UserRepo implements ports.UserRepository using GORM.
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo creates a user repository.
func NewUserRepo(db *gorm.DB) ports.UserRepository {
	return &UserRepo{db: db}
}

// Insert creates a new user record.
func (r *UserRepo) Insert(user *domain.User) error {
	return r.db.Create(user).Error
}

// GetByID retrieves a user by primary key.
func (r *UserRepo) GetByID(id string) (*domain.User, error) {
	var user domain.User
	if err := r.db.First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername retrieves a user by username.
func (r *UserRepo) GetByUsername(username string) (*domain.User, error) {
	var user domain.User
	if err := r.db.First(&user, "username = ?", username).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Exists returns true if a username is already taken.
func (r *UserRepo) Exists(username string) (bool, error) {
	var count int64
	if err := r.db.Model(&domain.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Update modifies an existing user.
func (r *UserRepo) Update(user *domain.User) error {
	return r.db.Save(user).Error
}

// Compile-time check.
var _ ports.UserRepository = (*UserRepo)(nil)
