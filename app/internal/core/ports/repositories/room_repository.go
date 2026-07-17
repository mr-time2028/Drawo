package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"drawo/internal/core/domain"
)

const (
	roomKeyPrefix   = "room:"
	inviteKeyPrefix = "invite:"
	publicRoomsKey  = "rooms:public:"
	roomDefaultTTL  = 24 * time.Hour
)

type RoomRepository interface {
	Save(ctx context.Context, room *domain.Room) error
	GetByID(ctx context.Context, id string) (*domain.Room, error)
	GetByInviteCode(ctx context.Context, inviteCode string) (*domain.Room, error)
	Delete(ctx context.Context, id string, inviteCode string) error
	ListPublic(ctx context.Context, language string, paging domain.Paging) (*domain.PageOf[domain.Room], error)
}

type ephemeralRoomRepo struct {
	cache CacheRepository
}

func NewRoomRepo(cache CacheRepository) RoomRepository {
	return &ephemeralRoomRepo{cache: cache}
}

func (r *ephemeralRoomRepo) Save(ctx context.Context, room *domain.Room) error {
	if room == nil || room.ID == "" {
		return errors.New("invalid room: ID required")
	}
	data, _ := json.Marshal(room)
	roomKey := roomKeyPrefix + room.ID
	if err := r.cache.Set(ctx, roomKey, string(data), roomDefaultTTL); err != nil {
		return fmt.Errorf("save room metadata: %w", err)
	}
	if room.InviteCode != "" {
		inviteKey := inviteKeyPrefix + strings.ToUpper(room.InviteCode)
		_ = r.cache.Set(ctx, inviteKey, room.ID, roomDefaultTTL)
	}
	return nil
}

func (r *ephemeralRoomRepo) GetByID(ctx context.Context, id string) (*domain.Room, error) {
	data, err := r.cache.Get(ctx, roomKeyPrefix+id)
	if err != nil {
		return nil, nil
	}
	var room domain.Room
	_ = json.Unmarshal([]byte(data), &room)
	return &room, nil
}

func (r *ephemeralRoomRepo) GetByInviteCode(ctx context.Context, inviteCode string) (*domain.Room, error) {
	if inviteCode == "" {
		return nil, nil
	}
	roomID, err := r.cache.Get(ctx, inviteKeyPrefix+strings.ToUpper(inviteCode))
	if err != nil || roomID == "" {
		return nil, nil
	}
	return r.GetByID(ctx, roomID)
}

func (r *ephemeralRoomRepo) Delete(ctx context.Context, id string, inviteCode string) error {
	keysToDelete := []string{roomKeyPrefix + id}
	if inviteCode != "" {
		keysToDelete = append(keysToDelete, inviteKeyPrefix+strings.ToUpper(inviteCode))
	}
	return r.cache.Delete(ctx, keysToDelete...)
}

func (r *ephemeralRoomRepo) ListPublic(ctx context.Context, language string, paging domain.Paging) (*domain.PageOf[domain.Room], error) {
	return &domain.PageOf[domain.Room]{Items: []domain.Room{}, Total: 0, Limit: paging.Limit, Offset: paging.Offset}, nil
}
