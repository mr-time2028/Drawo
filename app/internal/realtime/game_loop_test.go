package realtime

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"drawo/internal/core/domain"
)

type fakeProfileRepo struct {
	profiles map[string]*domain.Profile
}

func (r *fakeProfileRepo) Insert(profile *domain.Profile) error {
	if r.profiles == nil {
		r.profiles = make(map[string]*domain.Profile)
	}
	copy := *profile
	r.profiles[profile.UserID] = &copy
	return nil
}

func (r *fakeProfileRepo) GetByUserID(userID string) (*domain.Profile, error) {
	profile := r.profiles[userID]
	if profile == nil {
		return nil, errors.New("not found")
	}
	copy := *profile
	return &copy, nil
}

func (r *fakeProfileRepo) Update(profile *domain.Profile) error {
	copy := *profile
	r.profiles[profile.UserID] = &copy
	return nil
}

func newGameTestRoom(profileRepo *fakeProfileRepo) (*Room, *Client, *Client) {
	state := &domain.Room{ID: "game-room", Type: domain.RoomTypePublic, Language: "en", State: domain.RoomStateLobby, MinPlayers: 2, MaxPlayers: 8, RoundTime: 30, MaxRounds: 1}
	room := NewRoom(state, func(string, string) {}, nil, profileRepo, nil, nil)
	drawer := &Client{ID: "drawer-conn", UserID: "drawer", Username: "Alice", Send: make(chan []byte, 50), Done: make(chan struct{})}
	guesser := &Client{ID: "guesser-conn", UserID: "guesser", Username: "Bob", Send: make(chan []byte, 50), Done: make(chan struct{})}
	return room, drawer, guesser
}

func TestGameLoopStartsWhenEnoughPlayersAndHandlesCorrectGuess(t *testing.T) {
	oldCountdown := countdownDuration
	oldWordSelection := wordSelectionDuration
	oldRoundEnd := roundEndDuration
	oldLeaderboard := leaderboardDuration
	countdownDuration = time.Hour
	wordSelectionDuration = time.Hour
	roundEndDuration = time.Hour
	leaderboardDuration = time.Hour
	t.Cleanup(func() {
		countdownDuration = oldCountdown
		wordSelectionDuration = oldWordSelection
		roundEndDuration = oldRoundEnd
		leaderboardDuration = oldLeaderboard
	})

	profiles := &fakeProfileRepo{profiles: map[string]*domain.Profile{
		"drawer":  {UserID: "drawer", ReputationScore: 10000},
		"guesser": {UserID: "guesser", ReputationScore: 10000},
	}}
	room, drawer, guesser := newGameTestRoom(profiles)

	room.handleEvent(&RoomEvent{Type: EventJoin, Client: drawer, Timestamp: time.Now()})
	assert.Equal(t, GameStateWaitingForPlayers, room.gameState)
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: guesser, Timestamp: time.Now()})
	assert.Equal(t, GameStateCountdown, room.gameState)

	room.handleTimer()
	assert.Equal(t, GameStateWordSelection, room.gameState)
	assert.Equal(t, "drawer", room.state.CurrentDrawerID)
	assert.Len(t, room.suggestedWords, 3)

	room.chooseWord(room.suggestedWords[0].GroupID)
	assert.Equal(t, GameStateDrawing, room.gameState)
	assert.NotNil(t, room.currentWord)

	guessPayload, _ := json.Marshal(ChatPayload{Text: room.currentWord.Text})
	room.handleEvent(&RoomEvent{Type: EventChat, Client: guesser, Payload: guessPayload, Timestamp: time.Now()})

	assert.True(t, room.players[guesser.UserID].GuessedWord)
	assert.Greater(t, room.players[guesser.UserID].Score, int64(0))
	assert.Greater(t, room.players[drawer.UserID].Score, int64(0))
	assert.Equal(t, GameStateRoundEnd, room.gameState)

	room.handleTimer()
	assert.Equal(t, GameStateLeaderboard, room.gameState)
	room.handleTimer()
	assert.Equal(t, GameStateGameEnd, room.gameState)

	drawerProfile, _ := profiles.GetByUserID("drawer")
	guesserProfile, _ := profiles.GetByUserID("guesser")
	assert.Greater(t, drawerProfile.ReputationScore, int64(10000))
	assert.Greater(t, guesserProfile.ReputationScore, int64(10000))
}

func TestDrawerCannotChatDuringDrawing(t *testing.T) {
	room, drawer, guesser := newGameTestRoom(nil)
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: drawer, Timestamp: time.Now()})
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: guesser, Timestamp: time.Now()})
	room.startWordSelection()
	room.chooseWord(room.suggestedWords[0].GroupID)
	drainClient(drawer)

	payload, _ := json.Marshal(ChatPayload{Text: "hint"})
	room.handleEvent(&RoomEvent{Type: EventChat, Client: drawer, Payload: payload, Timestamp: time.Now()})

	msg := nextEnvelope(t, drawer)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "drawer_chat_blocked")
}

func TestGuessNormalizationEnglishAndPersian(t *testing.T) {
	assert.Equal(t, NormalizeGuess("Apple!", "en"), NormalizeGuess("a p p l e", "en"))
	assert.Equal(t, NormalizeGuess("سیب", "fa"), NormalizeGuess("س ي ب", "fa"))
	assert.Equal(t, NormalizeGuess("کباب", "fa"), NormalizeGuess("كباب", "fa"))
}

func TestAbandoningActiveGameDecreasesReputation(t *testing.T) {
	profiles := &fakeProfileRepo{profiles: map[string]*domain.Profile{
		"drawer":  {UserID: "drawer", ReputationScore: 10000},
		"guesser": {UserID: "guesser", ReputationScore: 10000},
	}}
	room, drawer, guesser := newGameTestRoom(profiles)
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: drawer, Timestamp: time.Now()})
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: guesser, Timestamp: time.Now()})
	room.startWordSelection()
	room.chooseWord(room.suggestedWords[0].GroupID)

	room.handleEvent(&RoomEvent{Type: EventLeave, Client: drawer, Timestamp: time.Now()})
	room.endGame()

	profile, _ := profiles.GetByUserID("drawer")
	assert.Less(t, profile.ReputationScore, int64(10000))
	assert.True(t, room.players[drawer.UserID].Abandoned)
}
