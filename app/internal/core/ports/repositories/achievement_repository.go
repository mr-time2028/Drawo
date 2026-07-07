package repositories

import (
	"context"
	"drawo/internal/core/domain"
	"gorm.io/gorm"
)

type AchievementRepository interface {
	UnlockAchievement(ctx context.Context, achievement *domain.Achievement) error
	ListUserAchievements(ctx context.Context, userID string) ([]domain.Achievement, error)
}

type achievementRepo struct {
	db *gorm.DB
}

func NewAchievementRepo(db *gorm.DB) AchievementRepository {
	return &achievementRepo{db: db}
}

func (r *achievementRepo) UnlockAchievement(ctx context.Context, achievement *domain.Achievement) error {
	return r.db.WithContext(ctx).Create(achievement).Error
}

func (r *achievementRepo) ListUserAchievements(ctx context.Context, userID string) ([]domain.Achievement, error) {
	var list []domain.Achievement
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
