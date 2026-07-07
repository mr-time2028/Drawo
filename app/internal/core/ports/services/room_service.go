package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/pkg/errors"
)

type RoomService interface {
	CreateRoom(ctx context.Context, name, ownerID string, roomType domain.RoomType, language string, minPlayers, maxPlayers, roundTime, maxRounds int) (*domain.Room, error)
	GetRoom(ctx context.Context, roomID string) (*domain.Room, error)
	JoinByInviteCode(ctx context.Context, inviteCode string) (*domain.Room, error)
	SetCustomWords(ctx context.Context, roomID, userID string, words []string) error
}

type roomService struct {
	roomRepo repositories.RoomRepository
}

func NewRoomService(roomRepo repositories.RoomRepository) RoomService {
	return &roomService{roomRepo: roomRepo}
}

func (s *roomService) CreateRoom(ctx context.Context, name, ownerID string, roomType domain.RoomType, language string, minPlayers, maxPlayers, roundTime, maxRounds int) (*domain.Room, error) {
	room := &domain.Room{
		ID: uuid.New().String(),
		Name: name,
		OwnerID: ownerID,
		Type: roomType,
		Language: strings.ToLower(language),
		State: domain.RoomStateLobby,
		MinPlayers: minPlayers,
		MaxPlayers: maxPlayers,
		RoundTime: roundTime,
		MaxRounds: maxRounds,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if roomType == domain.RoomTypePrivate {
		b := make([]byte, 3); rand.Read(b)
		room.InviteCode = strings.ToUpper(hex.EncodeToString(b))
	}
	if err := s.roomRepo.Save(ctx, room); err != nil {
		return nil, errors.New(errors.ErrInternalServer, "failed to save room")
	}
	return room, nil
}

func (s *roomService) GetRoom(ctx context.Context, roomID string) (*domain.Room, error) {
	return s.roomRepo.GetByID(ctx, roomID)
}

func (s *roomService) JoinByInviteCode(ctx context.Context, inviteCode string) (*domain.Room, error) {
	return s.roomRepo.GetByInviteCode(ctx, inviteCode)
}

func (s *roomService) SetCustomWords(ctx context.Context, roomID, userID string, words []string) error {
	room, _ := s.roomRepo.GetByID(ctx, roomID)
	if room == nil || room.OwnerID != userID { return errors.New(errors.ErrForbidden, "unauthorized") }
	room.CustomWords = words
	return s.roomRepo.Save(ctx, room)
}
