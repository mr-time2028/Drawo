package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
)

func TestContentService(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&domain.Category{}, &domain.Word{}, &domain.BadWord{}, &domain.Profile{})

	repo := repositories.NewContentRepo(db)
	pRepo := repositories.NewProfileRepo(db)
	svc := NewContentService(repo, pRepo, 100)
	ctx := context.Background()

	// 1. SuggestWords
	repo.InsertCategory(ctx, &domain.Category{ID: "c1", Name: "Animals", Language: "en"})
	repo.InsertWord(ctx, &domain.Word{CategoryID: "c1", Text: "Cat", Language: "en"})
	
	words, err := svc.SuggestWords(ctx, "c1", "en", 1)
	require.NoError(t, err)
	assert.Len(t, words, 1)

	// 2. FilterMessage (Clean)
	cleaned, dirty := svc.FilterMessage(ctx, "Hello world", "en")
	assert.False(t, dirty)
	assert.Equal(t, "Hello world", cleaned)

	// 3. FilterMessage (Dirty)
	repo.InsertBadWord(ctx, &domain.BadWord{Text: "badword", Language: "en"})
	cleaned, dirty = svc.FilterMessage(ctx, "You are a badword", "en")
	assert.True(t, dirty)
	assert.Equal(t, "You are a *******", cleaned)
}
