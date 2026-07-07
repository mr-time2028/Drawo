package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/internal/infrastructure/cache"
)

type failRoomRepo struct {
    repositories.RoomRepository
}
func (f *failRoomRepo) Save(ctx context.Context, room *domain.Room) error {
    return assert.AnError
}
func (f *failRoomRepo) GetByID(ctx context.Context, id string) (*domain.Room, error) {
    return nil, assert.AnError
}

func TestRoomService_CreateAndJoin(t *testing.T) {
	cacheClient, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	defer cacheClient.Close()

	roomRepo := repositories.NewRoomRepo(cacheClient)
	svc := NewRoomService(roomRepo)
	ctx := context.Background()

	// Create private room
	privateRoom, err := svc.CreateRoom(ctx, "Private Fun", "owner-1", domain.RoomTypePrivate, "en", 2, 6, 80, 3)
	require.NoError(t, err)
	require.NotNil(t, privateRoom)
	assert.NotEmpty(t, privateRoom.ID)
	assert.NotEmpty(t, privateRoom.InviteCode)
	assert.Equal(t, "Private Fun", privateRoom.Name)
	assert.Equal(t, 2, privateRoom.MinPlayers)

	// Get room by ID
	fetched, err := svc.GetRoom(ctx, privateRoom.ID)
	require.NoError(t, err)
	assert.Equal(t, privateRoom.ID, fetched.ID)

	// Join by Invite Code
	joined, err := svc.JoinByInviteCode(ctx, privateRoom.InviteCode)
	require.NoError(t, err)
	assert.Equal(t, privateRoom.ID, joined.ID)

	// Create public room
	publicRoom, err := svc.CreateRoom(ctx, "Public Lobby", "owner-2", domain.RoomTypePublic, "en", 3, 8, 80, 5)
	require.NoError(t, err)
	assert.Empty(t, publicRoom.InviteCode)
	assert.Equal(t, 3, publicRoom.MinPlayers)

	// Test SetCustomWords
	words := []string{"apple", "banana", "cherry"}
	err = svc.SetCustomWords(ctx, privateRoom.ID, "owner-1", words)
	require.NoError(t, err)

	updated, err := svc.GetRoom(ctx, privateRoom.ID)
	require.NoError(t, err)
	assert.Equal(t, words, updated.CustomWords)

	// Test SetCustomWords Unauthorized
	err = svc.SetCustomWords(ctx, privateRoom.ID, "not-owner", words)
	assert.Error(t, err)

	// Test SetCustomWords Room Not Found
	err = svc.SetCustomWords(ctx, "non-existent", "owner-1", words)
	assert.Error(t, err)
}

func TestRoomService_Failures(t *testing.T) {
    svc := NewRoomService(&failRoomRepo{})
    ctx := context.Background()

    _, err := svc.CreateRoom(ctx, "fail", "o", domain.RoomTypePublic, "en", 1, 10, 10, 10)
    assert.Error(t, err)

    _, err = svc.GetRoom(ctx, "1")
    assert.Error(t, err)

    err = svc.SetCustomWords(ctx, "1", "o", []string{})
    assert.Error(t, err)
}
