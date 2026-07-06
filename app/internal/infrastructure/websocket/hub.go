package websocket

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports"
	"drawo/pkg/logger"
)

// Hub coordinates active ephemeral room goroutines running on this server instance.
//
// DESIGN DECISION:
//   While individual room state is kept exclusively in memory within each room's goroutine,
//   the Hub uses `ports.RoomRepository` (backed by Redis/distributed cache) to register
//   discovery metadata and unique invite codes across server cluster instances.
//
//   When a room is closed or all players leave, the Hub automatically deletes the discovery
//   record and invalidates any associated invite code in the distributed cache.
type Hub struct {
	mu       sync.RWMutex
	rooms    map[string]*Room
	roomRepo ports.RoomRepository
}

// NewHub creates a thread-safe Hub for ephemeral room goroutines.
func NewHub(roomRepo ports.RoomRepository) *Hub {
	return &Hub{
		rooms:    make(map[string]*Room),
		roomRepo: roomRepo,
	}
}

// CreateRoom initializes an ephemeral room in memory and registers discovery/invite code in distributed cache.
func (h *Hub) CreateRoom(ctx context.Context, state *domain.Room) (*Room, error) {
	if state == nil || state.ID == "" {
		return nil, errors.New("room state with valid ID required")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.rooms[state.ID]; exists {
		return nil, errors.New("room already exists on this server instance")
	}

	// Persist transient discovery and invite code mapping in distributed cache
	if err := h.roomRepo.Save(ctx, state); err != nil {
		return nil, err
	}

	room := NewRoom(state, h.onRoomClosed)
	h.rooms[state.ID] = room
	go room.Run(ctx)

	return room, nil
}

// GetRoom retrieves an active room goroutine from memory or checks distributed cache discovery.
func (h *Hub) GetRoom(ctx context.Context, roomID string) (*Room, *domain.Room, error) {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()

	if exists {
		return room, room.state, nil
	}

	// Check distributed cache if room exists on another instance
	state, err := h.roomRepo.GetByID(ctx, roomID)
	if err != nil || state == nil {
		return nil, nil, errors.New("room not found")
	}

	return nil, state, nil
}

// GetRoomByInviteCode resolves an invite code via distributed cache discovery.
func (h *Hub) GetRoomByInviteCode(ctx context.Context, inviteCode string) (*domain.Room, error) {
	return h.roomRepo.GetByInviteCode(ctx, inviteCode)
}

// JoinRoom routes a connecting client to the ephemeral room's inbox channel.
func (h *Hub) JoinRoom(ctx context.Context, roomID string, client *Client) error {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()

	if !exists {
		return errors.New("room goroutine not running on this instance")
	}

	room.Dispatch(&RoomEvent{
		Type:      EventJoin,
		Client:    client,
		Timestamp: time.Now(),
	})
	return nil
}

// LeaveRoom notifies an active room goroutine that a player disconnected.
func (h *Hub) LeaveRoom(roomID string, client *Client) {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()

	if exists {
		room.Dispatch(&RoomEvent{
			Type:      EventLeave,
			Client:    client,
			Timestamp: time.Now(),
		})
	}
}

// onRoomClosed is triggered automatically when an ephemeral room goroutine terminates.
func (h *Hub) onRoomClosed(roomID, inviteCode string) {
	h.mu.Lock()
	delete(h.rooms, roomID)
	h.mu.Unlock()

	// Automatically invalidate discovery record and invite code across cluster
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.roomRepo.Delete(ctx, roomID, inviteCode); err != nil {
		logger.L.Error("failed to delete ephemeral room discovery record", slog.String("room_id", roomID), slog.String("error", err.Error()))
	}
}
