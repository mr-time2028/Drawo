package repositories

import (
	"context"
	"drawo/internal/core/domain"
	"gorm.io/gorm"
)

type GameHistoryRepository interface {
	SaveGameSummary(ctx context.Context, summary *domain.GameHistory, rounds []domain.Round, scores []domain.Score) error
	GetGameSummary(ctx context.Context, gameID string) (*domain.GameHistory, []domain.Round, []domain.Score, error)
	ListUserGames(ctx context.Context, userID string, paging domain.Paging) (*domain.PageOf[domain.GameHistory], error)
}

type gameHistoryRepo struct {
	db *gorm.DB
}

func NewGameHistoryRepo(db *gorm.DB) GameHistoryRepository {
	return &gameHistoryRepo{db: db}
}

func (r *gameHistoryRepo) SaveGameSummary(ctx context.Context, summary *domain.GameHistory, rounds []domain.Round, scores []domain.Score) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(summary).Error; err != nil { return err }
		if len(rounds) > 0 { if err := tx.Create(&rounds).Error; err != nil { return err } }
		if len(scores) > 0 { if err := tx.Create(&scores).Error; err != nil { return err } }
		return nil
	})
}

func (r *gameHistoryRepo) GetGameSummary(ctx context.Context, gameID string) (*domain.GameHistory, []domain.Round, []domain.Score, error) {
	var summary domain.GameHistory
	var rounds []domain.Round
	var scores []domain.Score
	if err := r.db.WithContext(ctx).First(&summary, "id = ?", gameID).Error; err != nil { return nil, nil, nil, err }
	r.db.WithContext(ctx).Where("game_history_id = ?", gameID).Find(&rounds)
	r.db.WithContext(ctx).Where("game_history_id = ?", gameID).Find(&scores)
	return &summary, rounds, scores, nil
}

func (r *gameHistoryRepo) ListUserGames(ctx context.Context, userID string, paging domain.Paging) (*domain.PageOf[domain.GameHistory], error) {
	var list []domain.GameHistory
	var total int64
	r.db.WithContext(ctx).Model(&domain.GameHistory{}).Count(&total)
	r.db.WithContext(ctx).Limit(paging.Limit).Offset(paging.Offset).Find(&list)
	return &domain.PageOf[domain.GameHistory]{Items: list, Total: total, Limit: paging.Limit, Offset: paging.Offset}, nil
}
