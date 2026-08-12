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

// Guest-token TTL is mirrored from the service layer (services.guestTokenTTL)
// so the ephemeral repo can set Redis expiry without a circular import. The
// two values MUST stay in sync: 24 hours.
const guestTokenTTL = 24 * time.Hour

const (
	roomKeyPrefix   = "room:"
	inviteKeyPrefix = "invite:"
	publicRoomsKey  = "rooms:public:"
	guestKeyPrefix  = "guest:"
	roomDefaultTTL  = 24 * time.Hour
)

type RoomRepository interface {
	Save(ctx context.Context, room *domain.Room) error
	GetByID(ctx context.Context, id string) (*domain.Room, error)
	GetByInviteCode(ctx context.Context, inviteCode string) (*domain.Room, error)
	Delete(ctx context.Context, id string, inviteCode string) error
	ListPublic(ctx context.Context, language string, paging domain.Paging) (*domain.PageOf[domain.Room], error)

	// Guest tokens are short-lived, room-bound tokens for anonymous players.
	SaveGuest(ctx context.Context, g *domain.GuestAuth) error
	GetGuest(ctx context.Context, token string) (*domain.GuestAuth, error)
	DeleteGuest(ctx context.Context, token string) error
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

func (r *ephemeralRoomRepo) SaveGuest(ctx context.Context, g *domain.GuestAuth) error {
	if g == nil || g.Token == "" {
		return errors.New("invalid guest: token required")
	}
	data, _ := json.Marshal(g)
	return r.cache.Set(ctx, guestKeyPrefix+g.Token, string(data), guestTokenTTL)
}

func (r *ephemeralRoomRepo) GetGuest(ctx context.Context, token string) (*domain.GuestAuth, error) {
	data, err := r.cache.Get(ctx, guestKeyPrefix+token)
	if err != nil || data == "" {
		return nil, nil
	}
	var g domain.GuestAuth
	if err := json.Unmarshal([]byte(data), &g); err != nil {
		return nil, nil
	}
	if time.Now().After(g.ExpiresAt) {
		return nil, nil
	}
	return &g, nil
}

func (r *ephemeralRoomRepo) DeleteGuest(ctx context.Context, token string) error {
	return r.cache.Delete(ctx, guestKeyPrefix+token)
}
