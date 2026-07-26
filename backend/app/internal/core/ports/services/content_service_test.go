package services

import (
	"context"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestContentService(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&domain.Category{}, &domain.Word{}, &domain.BadWord{}, &domain.Profile{})
	repo := repositories.NewContentRepo(db)
	pRepo := repositories.NewProfileRepo(db)
	svc := NewContentService(repo, pRepo, 100)
	ctx := context.Background()

	repo.InsertCategory(ctx, &domain.Category{ID: "c1", Name: "A", Language: "en"})
	repo.InsertWord(ctx, &domain.Word{ID: "w1", GroupID: "g1", CategoryID: "c1", Text: "T", Language: "en"})
	svc.SuggestWords(ctx, "c1", "en", 1)

	repo.InsertWord(ctx, &domain.Word{ID: "w2", GroupID: "g1", Text: "S", Language: "fa"})
	text, _ := svc.GetWordForPlayer(ctx, "g1", "fa")
	assert.Equal(t, "S", text)

	repo.InsertBadWord(ctx, &domain.BadWord{ID: "b1", Text: "bad", Language: "en"})
	cleaned, dirty := svc.FilterMessage(ctx, "hello bad world", "en")
	assert.True(t, dirty)
	assert.Equal(t, "hello *** world", cleaned)
}
