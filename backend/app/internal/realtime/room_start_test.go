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
	cacheinfra "drawo/internal/infrastructure/cache"
)

// newTestCacheRepo builds an in-memory cache + room repository pair for tests.
func newTestCacheRepo(t *testing.T) (repositories.CacheRepository, repositories.RoomRepository) {
	t.Helper()
	cacheClient, err := cacheinfra.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cacheClient.Close() })
	return cacheClient, repositories.NewRoomRepo(cacheClient)
}

// newTestRoom builds a Room wired to an in-memory room repository so persist()
// exercises the real Save path instead of the no-op nil-repo branch.
func newTestRoom(t *testing.T, state *domain.Room) *Room {
	t.Helper()
	_, roomRepo := newTestCacheRepo(t)
	require.NoError(t, roomRepo.Save(context.Background(), state))
	return NewRoom(state, func(string, string) {}, roomRepo, nil, nil, nil, nil)
}

func TestNewRoom_BootsIntoCountdownWhenRedisSaysPlaying(t *testing.T) {
	room := NewRoom(&domain.Room{
		ID:           "r-bootstrap",
		OwnerID:      "o",
		State:        domain.RoomStatePlaying,
		CurrentRound: 1,
		MinPlayers:   2, MaxPlayers: 8,
	}, func(string, string) {}, nil, nil, nil, nil, nil)
	assert.Equal(t, GameStateCountdown, room.gameState)
	assert.Equal(t, 0, room.state.CurrentRound, "REST StartGame sentinel must be normalized to 0 for the running goroutine")
}

func TestRoom_PersistFlushesStateToRepo(t *testing.T) {
	// Without a repo, persist is a safe no-op.
	room := NewRoom(&domain.Room{
		ID: "r-persist", OwnerID: "o", State: domain.RoomStateLobby,
		MinPlayers: 2, MaxPlayers: 8,
	}, func(string, string) {}, nil, nil, nil, nil, nil)
	assert.NoError(t, room.persist())

	// Wire in a real repo and verify state is saved.
	_, repo := newTestCacheRepo(t)
	require.NoError(t, repo.Save(context.Background(), room.state))
	room.roomRepo = repo
	room.state.Name = "renamed"
	require.NoError(t, room.persist())
	got, err := repo.GetByID(context.Background(), room.state.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "renamed", got.Name)
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestRoom_HandleGameEvent_StartRejectsUnauthorized(t *testing.T) {
	room := newTestRoomLobby(t, "owner-1", 2)
	nonOwner := makeClient("c2", "u2", "Friend")
	owner := makeClient("c1", "owner-1", "Owner")
	room.clients[owner.ID] = owner
	room.clients[nonOwner.ID] = nonOwner
	room.players[owner.UserID] = &PlayerState{UserID: owner.UserID, IsOnline: true}
	room.players[nonOwner.UserID] = &PlayerState{UserID: nonOwner.UserID, IsOnline: true}
	room.playerOrder = []string{owner.UserID, nonOwner.UserID}

	// Not the owner.
	room.handleGameEvent(&RoomEvent{Client: nonOwner, Payload: json.RawMessage(`{"event":"start"}`)})
	assert.Equal(t, GameStateWaitingForPlayers, room.gameState)
	// Drain the error envelope to non-owner.
	select {
	case <-nonOwner.Send:
	default:
	}

	// Not enough players (only owner online after removing friend).
	delete(room.clients, nonOwner.ID)
	room.players[nonOwner.UserID].IsOnline = false
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: json.RawMessage(`{"event":"start"}`)})
	assert.Equal(t, GameStateWaitingForPlayers, room.gameState)
	select {
	case <-owner.Send:
	default:
	}

	// Invalid JSON payload.
	room.players[nonOwner.UserID].IsOnline = true
	room.clients[nonOwner.ID] = nonOwner
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: json.RawMessage(`{bad`)})
	assert.Equal(t, GameStateWaitingForPlayers, room.gameState)
	select {
	case <-owner.Send:
	default:
	}

	// Unknown event.
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: json.RawMessage(`{"event":"fly"}`)})
	assert.Equal(t, GameStateWaitingForPlayers, room.gameState)
	select {
	case <-owner.Send:
	default:
	}
}

func TestRoom_HandleGameEvent_StartTransitionsToCountdown(t *testing.T) {
	room := newTestRoomLobby(t, "owner-1", 2)
	origCD := countdownDuration
	countdownDuration = 10 * time.Second
	t.Cleanup(func() { countdownDuration = origCD })

	owner := makeClient("c1", "owner-1", "Owner")
	friend := makeClient("c2", "u2", "Friend")
	room.clients[owner.ID] = owner
	room.clients[friend.ID] = friend
	room.players[owner.UserID] = &PlayerState{UserID: owner.UserID, IsOnline: true, IsOwner: true}
	room.players[friend.UserID] = &PlayerState{UserID: friend.UserID, IsOnline: true}
	room.playerOrder = []string{owner.UserID, friend.UserID}

	payload, _ := json.Marshal(GameEventPayload{Event: "start"})
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: payload})

	assert.Equal(t, GameStateCountdown, room.gameState)
	assert.Equal(t, domain.RoomStatePlaying, room.state.State)
	assert.NotNil(t, room.timer, "countdown timer must be started")
}

func TestRoom_HandleGameEvent_StartRejectsWhenAlreadyStarted(t *testing.T) {
	room := newTestRoomLobby(t, "owner-1", 2)
	origCD := countdownDuration
	countdownDuration = 10 * time.Second
	t.Cleanup(func() { countdownDuration = origCD })

	owner := makeClient("c1", "owner-1", "Owner")
	friend := makeClient("c2", "u2", "Friend")
	room.clients[owner.ID] = owner
	room.clients[friend.ID] = friend
	room.players[owner.UserID] = &PlayerState{UserID: owner.UserID, IsOnline: true}
	room.players[friend.UserID] = &PlayerState{UserID: friend.UserID, IsOnline: true}
	room.playerOrder = []string{owner.UserID, friend.UserID}

	payload, _ := json.Marshal(GameEventPayload{Event: "start"})
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: payload})
	require.Equal(t, GameStateCountdown, room.gameState)

	// Second start must fail with "already in progress".
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: payload})
	select {
	case <-owner.Send:
	default:
		t.Fatal("expected error envelope to owner")
	}
}

func TestRoom_HandleGameEvent_StartEnforcesMinPlayersAndState(t *testing.T) {
	room := newTestRoomLobby(t, "owner-1", 3)
	origCD := countdownDuration
	countdownDuration = 10 * time.Second
	t.Cleanup(func() { countdownDuration = origCD })

	owner := makeClient("c1", "owner-1", "Owner")
	friend := makeClient("c2", "u2", "Friend")
	room.clients[owner.ID] = owner
	room.clients[friend.ID] = friend
	room.players[owner.UserID] = &PlayerState{UserID: owner.UserID, IsOnline: true, IsOwner: true}
	room.players[friend.UserID] = &PlayerState{UserID: friend.UserID, IsOnline: true}
	room.playerOrder = []string{owner.UserID, friend.UserID}

	// Only 2 online but min is 3.
	payload, _ := json.Marshal(GameEventPayload{Event: "start"})
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: payload})
	assert.Equal(t, GameStateWaitingForPlayers, room.gameState)
	// Drain the error envelope.
	select {
	case <-owner.Send:
	default:
		t.Fatal("expected error envelope for not-enough-players")
	}
}

func TestRoom_HandleGameEvent_ChooseWordRejectsNonDrawerOrWrongState(t *testing.T) {
	room := newTestRoomLobby(t, "owner-1", 2)
	owner := makeClient("c1", "owner-1", "Owner")
	room.clients[owner.ID] = owner
	room.players[owner.UserID] = &PlayerState{UserID: owner.UserID, IsOnline: true, IsOwner: true}

	// In lobby, choose_word is rejected.
	bad, _ := json.Marshal(map[string]string{"event": "choose_word", "group_id": "g1"})
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: bad})
	select {
	case <-owner.Send:
	default:
		t.Fatal("expected error envelope for choose_word in lobby")
	}

	// Word selection state but not the drawer.
	room.gameState = GameStateWordSelection
	room.state.CurrentDrawerID = "other"
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: bad})
	select {
	case <-owner.Send:
	default:
		t.Fatal("expected error envelope for non-drawer choose_word")
	}

	// Invalid JSON payload.
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: json.RawMessage(`{"event":"choose_word"}`)})

	// Valid drawer chooses first suggestion when none loaded.
	room.state.CurrentDrawerID = owner.UserID
	room.suggestedWords = []WordCandidate{{GroupID: "g1", Text: "cat", Points: 1}}
	good, _ := json.Marshal(map[string]string{"event": "choose_word", "group_id": "g1"})
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: good})
	assert.Equal(t, GameStateDrawing, room.gameState)
	assert.NotNil(t, room.currentWord)
	assert.Equal(t, "cat", room.currentWord.Text)
}

func TestRoom_HandleGameEvent_ReportRejectsInvalidJSON(t *testing.T) {
	room := newTestRoomLobby(t, "owner-1", 2)
	owner := makeClient("c1", "owner-1", "Owner")
	room.clients[owner.ID] = owner
	room.players[owner.UserID] = &PlayerState{UserID: owner.UserID, IsOnline: true}
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: json.RawMessage(`{"event":"report"}`)})
	select {
	case <-owner.Send:
	default:
		t.Fatal("expected error envelope for invalid report payload")
	}
}

func TestRoom_PlayerSnapshotIncludesOwnerAndGuestFlags(t *testing.T) {
	room := newTestRoomLobby(t, "owner-1", 2)
	owner := makeClient("c1", "owner-1", "Owner")
	guestClient := makeClient("c2", "guest:g1", "Alice")
	room.players[owner.UserID] = &PlayerState{UserID: owner.UserID, Username: "Owner", IsOwner: true, IsOnline: true, Score: 0}
	room.players[guestClient.UserID] = &PlayerState{UserID: guestClient.UserID, Username: "Alice", IsGuest: true, IsOnline: true, Score: 0}
	room.playerOrder = []string{owner.UserID, guestClient.UserID}

	snap := room.playerSnapshot()
	require.Len(t, snap, 2)
	assert.True(t, snap[0].IsOwner)
	assert.False(t, snap[0].IsGuest)
	assert.False(t, snap[1].IsOwner)
	assert.True(t, snap[1].IsGuest)
	assert.Equal(t, "Alice", snap[1].Username)
}

func TestRoom_IsRankedGame(t *testing.T) {
	room := newTestRoomLobby(t, "owner-1", 2)
	room.state.Type = domain.RoomTypePublic
	room.state.WordSource = domain.WordSourceDefault
	assert.True(t, room.isRankedGame())
	room.state.WordSource = domain.WordSourceCustom
	assert.False(t, room.isRankedGame())
	room.state.Type = domain.RoomTypePrivate
	room.state.WordSource = domain.WordSourceDefault
	assert.False(t, room.isRankedGame())
}

func TestDisplayName(t *testing.T) {
	assert.Equal(t, "player", displayName(nil))
	assert.Equal(t, "ghost", displayName(&PlayerState{UserID: "ghost"}))
	assert.Equal(t, "Alice", displayName(&PlayerState{Username: "Alice", UserID: "u-1"}))
}

func TestNewTestRoomLobby_HelperSanity(t *testing.T) {
	room := newTestRoomLobby(t, "o", 2)
	assert.Equal(t, GameStateWaitingForPlayers, room.gameState)
	assert.Equal(t, 2, room.minPlayers())
	assert.Equal(t, 8, room.maxPlayers())
}

func newTestRoomLobby(t *testing.T, ownerID string, minPlayers int) *Room {
	t.Helper()
	return NewRoom(&domain.Room{
		ID:         "r-test",
		Name:       "Test",
		OwnerID:    ownerID,
		State:      domain.RoomStateLobby,
		Type:       domain.RoomTypePrivate,
		Language:   "en",
		WordSource: domain.WordSourceDefault,
		MinPlayers: minPlayers,
		MaxPlayers: 8,
		RoundTime:  40,
		MaxRounds:  2,
	}, func(string, string) {}, nil, nil, nil, nil, nil)
}

func makeClient(id, userID, username string) *Client {
	return &Client{
		ID:       id,
		UserID:   userID,
		Username: username,
		Send:     make(chan []byte, 32),
		Done:     make(chan struct{}),
	}
}

func TestRoom_HandleGameEvent_PlayAgain(t *testing.T) {
	room := newTestRoomLobby(t, "owner-1", 2)
	owner := makeClient("c1", "owner-1", "Owner")
	friend := makeClient("c2", "u2", "Friend")
	room.clients[owner.ID] = owner
	room.clients[friend.ID] = friend
	room.players[owner.UserID] = &PlayerState{UserID: owner.UserID, IsOnline: true, Score: 120, GuessedWord: true}
	room.players[friend.UserID] = &PlayerState{UserID: friend.UserID, IsOnline: true, Score: 80}
	room.playerOrder = []string{owner.UserID, friend.UserID}

	// Rejected while the game is not finished.
	room.gameState = GameStateDrawing
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: json.RawMessage(`{"event":"play_again"}`)})
	assert.Equal(t, GameStateDrawing, room.gameState)

	// Rejected for non-owners even when finished.
	room.gameState = GameStateGameEnd
	room.handleGameEvent(&RoomEvent{Client: friend, Payload: json.RawMessage(`{"event":"play_again"}`)})
	assert.Equal(t, GameStateGameEnd, room.gameState)

	// Owner restart: scores reset, fresh countdown, canvas cleared.
	room.canvasOps = append(room.canvasOps, DrawOperation{Op: DrawOpStroke, ID: "x"})
	room.handleGameEvent(&RoomEvent{Client: owner, Payload: json.RawMessage(`{"event":"play_again"}`)})
	assert.Equal(t, GameStateCountdown, room.gameState)
	assert.Equal(t, int64(0), room.players[owner.UserID].Score)
	assert.False(t, room.players[owner.UserID].GuessedWord)
	assert.Equal(t, 0, room.state.CurrentRound)
	assert.Empty(t, room.canvasOps)
	require.Equal(t, domain.RoomStatePlaying, room.state.State)
}
