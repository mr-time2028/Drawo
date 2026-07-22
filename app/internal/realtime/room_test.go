package realtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/internal/infrastructure/cache"
)

func TestEphemeralRoomGoroutine_Lifecycle(t *testing.T) {
	oldReconnectGrace := reconnectGrace
	reconnectGrace = 50 * time.Millisecond
	t.Cleanup(func() { reconnectGrace = oldReconnectGrace })

	cacheClient, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	defer cacheClient.Close()

	roomRepo := repositories.NewRoomRepo(cacheClient)
	hub := NewHub(roomRepo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	state := &domain.Room{
		ID:         "room-ephemeral-1",
		Name:       "Goroutine Test Room",
		InviteCode: "CODE123",
		Type:       domain.RoomTypePrivate,
		State:      domain.RoomStatePlaying,
		MinPlayers: 2,
		MaxPlayers: 8,
	}

	room, err := hub.CreateRoom(ctx, state)
	require.NoError(t, err)
	require.NotNil(t, room)

	// Verify room exists in Hub and discovery storage
	_, _, err = hub.GetRoom(ctx, "room-ephemeral-1")
	require.NoError(t, err)

	// Join client 1
	client1 := &Client{
		ID:       "c1",
		Username: "Alice",
		Send:     make(chan []byte, 20),
	}
	err = hub.JoinRoom(ctx, "room-ephemeral-1", client1)
	require.NoError(t, err)

	// Join client 2
	client2 := &Client{
		ID:       "c2",
		Username: "Bob",
		Send:     make(chan []byte, 20),
	}
	err = hub.JoinRoom(ctx, "room-ephemeral-1", client2)
	require.NoError(t, err)

	// Allow goroutine to process joins
	time.Sleep(50 * time.Millisecond)

	// Drain notifications
	for len(client2.Send) > 0 {
		<-client2.Send
	}

	room.gameState = GameStateDrawing

	// Client 1 sends a draw stroke event
	drawEvent := &RoomEvent{
		Type:      EventDraw,
		Client:    client1,
		Payload:   []byte(`{"op":"stroke","tool":"pencil","color":"#000000","size":4,"points":[{"x":10,"y":20},{"x":12,"y":22}]}`),
		Timestamp: time.Now(),
	}
	room.Dispatch(drawEvent)

	time.Sleep(50 * time.Millisecond)

	// Verify Client 2 received the draw
	require.NotEmpty(t, client2.Send)
	msg := <-client2.Send
	var env MessageEnvelope
	require.NoError(t, json.Unmarshal(msg, &env))
	assert.Equal(t, EventDraw, env.Type)

	// Client 1 leaves
	hub.LeaveRoom("room-ephemeral-1", client1)
	time.Sleep(50 * time.Millisecond)

	// Client 2 leaves
	hub.LeaveRoom("room-ephemeral-1", client2)
	time.Sleep(150 * time.Millisecond)

	// Verify room is removed from local hub map
	_, _, err = hub.GetRoom(ctx, "room-ephemeral-1")
	assert.Error(t, err)

	// Verify invite code is invalidated
	inviteLookup, _ := hub.GetRoomByInviteCode(ctx, "CODE123")
	assert.Nil(t, inviteLookup)
}

func TestHub_EdgeCases(t *testing.T) {
	cacheClient, _ := cache.NewClient(config.CacheConfig{Driver: "memory"})
	roomRepo := repositories.NewRoomRepo(cacheClient)
	hub := NewHub(roomRepo)
	ctx := context.Background()

	// Create room without ID should fail
	_, err := hub.CreateRoom(ctx, &domain.Room{})
	assert.Error(t, err)

	// Get non-existent room (checks cache too)
	_, _, err = hub.GetRoom(ctx, "ghost")
	assert.Error(t, err)

	// Test room in cache but NOT in local map (multi-instance simulation)
	roomInCache := &domain.Room{ID: "remote", Name: "remote"}
	roomRepo.Save(ctx, roomInCache)

	h, s, err := hub.GetRoom(ctx, "remote")
	assert.NoError(t, err)
	assert.Nil(t, h)
	assert.Equal(t, "remote", s.ID)

	// Join non-existent room
	err = hub.JoinRoom(ctx, "ghost", &Client{})
	assert.Error(t, err)
}
