package repositories

import (
	"context"

	"drawo/internal/core/domain"
	"gorm.io/gorm"
)

// ReputationRepository persists reputation audit events. The profile table stores
// the current reputation score; this repository stores the reason trail.
type ReputationRepository interface {
	InsertEvent(ctx context.Context, event *domain.ReputationEvent) error
	ListUserEvents(ctx context.Context, userID string, limit int) ([]domain.ReputationEvent, error)
}

type reputationRepo struct {
	db *gorm.DB
}

func NewReputationRepo(db *gorm.DB) ReputationRepository {
	return &reputationRepo{db: db}
}

func (r *reputationRepo) InsertEvent(ctx context.Context, event *domain.ReputationEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *reputationRepo) ListUserEvents(ctx context.Context, userID string, limit int) ([]domain.ReputationEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	var events []domain.ReputationEvent
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}
