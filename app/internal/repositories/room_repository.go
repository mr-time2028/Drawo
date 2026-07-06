// Package repositories contains concrete persistence and discovery implementations.
package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports"
)

const (
	roomKeyPrefix   = "room:"
	inviteKeyPrefix = "invite:"
	publicRoomsKey  = "rooms:public:"
	roomDefaultTTL  = 24 * time.Hour // Auto-expire abandoned ephemeral rooms
)

// EphemeralRoomRepo implements ports.RoomRepository using non-relational distributed caching (Redis/memory).
//
// DESIGN DECISION:
//   Per architectural requirements, public and private rooms are NOT stored in the relational database.
//   Instead, they exist as ephemeral runtime objects. This repository coordinates transient discovery
//   metadata and unique invite code mappings across server instances using distributed caching.
type EphemeralRoomRepo struct {
	cache ports.CacheRepository
}

// NewRoomRepo creates an ephemeral room discovery repository backed by CacheRepository.
func NewRoomRepo(cache ports.CacheRepository) ports.RoomRepository {
	return &EphemeralRoomRepo{cache: cache}
}

// Save registers or updates ephemeral room metadata and invite codes in non-relational storage.
func (r *EphemeralRoomRepo) Save(ctx context.Context, room *domain.Room) error {
	if room == nil || room.ID == "" {
		return errors.New("invalid room: ID required")
	}

	data, err := json.Marshal(room)
	if err != nil {
		return fmt.Errorf("marshal room: %w", err)
	}

	roomKey := roomKeyPrefix + room.ID
	if err := r.cache.Set(ctx, roomKey, string(data), roomDefaultTTL); err != nil {
		return fmt.Errorf("save room metadata: %w", err)
	}

	// For private rooms with an invite code, maintain a reverse mapping to room ID.
	if room.InviteCode != "" {
		inviteKey := inviteKeyPrefix + strings.ToUpper(room.InviteCode)
		if err := r.cache.Set(ctx, inviteKey, room.ID, roomDefaultTTL); err != nil {
			return fmt.Errorf("save invite code mapping: %w", err)
		}
	}

	// If public, maintain discovery index entry.
	if room.Type == domain.RoomTypePublic && room.State == domain.RoomStateLobby {
		indexKey := publicRoomsKey + strings.ToLower(room.Language)
		// For simplicity in cache port, we store serialized ID index or append.
		_ = r.cache.Set(ctx, indexKey+":"+room.ID, room.ID, roomDefaultTTL)
	}

	return nil
}

// GetByID retrieves ephemeral room metadata by ID from cache.
func (r *EphemeralRoomRepo) GetByID(ctx context.Context, id string) (*domain.Room, error) {
	data, err := r.cache.Get(ctx, roomKeyPrefix+id)
	if err != nil {
		return nil, nil // Not found or expired
	}

	var room domain.Room
	if err := json.Unmarshal([]byte(data), &room); err != nil {
		return nil, fmt.Errorf("unmarshal room data: %w", err)
	}
	return &room, nil
}

// GetByInviteCode looks up the room associated with a private invitation code.
func (r *EphemeralRoomRepo) GetByInviteCode(ctx context.Context, inviteCode string) (*domain.Room, error) {
	if inviteCode == "" {
		return nil, nil
	}
	roomID, err := r.cache.Get(ctx, inviteKeyPrefix+strings.ToUpper(inviteCode))
	if err != nil || roomID == "" {
		return nil, nil
	}
	return r.GetByID(ctx, roomID)
}

// Delete removes room metadata and automatically invalidates any associated invite code upon room destruction.
func (r *EphemeralRoomRepo) Delete(ctx context.Context, id string, inviteCode string) error {
	keysToDelete := []string{roomKeyPrefix + id}
	if inviteCode != "" {
		keysToDelete = append(keysToDelete, inviteKeyPrefix+strings.ToUpper(inviteCode))
	}
	return r.cache.Delete(ctx, keysToDelete...)
}

// ListPublic retrieves active public matchmaking rooms from cache discovery.
func (r *EphemeralRoomRepo) ListPublic(ctx context.Context, language string, paging domain.Paging) (*domain.PageOf[domain.Room], error) {
	// Note: In Phase 9 when full matchmaking is active, this iterates discovery index keys.
	return &domain.PageOf[domain.Room]{
		Items:  []domain.Room{},
		Total:  0,
		Limit:  paging.Limit,
		Offset: paging.Offset,
	}, nil
}

// Compile-time check: EphemeralRoomRepo implements ports.RoomRepository.
var _ ports.RoomRepository = (*EphemeralRoomRepo)(nil)
