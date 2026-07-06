package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports"
	"drawo/pkg/errors"
)

// RoomService implements ephemeral room creation and discovery orchestration.
type RoomService struct {
	roomRepo ports.RoomRepository
}

// NewRoomService creates a new application service for ephemeral room management.
func NewRoomService(roomRepo ports.RoomRepository) ports.RoomService {
	return &RoomService{roomRepo: roomRepo}
}

// CreateRoom initializes an ephemeral room metadata struct and saves it to discovery storage.
func (s *RoomService) CreateRoom(ctx context.Context, name, ownerID string, roomType domain.RoomType, language string, maxPlayers, roundTime, maxRounds int) (*domain.Room, error) {
	roomID := uuid.New().String()
	var inviteCode string
	if roomType == domain.RoomTypePrivate {
		inviteCode = generateInviteCode()
	}

	room := &domain.Room{
		ID:           roomID,
		Name:         name,
		InviteCode:   inviteCode,
		OwnerID:      ownerID,
		Type:         roomType,
		Language:     strings.ToLower(language),
		State:        domain.RoomStateLobby,
		MaxPlayers:   maxPlayers,
		RoundTime:    roundTime,
		MaxRounds:    maxRounds,
		CurrentRound: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.roomRepo.Save(ctx, room); err != nil {
		return nil, errors.New(errors.ErrInternalServer, "failed to register ephemeral room discovery record")
	}

	return room, nil
}

// GetRoom retrieves ephemeral room discovery metadata by ID.
func (s *RoomService) GetRoom(ctx context.Context, roomID string) (*domain.Room, error) {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServer, "failed to look up room")
	}
	if room == nil {
		return nil, errors.New(errors.ErrNotFound, "room not found or has been closed")
	}
	return room, nil
}

// JoinByInviteCode resolves a private invitation code to an ephemeral room.
func (s *RoomService) JoinByInviteCode(ctx context.Context, inviteCode string) (*domain.Room, error) {
	room, err := s.roomRepo.GetByInviteCode(ctx, inviteCode)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServer, "failed to look up invite code")
	}
	if room == nil {
		return nil, errors.New(errors.ErrNotFound, "invite code is invalid or room has closed")
	}
	return room, nil
}

func generateInviteCode() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

var _ ports.RoomService = (*RoomService)(nil)
