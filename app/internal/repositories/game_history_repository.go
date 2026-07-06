package repositories

import (
	"context"

	"gorm.io/gorm"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports"
)

// GameHistoryRepo implements ports.GameHistoryRepository using GORM.
type GameHistoryRepo struct {
	db *gorm.DB
}

// NewGameHistoryRepo creates a persistent game history repository.
func NewGameHistoryRepo(db *gorm.DB) ports.GameHistoryRepository {
	return &GameHistoryRepo{db: db}
}

// SaveGameSummary atomically saves a completed game summary along with its rounds and scores.
func (r *GameHistoryRepo) SaveGameSummary(ctx context.Context, summary *domain.GameHistory, rounds []domain.Round, scores []domain.Score) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(summary).Error; err != nil {
			return err
		}
		if len(rounds) > 0 {
			if err := tx.Create(&rounds).Error; err != nil {
				return err
			}
		}
		if len(scores) > 0 {
			if err := tx.Create(&scores).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetGameSummary retrieves a completed game summary along with its rounds and scores.
func (r *GameHistoryRepo) GetGameSummary(ctx context.Context, gameID string) (*domain.GameHistory, []domain.Round, []domain.Score, error) {
	var summary domain.GameHistory
	if err := r.db.WithContext(ctx).First(&summary, "id = ?", gameID).Error; err != nil {
		return nil, nil, nil, err
	}

	var rounds []domain.Round
	_ = r.db.WithContext(ctx).Where("game_history_id = ?", gameID).Order("round_number asc").Find(&rounds).Error

	var scores []domain.Score
	_ = r.db.WithContext(ctx).Where("game_history_id = ?", gameID).Order("rank asc").Find(&scores).Error

	return &summary, rounds, scores, nil
}

// ListUserGames lists paginated game histories for a specific player.
func (r *GameHistoryRepo) ListUserGames(ctx context.Context, userID string, paging domain.Paging) (*domain.PageOf[domain.GameHistory], error) {
	var list []domain.GameHistory
	var total int64

	subQuery := r.db.Model(&domain.Score{}).Select("game_history_id").Where("user_id = ?", userID)
	query := r.db.WithContext(ctx).Model(&domain.GameHistory{}).Where("id IN (?)", subQuery)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.Limit(paging.Limit).Offset(paging.Offset).Order("ended_at desc").Find(&list).Error; err != nil {
		return nil, err
	}

	return &domain.PageOf[domain.GameHistory]{
		Items:  list,
		Total:  total,
		Limit:  paging.Limit,
		Offset: paging.Offset,
	}, nil
}

var _ ports.GameHistoryRepository = (*GameHistoryRepo)(nil)
