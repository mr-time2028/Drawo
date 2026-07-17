package repositories

import (
	"drawo/internal/core/domain"
	"gorm.io/gorm"
)

type ProfileRepository interface {
	Insert(profile *domain.Profile) error
	GetByUserID(userID string) (*domain.Profile, error)
	Update(profile *domain.Profile) error
}

type profileRepo struct {
	db *gorm.DB
}

func NewProfileRepo(db *gorm.DB) ProfileRepository {
	return &profileRepo{db: db}
}

func (r *profileRepo) Insert(profile *domain.Profile) error {
	return r.db.Create(profile).Error
}

func (r *profileRepo) GetByUserID(userID string) (*domain.Profile, error) {
	var profile domain.Profile
	if err := r.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepo) Update(profile *domain.Profile) error {
	return r.db.Where("user_id = ?", profile.UserID).Save(profile).Error
}
