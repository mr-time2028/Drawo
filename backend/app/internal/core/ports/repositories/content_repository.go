package repositories

import (
	"context"
	"drawo/internal/core/domain"
	"gorm.io/gorm"
)

type ContentRepository interface {
	InsertCategory(ctx context.Context, cat *domain.Category) error
	ListCategories(ctx context.Context, lang string) ([]domain.Category, error)
	DeleteCategory(ctx context.Context, id string) error
	InsertWord(ctx context.Context, word *domain.Word) error
	ListWords(ctx context.Context, categoryID string, lang string) ([]domain.Word, error)
	DeleteWord(ctx context.Context, id string) error
	GetRandomWordGroups(ctx context.Context, categoryID string, lang string, count int) ([]domain.Word, error)
	GetTranslation(ctx context.Context, wordGroupID string, lang string) (*domain.Word, error)
	InsertBadWord(ctx context.Context, bw *domain.BadWord) error
	ListBadWords(ctx context.Context, lang string) ([]domain.BadWord, error)
	DeleteBadWord(ctx context.Context, id string) error
}

type contentRepo struct {
	db *gorm.DB
}

func NewContentRepo(db *gorm.DB) ContentRepository {
	return &contentRepo{db: db}
}

func (r *contentRepo) InsertCategory(ctx context.Context, cat *domain.Category) error {
	return r.db.WithContext(ctx).Create(cat).Error
}

func (r *contentRepo) ListCategories(ctx context.Context, lang string) ([]domain.Category, error) {
	var list []domain.Category
	err := r.db.WithContext(ctx).Where("language = ?", lang).Find(&list).Error
	return list, err
}

func (r *contentRepo) InsertWord(ctx context.Context, word *domain.Word) error {
	return r.db.WithContext(ctx).Create(word).Error
}

func (r *contentRepo) GetRandomWordGroups(ctx context.Context, categoryID string, lang string, count int) ([]domain.Word, error) {
	var list []domain.Word
	query := r.db.WithContext(ctx).Where("language = ?", lang)
	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	err := query.Order("RANDOM()").Limit(count).Find(&list).Error
	return list, err
}

func (r *contentRepo) GetTranslation(ctx context.Context, wordGroupID string, lang string) (*domain.Word, error) {
	var word domain.Word
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND language = ?", wordGroupID, lang).
		First(&word).Error
	if err != nil {
		return nil, err
	}
	return &word, nil
}

func (r *contentRepo) InsertBadWord(ctx context.Context, bw *domain.BadWord) error {
	return r.db.WithContext(ctx).Create(bw).Error
}

func (r *contentRepo) ListBadWords(ctx context.Context, lang string) ([]domain.BadWord, error) {
	var list []domain.BadWord
	err := r.db.WithContext(ctx).Where("language = ?", lang).Find(&list).Error
	return list, err
}

func (r *contentRepo) DeleteBadWord(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.BadWord{}).Error
}

func (r *contentRepo) DeleteCategory(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Category{}).Error
}

func (r *contentRepo) ListWords(ctx context.Context, categoryID string, lang string) ([]domain.Word, error) {
	var list []domain.Word
	query := r.db.WithContext(ctx).Where("language = ?", lang)
	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	err := query.Order("created_at desc").Find(&list).Error
	return list, err
}

func (r *contentRepo) DeleteWord(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Word{}).Error
}
