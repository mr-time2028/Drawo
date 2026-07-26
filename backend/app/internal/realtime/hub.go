package realtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
)

const (
	defaultPublicRoomName = "Public Room"
	defaultLanguage       = "en"
	defaultMinPlayers     = 2
	defaultMaxPlayers     = 8
	defaultRoundTime      = 80
	defaultMaxRounds      = 3
)

type Hub struct {
	mu             sync.RWMutex
	rooms          map[string]*Room
	loads          map[string]int // Hub-owned approximate client counts for matchmaking.
	activeRooms    map[string]string
	roomRepo       repositories.RoomRepository
	contentRepo    repositories.ContentRepository
	profileRepo    repositories.ProfileRepository
	reputationRepo repositories.ReputationRepository
	reportRepo     repositories.ReportRepository
	userRepo       repositories.UserRepository
	sessionRepo    repositories.SessionRepository
}

func NewHub(roomRepo repositories.RoomRepository) *Hub {
	return NewHubWithDependencies(roomRepo, nil, nil)
}

func NewHubWithDependencies(roomRepo repositories.RoomRepository, contentRepo repositories.ContentRepository, profileRepo repositories.ProfileRepository, extraRepos ...interface{}) *Hub {
	var repRepo repositories.ReputationRepository
	var reportRepo repositories.ReportRepository
	var userRepo repositories.UserRepository
	var sessionRepo repositories.SessionRepository
	for _, repo := range extraRepos {
		switch typed := repo.(type) {
		case repositories.ReputationRepository:
			repRepo = typed
		case repositories.ReportRepository:
			reportRepo = typed
		case repositories.UserRepository:
			userRepo = typed
		case repositories.SessionRepository:
			sessionRepo = typed
		}
	}
	return &Hub{
		rooms:          make(map[string]*Room),
		loads:          make(map[string]int),
		activeRooms:    make(map[string]string),
		roomRepo:       roomRepo,
		contentRepo:    contentRepo,
		profileRepo:    profileRepo,
		reputationRepo: repRepo,
		reportRepo:     reportRepo,
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
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
	room := NewRoom(state, h.onRoomClosed, h.contentRepo, h.profileRepo, h.reputationRepo, h.reportRepo, h.userRepo, h.sessionRepo)
	h.rooms[state.ID] = room
	if _, ok := h.loads[state.ID]; !ok {
		h.loads[state.ID] = 0
	}
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

// JoinByRequest is the backend-owned room matching entry point for WebSocket joins.
//
// Frontend should normally send an empty/default join payload for public
// matchmaking. The backend then finds a non-full public room or creates a new
// one. Private rooms use invite_code. room_id remains supported for reconnects,
// tests, and admin/debug flows, but normal public clients do not need to know it.
func (h *Hub) JoinByRequest(ctx context.Context, payload JoinPayload, client *Client) (string, error) {
	payload.Mode = strings.ToLower(strings.TrimSpace(payload.Mode))
	payload.RoomID = strings.TrimSpace(payload.RoomID)
	payload.InviteCode = strings.TrimSpace(payload.InviteCode)
	payload.Language = strings.ToLower(strings.TrimSpace(payload.Language))
	payload.CategoryID = strings.TrimSpace(payload.CategoryID)

	if payload.Mode == "reconnect" {
		h.mu.RLock()
		roomID := h.activeRooms[client.UserID]
		h.mu.RUnlock()
		if roomID == "" {
			return "", errors.New("no active room to reconnect")
		}
		return roomID, h.JoinRoom(ctx, roomID, client)
	}

	if payload.RoomID != "" {
		return payload.RoomID, h.JoinRoom(ctx, payload.RoomID, client)
	}

	if payload.InviteCode != "" || payload.Mode == "private" {
		if payload.InviteCode == "" {
			return "", errors.New("invite_code is required for private room join")
		}
		room, err := h.roomRepo.GetByInviteCode(ctx, payload.InviteCode)
		if err != nil || room == nil {
			return "", errors.New("private room not found")
		}
		return room.ID, h.JoinRoom(ctx, room.ID, client)
	}

	roomID, err := h.matchPublicRoom(ctx, payload)
	if err != nil {
		return "", err
	}
	return roomID, h.JoinRoom(ctx, roomID, client)
}

func (h *Hub) matchPublicRoom(ctx context.Context, payload JoinPayload) (string, error) {
	language := payload.Language
	if language == "" {
		language = defaultLanguage
	}

	h.mu.RLock()
	for id, room := range h.rooms {
		state := room.state
		maxPlayers := state.MaxPlayers
		if maxPlayers <= 0 {
			maxPlayers = defaultMaxPlayers
		}
		if state.Type == domain.RoomTypePublic && state.State == domain.RoomStateLobby && strings.EqualFold(state.Language, language) && state.CategoryID == payload.CategoryID && h.loads[id] < maxPlayers {
			h.mu.RUnlock()
			return id, nil
		}
	}
	h.mu.RUnlock()

	state := &domain.Room{
		ID:         uuid.New().String(),
		Name:       fmt.Sprintf("%s %s", defaultPublicRoomName, language),
		Type:       domain.RoomTypePublic,
		Language:   language,
		CategoryID: payload.CategoryID,
		State:      domain.RoomStateLobby,
		MinPlayers: defaultMinPlayers,
		MaxPlayers: defaultMaxPlayers,
		RoundTime:  defaultRoundTime,
		MaxRounds:  defaultMaxRounds,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	_, err := h.CreateRoom(ctx, state)
	if err != nil {
		return "", err
	}
	return state.ID, nil
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
	h.mu.Lock()
	h.loads[roomID]++
	if client.UserID != "" {
		h.activeRooms[client.UserID] = roomID
	}
	h.mu.Unlock()
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
		h.mu.Lock()
		if h.loads[roomID] > 0 {
			h.loads[roomID]--
		}
		h.mu.Unlock()
	}
}

func (h *Hub) onRoomClosed(roomID, inviteCode string) {
	h.mu.Lock()
	delete(h.rooms, roomID)
	delete(h.loads, roomID)
	for userID, activeRoomID := range h.activeRooms {
		if activeRoomID == roomID {
			delete(h.activeRooms, userID)
		}
	}
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = h.roomRepo.Delete(ctx, roomID, inviteCode)
}
