package services

import (
	"context"
	"strings"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
)

// ContentService manages the synced multi-language dictionary.
type ContentService interface {
	// SuggestWords picks random concepts in the drawer's language.
	SuggestWords(ctx context.Context, categoryID, lang string, count int) ([]domain.Word, error)
	// GetWordForPlayer returns the specific translation for a concept based on player preference.
	GetWordForPlayer(ctx context.Context, wordGroupID, playerLang string) (string, error)
	// FilterMessage masks bad words.
	FilterMessage(ctx context.Context, text, lang string) (string, bool)
}

type contentService struct {
	repo         repositories.ContentRepository
	profileRepo  repositories.ProfileRepository
	penaltyScore int64
}

func NewContentService(r repositories.ContentRepository, p repositories.ProfileRepository, penalty int64) ContentService {
	return &contentService{
		repo:         r,
		profileRepo:  p,
		penaltyScore: penalty,
	}
}

func (s *contentService) SuggestWords(ctx context.Context, categoryID, lang string, count int) ([]domain.Word, error) {
	return s.repo.GetRandomWordGroups(ctx, categoryID, lang, count)
}

func (s *contentService) GetWordForPlayer(ctx context.Context, wordGroupID, playerLang string) (string, error) {
	word, err := s.repo.GetTranslation(ctx, wordGroupID, playerLang)
	if err != nil {
		// FALLBACK: If no translation exists for this language, return the master (original) word.
		// In production, we'd find the default language entry.
		return "", err
	}
	return word.Text, nil
}

func (s *contentService) FilterMessage(ctx context.Context, text, lang string) (string, bool) {
	badWords, err := s.repo.ListBadWords(ctx, lang)
	if err != nil || len(badWords) == 0 {
		return text, false
	}

	lowerText := strings.ToLower(text)
	isDirty := false
	cleaned := text

	for _, bw := range badWords {
		bad := strings.ToLower(bw.Text)
		if strings.Contains(lowerText, bad) {
			isDirty = true
			mask := strings.Repeat("*", len(bw.Text))
			cleaned = strings.ReplaceAll(cleaned, bw.Text, mask)
		}
	}

	return cleaned, isDirty
}
