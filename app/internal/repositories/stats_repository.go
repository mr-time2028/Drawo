package repositories

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports"
)

// StatsRepo implements persistent stats, reports, achievements, and user settings using GORM.
type StatsRepo struct {
	db *gorm.DB
}

// NewStatsRepo creates a persistent statistics repository.
func NewStatsRepo(db *gorm.DB) *StatsRepo {
	return &StatsRepo{db: db}
}

// InsertReport records a user moderation report.
func (r *StatsRepo) InsertReport(ctx context.Context, report *domain.Report) error {
	return r.db.WithContext(ctx).Create(report).Error
}

// ListReports returns paginated moderation reports.
func (r *StatsRepo) ListReports(ctx context.Context, paging domain.Paging) (*domain.PageOf[domain.Report], error) {
	var list []domain.Report
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Report{})
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := query.Limit(paging.Limit).Offset(paging.Offset).Order("created_at desc").Find(&list).Error; err != nil {
		return nil, err
	}

	return &domain.PageOf[domain.Report]{
		Items:  list,
		Total:  total,
		Limit:  paging.Limit,
		Offset: paging.Offset,
	}, nil
}

// UnlockAchievement saves an achievement unlocked by a player.
func (r *StatsRepo) UnlockAchievement(ctx context.Context, achievement *domain.Achievement) error {
	return r.db.WithContext(ctx).Create(achievement).Error
}

// ListUserAchievements retrieves all achievements unlocked by a user.
func (r *StatsRepo) ListUserAchievements(ctx context.Context, userID string) ([]domain.Achievement, error) {
	var list []domain.Achievement
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetStats returns lifetime statistics for a player.
func (r *StatsRepo) GetStats(ctx context.Context, userID string) (*domain.PlayerStatistic, error) {
	var stats domain.PlayerStatistic
	if err := r.db.WithContext(ctx).First(&stats, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &stats, nil
}

// UpsertStats inserts or updates lifetime statistics for a player.
func (r *StatsRepo) UpsertStats(ctx context.Context, stats *domain.PlayerStatistic) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		UpdateAll: true,
	}).Create(stats).Error
}

// GetSettings returns persistent UI preferences for a user.
func (r *StatsRepo) GetSettings(ctx context.Context, userID string) (*domain.UserSettings, error) {
	var settings domain.UserSettings
	if err := r.db.WithContext(ctx).First(&settings, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

// SaveSettings updates or creates UI preferences for a user.
func (r *StatsRepo) SaveSettings(ctx context.Context, settings *domain.UserSettings) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		UpdateAll: true,
	}).Create(settings).Error
}

var _ ports.ReportRepository = (*StatsRepo)(nil)
var _ ports.AchievementRepository = (*StatsRepo)(nil)
var _ ports.PlayerStatsRepository = (*StatsRepo)(nil)
var _ ports.UserSettingsRepository = (*StatsRepo)(nil)
