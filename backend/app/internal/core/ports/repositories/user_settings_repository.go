package repositories

import (
	"context"
	"drawo/internal/core/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserSettingsRepository interface {
	GetSettings(ctx context.Context, userID string) (*domain.UserSettings, error)
	SaveSettings(ctx context.Context, settings *domain.UserSettings) error
}

type userSettingsRepo struct {
	db *gorm.DB
}

func NewUserSettingsRepo(db *gorm.DB) UserSettingsRepository {
	return &userSettingsRepo{db: db}
}

func (r *userSettingsRepo) GetSettings(ctx context.Context, userID string) (*domain.UserSettings, error) {
	var s domain.UserSettings
	if err := r.db.WithContext(ctx).First(&s, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *userSettingsRepo) SaveSettings(ctx context.Context, settings *domain.UserSettings) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(settings).Error
}
