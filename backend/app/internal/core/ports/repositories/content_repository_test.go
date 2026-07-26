package repositories

import (
	"context"
	"drawo/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContentRepo(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContentRepo(db)
	ctx := context.Background()

	// Test Categories
	cat := &domain.Category{ID: "c1", GroupID: "g1", Name: "Cat1", Language: "fa"}
	assert.NoError(t, repo.InsertCategory(ctx, cat))
	list, _ := repo.ListCategories(ctx, "fa")
	assert.Len(t, list, 1)

	// Test Words
	w := &domain.Word{ID: "w1", GroupID: "wg1", CategoryID: "c1", Text: "Word1", Language: "fa"}
	assert.NoError(t, repo.InsertWord(ctx, w))

	// Test BadWords
	bw := &domain.BadWord{ID: "b1", Text: "Bad1", Language: "fa"}
	assert.NoError(t, repo.InsertBadWord(ctx, bw))
	bws, _ := repo.ListBadWords(ctx, "fa")
	assert.Len(t, bws, 1)
}
