package repositories

import (
	"drawo/internal/core/domain"
	"drawo/internal/core/ports"
	"errors"

	"gorm.io/gorm"
)

// ProfileRepo implements ports.ProfileRepository using GORM.
type ProfileRepo struct {
	db *gorm.DB
}

// NewProfileRepo creates a profile repository.
func NewProfileRepo(db *gorm.DB) ports.ProfileRepository {
	return &ProfileRepo{db: db}
}

// Insert creates a new profile record.
func (r *ProfileRepo) Insert(profile *domain.Profile) error {
	return r.db.Create(profile).Error
}

// GetByUserID retrieves a profile by user ID.
func (r *ProfileRepo) GetByUserID(userID string) (*domain.Profile, error) {
	var profile domain.Profile
	if err := r.db.First(&profile, "user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// Update modifies an existing profile.
func (r *ProfileRepo) Update(profile *domain.Profile) error {
	return r.db.Save(profile).Error
}

// Compile-time check.
var _ ports.ProfileRepository = (*ProfileRepo)(nil)
