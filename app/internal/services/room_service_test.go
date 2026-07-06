package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/infrastructure/cache"
	"drawo/internal/repositories"
)

func TestRoomService_CreateAndJoin(t *testing.T) {
	cacheClient, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	defer cacheClient.Close()

	roomRepo := repositories.NewRoomRepo(cacheClient)
	svc := NewRoomService(roomRepo)
	ctx := context.Background()

	// Create private room
	privateRoom, err := svc.CreateRoom(ctx, "Private Fun", "owner-1", domain.RoomTypePrivate, "en", 6, 80, 3)
	require.NoError(t, err)
	require.NotNil(t, privateRoom)
	assert.NotEmpty(t, privateRoom.ID)
	assert.NotEmpty(t, privateRoom.InviteCode)
	assert.Equal(t, "Private Fun", privateRoom.Name)

	// Get room by ID
	fetched, err := svc.GetRoom(ctx, privateRoom.ID)
	require.NoError(t, err)
	assert.Equal(t, privateRoom.ID, fetched.ID)

	// Join by Invite Code
	joined, err := svc.JoinByInviteCode(ctx, privateRoom.InviteCode)
	require.NoError(t, err)
	assert.Equal(t, privateRoom.ID, joined.ID)

	// Create public room
	publicRoom, err := svc.CreateRoom(ctx, "Public Lobby", "owner-2", domain.RoomTypePublic, "en", 8, 80, 5)
	require.NoError(t, err)
	assert.Empty(t, publicRoom.InviteCode)
}
