package repositories

import (
	"drawo/internal/core/domain"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

// UserRepository defines persistence operations for accounts stored in the relational database.
type UserRepository interface {
	Insert(user *domain.User) error
	GetByID(id string) (*domain.User, error)
	GetByUsername(username string) (*domain.User, error)
	Exists(username string) (bool, error)
	Update(user *domain.User) error

	// SearchUsers allows admin to find users by partial username, email, or phone.
	// This joins the users and profiles tables.
	SearchUsers(query string) ([]domain.UserWithProfile, error)
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

func (r *userRepo) SearchUsers(query string) ([]domain.UserWithProfile, error) {
	var results []domain.UserWithProfile

	// Build a join query to search across users (username) and profiles (email, phone)
	likeQuery := fmt.Sprintf("%%%s%%", query)

	// In GORM, we can scan into a slice of a helper struct or a custom type.
	// We'll perform a Join between users and profiles.
	rows, err := r.db.Table("users").
		Select("users.*, profiles.*").
		Joins("left join profiles on profiles.user_id = users.id").
		Where("users.username LIKE ? OR profiles.email LIKE ? OR profiles.phone LIKE ?", likeQuery, likeQuery, likeQuery).
		Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var u domain.User
		var p domain.Profile
		// Scan both structs from the row
		if err := r.db.ScanRows(rows, &u); err != nil {
			return nil, err
		}
		if err := r.db.ScanRows(rows, &p); err != nil {
			return nil, err
		}

		results = append(results, domain.UserWithProfile{
			User:    u,
			Profile: p,
		})
	}

	return results, nil
}
