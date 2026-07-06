package repositories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/infrastructure/cache"
)

func TestEphemeralRoomRepo_SaveAndGet(t *testing.T) {
	cacheClient, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	defer cacheClient.Close()

	repo := NewRoomRepo(cacheClient)
	ctx := context.Background()

	room := &domain.Room{
		ID:         "room-123",
		Name:       "Friendly Match",
		InviteCode: "INV999",
		OwnerID:    "user-1",
		Type:       domain.RoomTypePrivate,
		State:      domain.RoomStateLobby,
		MaxPlayers: 8,
	}

	// Test Save
	err = repo.Save(ctx, room)
	require.NoError(t, err)

	// Test GetByID
	fetched, err := repo.GetByID(ctx, "room-123")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "Friendly Match", fetched.Name)
	assert.Equal(t, "INV999", fetched.InviteCode)

	// Test GetByInviteCode
	byInvite, err := repo.GetByInviteCode(ctx, "inv999") // Case-insensitive lookup
	require.NoError(t, err)
	require.NotNil(t, byInvite)
	assert.Equal(t, "room-123", byInvite.ID)

	// Test Delete (destroys room and invalidates invite code)
	err = repo.Delete(ctx, "room-123", "INV999")
	require.NoError(t, err)

	deleted, err := repo.GetByID(ctx, "room-123")
	require.NoError(t, err)
	assert.Nil(t, deleted)

	deletedInvite, err := repo.GetByInviteCode(ctx, "INV999")
	require.NoError(t, err)
	assert.Nil(t, deletedInvite)
}
