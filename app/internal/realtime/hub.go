package realtime

import (
	"context"
	"errors"
	"sync"
	"time"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
)

type Hub struct {
	mu       sync.RWMutex
	rooms    map[string]*Room
	roomRepo repositories.RoomRepository
}

func NewHub(roomRepo repositories.RoomRepository) *Hub {
	return &Hub{
		rooms:    make(map[string]*Room),
		roomRepo: roomRepo,
	}
}

func (h *Hub) CreateRoom(ctx context.Context, state *domain.Room) (*Room, error) {
	if state == nil || state.ID == "" {
		return nil, errors.New("invalid room: ID required")
	}
	if err := h.roomRepo.Save(ctx, state); err != nil {
		return nil, err
	}
	return h.startRoom(ctx, state), nil
}

// GetOrStartRoom returns the local room goroutine for this instance, creating it
// from cached/discovery metadata when necessary. This is the bridge between the
// distributed room registry (Redis/cache) and the in-process per-room goroutine.
func (h *Hub) GetOrStartRoom(ctx context.Context, roomID string) (*Room, error) {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()
	if exists {
		return room, nil
	}

	state, err := h.roomRepo.GetByID(ctx, roomID)
	if err != nil || state == nil || state.ID == "" {
		return nil, errors.New("room not found")
	}
	return h.startRoom(ctx, state), nil
}

func (h *Hub) startRoom(ctx context.Context, state *domain.Room) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing, ok := h.rooms[state.ID]; ok {
		return existing
	}
	room := NewRoom(state, h.onRoomClosed)
	h.rooms[state.ID] = room
	go room.Run(ctx)
	return room
}

func (h *Hub) GetRoom(ctx context.Context, roomID string) (*Room, *domain.Room, error) {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()
	if exists {
		return room, room.state, nil
	}
	state, err := h.roomRepo.GetByID(ctx, roomID)
	if err != nil || state == nil {
		return nil, nil, errors.New("room not found")
	}
	return nil, state, nil
}

func (h *Hub) GetRoomByInviteCode(ctx context.Context, inviteCode string) (*domain.Room, error) {
	return h.roomRepo.GetByInviteCode(ctx, inviteCode)
}

func (h *Hub) JoinRoom(ctx context.Context, roomID string, client *Client) error {
	room, err := h.GetOrStartRoom(ctx, roomID)
	if err != nil {
		return err
	}
	client.RoomID = roomID
	if !room.Dispatch(&RoomEvent{Type: EventJoin, Client: client, Timestamp: time.Now()}) {
		return errors.New("room overloaded")
	}
	return nil
}

func (h *Hub) DispatchToRoom(roomID string, event *RoomEvent) error {
	h.mu.RLock()
	room, ok := h.rooms[roomID]
	h.mu.RUnlock()
	if !ok {
		return errors.New("room not found")
	}
	if !room.Dispatch(event) {
		return errors.New("room overloaded")
	}
	return nil
}

func (h *Hub) LeaveRoom(roomID string, client *Client) {
	h.mu.RLock()
	room, ok := h.rooms[roomID]
	h.mu.RUnlock()
	if ok {
		room.Dispatch(&RoomEvent{Type: EventLeave, Client: client, Timestamp: time.Now()})
	}
}

func (h *Hub) onRoomClosed(roomID, inviteCode string) {
	h.mu.Lock()
	delete(h.rooms, roomID)
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = h.roomRepo.Delete(ctx, roomID, inviteCode)
}
