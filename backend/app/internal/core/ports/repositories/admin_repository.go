package repositories

import (
	"context"
	"errors"

	"drawo/internal/core/domain"

	"gorm.io/gorm"
)

// AdminRepository handles global settings and song metadata in the database.
type AdminRepository interface {
	// Song Metadata
	SaveSong(ctx context.Context, song *domain.Song) error
	ListSongs(ctx context.Context, songType domain.SongType) ([]domain.Song, error)
	GetSongByID(ctx context.Context, id string) (*domain.Song, error)
	DeleteSong(ctx context.Context, id string) error

	// Global Settings
	GetSetting(ctx context.Context, key string) (string, error)
	UpdateSetting(ctx context.Context, key, value string) error
}

type adminRepo struct {
	db *gorm.DB
}

// NewAdminRepo creates a repository for admin-level configurations.
func NewAdminRepo(db *gorm.DB) AdminRepository {
	return &adminRepo{db: db}
}

func (r *adminRepo) SaveSong(ctx context.Context, song *domain.Song) error {
	return r.db.WithContext(ctx).Save(song).Error
}

func (r *adminRepo) ListSongs(ctx context.Context, songType domain.SongType) ([]domain.Song, error) {
	var list []domain.Song
	err := r.db.WithContext(ctx).Where("type = ?", songType).Find(&list).Error
	return list, err
}

func (r *adminRepo) GetSongByID(ctx context.Context, id string) (*domain.Song, error) {
	var song domain.Song
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&song).Error
	return &song, err
}

func (r *adminRepo) DeleteSong(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Song{}).Error
}

func (r *adminRepo) GetSetting(ctx context.Context, key string) (string, error) {
	var setting domain.GlobalSetting
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return setting.Value, err
}

func (r *adminRepo) UpdateSetting(ctx context.Context, key, value string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&domain.GlobalSetting{}).
			Where("key = ?", key).
			Update("value", value)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return nil
		}
		return tx.Create(&domain.GlobalSetting{Key: key, Value: value}).Error
	})
}
