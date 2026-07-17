package repositories

import (
	"context"
	"drawo/internal/core/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PlayerStatisticRepository interface {
	GetStats(ctx context.Context, userID string) (*domain.PlayerStatistic, error)
	UpsertStats(ctx context.Context, stats *domain.PlayerStatistic) error
}

type playerStatisticRepo struct {
	db *gorm.DB
}

func NewPlayerStatisticRepo(db *gorm.DB) PlayerStatisticRepository {
	return &playerStatisticRepo{db: db}
}

func (r *playerStatisticRepo) GetStats(ctx context.Context, userID string) (*domain.PlayerStatistic, error) {
	var s domain.PlayerStatistic
	if err := r.db.WithContext(ctx).First(&s, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *playerStatisticRepo) UpsertStats(ctx context.Context, stats *domain.PlayerStatistic) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(stats).Error
}
