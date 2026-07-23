package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"drawo/internal/core/domain"
)

type fakeContentRepo struct {
	badWords []domain.BadWord
}

func (f *fakeContentRepo) InsertCategory(ctx context.Context, cat *domain.Category) error { return nil }
func (f *fakeContentRepo) ListCategories(ctx context.Context, lang string) ([]domain.Category, error) {
	return nil, nil
}
func (f *fakeContentRepo) InsertWord(ctx context.Context, word *domain.Word) error { return nil }
func (f *fakeContentRepo) GetRandomWordGroups(ctx context.Context, categoryID string, lang string, count int) ([]domain.Word, error) {
	return nil, nil
}
func (f *fakeContentRepo) GetTranslation(ctx context.Context, wordGroupID string, lang string) (*domain.Word, error) {
	return nil, nil
}
func (f *fakeContentRepo) InsertBadWord(ctx context.Context, bw *domain.BadWord) error {
	f.badWords = append(f.badWords, *bw)
	return nil
}
func (f *fakeContentRepo) ListBadWords(ctx context.Context, lang string) ([]domain.BadWord, error) {
	out := []domain.BadWord{}
	for _, bw := range f.badWords {
		if bw.Language == lang {
			out = append(out, bw)
		}
	}
	return out, nil
}
func (f *fakeContentRepo) DeleteBadWord(ctx context.Context, id string) error { return nil }

func TestNormalizeModerationText(t *testing.T) {
	assert.Equal(t, "badword", NormalizeModerationText("B A.D-word!", "en"))
	assert.Equal(t, "کلمه", NormalizeModerationText("ك‌لِ م ه", "fa"))
}

func TestRoomRejectsBadWordsAndChatSpam(t *testing.T) {
	repo := &fakeContentRepo{badWords: []domain.BadWord{{Text: "badword", Language: "en"}}}
	state := &domain.Room{ID: "moderation", State: domain.RoomStatePlaying, Language: "en", CurrentDrawerID: "drawer"}
	room := NewRoom(state, func(string, string) {}, repo, nil, nil)
	client := &Client{ID: "c1", UserID: "guesser", Send: make(chan []byte, 20), Done: make(chan struct{})}
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: client, Timestamp: time.Now()})
	drainClient(client)
	room.gameState = GameStateDrawing

	room.handleEvent(&RoomEvent{Type: EventChat, Client: client, Payload: mustMarshalRaw(ChatPayload{Text: "b a d w o r d"}), Timestamp: time.Now()})
	msg := nextEnvelope(t, client)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "bad_word")

	for i := 0; i < maxChatMessagesPerWind; i++ {
		room.handleEvent(&RoomEvent{Type: EventChat, Client: client, Payload: mustMarshalRaw(ChatPayload{Text: "hello"}), Timestamp: time.Now()})
	}
	drainClient(client)
	room.handleEvent(&RoomEvent{Type: EventChat, Client: client, Payload: mustMarshalRaw(ChatPayload{Text: "hello again"}), Timestamp: time.Now()})
	msg = nextEnvelope(t, client)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "chat_rate_limited")
}
