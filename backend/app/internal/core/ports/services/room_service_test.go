package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/internal/infrastructure/cache"
)

// --- Fake repository for failure-path testing --------------------------------

type fakeRoomRepo struct {
	repositories.RoomRepository
	saveFn        func(ctx context.Context, room *domain.Room) error
	getByIDFn     func(ctx context.Context, id string) (*domain.Room, error)
	getByInviteFn func(ctx context.Context, code string) (*domain.Room, error)
	saved         *domain.Room
}

func (f *fakeRoomRepo) Save(ctx context.Context, room *domain.Room) error {
	f.saved = room
	if f.saveFn != nil {
		return f.saveFn(ctx, room)
	}
	return nil
}
func (f *fakeRoomRepo) GetByID(ctx context.Context, id string) (*domain.Room, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (f *fakeRoomRepo) GetByInviteCode(ctx context.Context, code string) (*domain.Room, error) {
	if f.getByInviteFn != nil {
		return f.getByInviteFn(ctx, code)
	}
	return nil, nil
}

// --- Helpers -----------------------------------------------------------------

func newTestSvc(t *testing.T) (RoomService, repositories.RoomRepository) {
	t.Helper()
	cacheClient, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cacheClient.Close() })
	repo := repositories.NewRoomRepo(cacheClient)
	return NewRoomService(repo), repo
}

func validCustomCats() []domain.CustomCategory {
	return []domain.CustomCategory{
		{
			Name: "Animals",
			Words: map[int][]string{
				domain.WordPointsEasy:   {"cat", "dog"},
				domain.WordPointsMedium: {"tiger", "zebra"},
				domain.WordPointsHard:   {"platypus", "elephant", "octopus"},
			},
		},
	}
}

func validSettings() RoomSettings {
	return RoomSettings{
		Name:       "Hamid's Room",
		Language:   "en",
		MaxPlayers: 8, MaxRounds: 3, RoundTime: 80,
		WordSource: domain.WordSourceDefault,
	}
}

// --- createRoom tests --------------------------------------------------------

func TestCreateRoom_DefaultsAndPersists(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()

	// Minimal settings — everything else should default.
	room, err := svc.CreateRoom(ctx, "owner-1", domain.RoomTypePrivate, RoomSettings{
		Name: "Test",
	})
	require.NoError(t, err)
	require.NotNil(t, room)

	assert.NotEmpty(t, room.ID)
	assert.Len(t, room.InviteCode, domain.RoomInviteCodeLen)
	assert.Equal(t, "Test", room.Name)
	assert.Equal(t, domain.RoomTypePrivate, room.Type)
	assert.Equal(t, "en", room.Language) // default when empty
	assert.Equal(t, domain.RoomDefaultPlayers, room.MaxPlayers)
	assert.Equal(t, domain.RoomDefaultRounds, room.MaxRounds)
	assert.Equal(t, domain.RoomDefaultRoundTime, room.RoundTime)
	assert.Equal(t, domain.WordSourceDefault, room.WordSource)
	assert.Equal(t, domain.RoomStateLobby, room.State)
	assert.Equal(t, domain.RoomMinPlayers, room.MinPlayers)
	assert.False(t, room.HasPassword)
	assert.Empty(t, room.CustomCategories)
	assert.False(t, room.CreatedAt.IsZero())
}

func TestCreateRoom_PublicHasNoInviteCode(t *testing.T) {
	svc, _ := newTestSvc(t)
	room, err := svc.CreateRoom(context.Background(), "o", domain.RoomTypePublic, validSettings())
	require.NoError(t, err)
	assert.Empty(t, room.InviteCode)
	assert.Equal(t, domain.RoomTypePublic, room.Type)
}

func TestCreateRoom_InvalidName(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()

	_, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, RoomSettings{Name: "  "})
	assert.Error(t, err)

	_, err = svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, RoomSettings{Name: "ab"}) // 2 runes < 3
	assert.Error(t, err)

	long := strings.Repeat("a", domain.RoomMaxNameLength+1)
	_, err = svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, RoomSettings{Name: long})
	assert.Error(t, err)
}

func TestCreateRoom_PasswordHashing(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()

	pw := "secret123"
	room, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, RoomSettings{
		Name:     "PW room",
		Password: &pw,
	})
	require.NoError(t, err)
	assert.True(t, room.HasPassword)
	assert.NotEmpty(t, room.PasswordHash)
	assert.NotEqual(t, pw, room.PasswordHash) // hashed, not plaintext

	// Empty string pointer clears (no password).
	empty := ""
	room2, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, RoomSettings{
		Name:     "NoPW",
		Password: &empty,
	})
	require.NoError(t, err)
	assert.False(t, room2.HasPassword)
	assert.Empty(t, room2.PasswordHash)

	// Nil pointer means not set.
	room3, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, RoomSettings{Name: "Nil"})
	require.NoError(t, err)
	assert.False(t, room3.HasPassword)

	// Too short / too long rejected.
	short := "abc"
	_, err = svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, RoomSettings{Name: "X", Password: &short})
	assert.Error(t, err)

	longPw := strings.Repeat("a", 33)
	_, err = svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, RoomSettings{Name: "X", Password: &longPw})
	assert.Error(t, err)
}

func TestCreateRoom_ClampsSettings(t *testing.T) {
	svc, _ := newTestSvc(t)
	room, err := svc.CreateRoom(context.Background(), "o", domain.RoomTypePrivate, RoomSettings{
		Name: "Clamp Rm", Language: "en",
		MaxPlayers: 0, MaxRounds: 0, RoundTime: 0,
	})
	require.NoError(t, err)
	// Zero size fields fall back to the Drawo defaults, not the absolute minimum.
	assert.Equal(t, domain.RoomDefaultPlayers, room.MaxPlayers)
	assert.Equal(t, domain.RoomMinPlayers, room.MinPlayers)
	assert.Equal(t, domain.RoomDefaultRounds, room.MaxRounds)
	assert.Equal(t, domain.RoomDefaultRoundTime, room.RoundTime)
	assert.Equal(t, "en", room.Language)

	// Explicit values below the minimum floor snap up to the minimum and keep
	// the min≤max invariant.
	roomLow, err := svc.CreateRoom(context.Background(), "o", domain.RoomTypePrivate, RoomSettings{
		Name: "Low", Language: "en",
		MinPlayers: 0, MaxPlayers: 1, // both below floor of 2
		MaxRounds: 0, RoundTime: 10,  // round time below floor
	})
	require.NoError(t, err)
	assert.Equal(t, domain.RoomMinPlayers, roomLow.MinPlayers)
	assert.Equal(t, domain.RoomMinPlayers, roomLow.MaxPlayers)
	assert.Equal(t, domain.RoomDefaultRounds, roomLow.MaxRounds)
	assert.Equal(t, domain.RoomMinRoundTime, roomLow.RoundTime)

	// Invalid language is rejected.
	_, err = svc.CreateRoom(context.Background(), "o", domain.RoomTypePrivate, RoomSettings{Name: "BadLang", Language: "xx"})
	assert.Error(t, err)

	// Upper bounds + snap to step.
	room2, err := svc.CreateRoom(context.Background(), "o", domain.RoomTypePrivate, RoomSettings{
		Name: "Max", MaxPlayers: 99, MaxRounds: 99, RoundTime: 177, // snaps to 170
	})
	require.NoError(t, err)
	assert.Equal(t, domain.RoomMaxPlayers, room2.MaxPlayers)
	assert.Equal(t, domain.RoomMaxRounds, room2.MaxRounds)
	assert.Equal(t, 170, room2.RoundTime)

	// Persian language accepted.
	room3, err := svc.CreateRoom(context.Background(), "o", domain.RoomTypePrivate, RoomSettings{
		Name: "Farsi", Language: "fa",
	})
	require.NoError(t, err)
	assert.Equal(t, "fa", room3.Language)
}

func TestCreateRoom_CustomWordSourceValidates(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()

	sets := validSettings()
	sets.WordSource = domain.WordSourceCustom
	sets.CustomCategories = validCustomCats()
	room, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, sets)
	require.NoError(t, err)
	assert.Len(t, room.CustomCategories, 1)
	assert.Equal(t, domain.WordSourceCustom, room.WordSource)

	// Not enough words.
	bad := validSettings()
	bad.WordSource = domain.WordSourceCustom
	bad.CustomCategories = []domain.CustomCategory{{
		Name: "Small", Words: map[int][]string{1: {"only", "two", "wrds"}},
	}}
	_, err = svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, bad)
	assert.Error(t, err)

	// Invalid word source.
	invalid := validSettings()
	invalid.WordSource = "bogus"
	_, err = svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, invalid)
	assert.Error(t, err)
}

func TestCreateRoom_UnauthorizedWithoutOwner(t *testing.T) {
	svc, _ := newTestSvc(t)
	_, err := svc.CreateRoom(context.Background(), "", domain.RoomTypePrivate, validSettings())
	assert.Error(t, err)
}

func TestCreateRoom_SaveError(t *testing.T) {
	repo := &fakeRoomRepo{saveFn: func(ctx context.Context, room *domain.Room) error {
		return errors.New("redis down")
	}}
	svc := NewRoomService(repo)
	_, err := svc.CreateRoom(context.Background(), "o", domain.RoomTypePrivate, validSettings())
	assert.Error(t, err)
}

func TestGenerateInviteCode_UsesUnambiguousAlphabet(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		code := generateInviteCode()
		assert.Len(t, code, domain.RoomInviteCodeLen)
		for _, r := range code {
			assert.Contains(t, inviteAlphabet, string(r), "code %q contains forbidden rune %c", code, r)
			assert.NotContains(t, "0O1lI", string(r))
		}
		seen[code] = true
	}
	// 6 chars from 32-char alphabet, 50 rolls → almost certainly unique, but
	// this isn't a strict uniqueness test.
	assert.Greater(t, len(seen), 1)
}

// --- GetRoom / GetRoomByInvite ------------------------------------------------

func TestGetRoom_NotFoundAndInvalid(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()

	_, err := svc.GetRoom(ctx, "")
	assert.Error(t, err)

	_, err = svc.GetRoom(ctx, "missing")
	assert.Error(t, err)
}

func TestGetRoomByInvite(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()
	room, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, validSettings())
	require.NoError(t, err)

	got, err := svc.GetRoomByInvite(ctx, room.InviteCode)
	require.NoError(t, err)
	assert.Equal(t, room.ID, got.ID)

	// case-insensitive + whitespace
	got2, err := svc.GetRoomByInvite(ctx, "  "+strings.ToLower(room.InviteCode)+" ")
	require.NoError(t, err)
	assert.Equal(t, room.ID, got2.ID)

	_, err = svc.GetRoomByInvite(ctx, "")
	assert.Error(t, err)
	_, err = svc.GetRoomByInvite(ctx, "NOPE")
	assert.Error(t, err)
}

func TestGetRoomByInvite_RepoError(t *testing.T) {
	repo := &fakeRoomRepo{getByInviteFn: func(ctx context.Context, s string) (*domain.Room, error) {
		return nil, errors.New("boom")
	}}
	svc := NewRoomService(repo)
	_, err := svc.GetRoomByInvite(context.Background(), "ABCDEF")
	assert.Error(t, err)
}

func TestGetRoom_RepoError(t *testing.T) {
	repo := &fakeRoomRepo{getByIDFn: func(ctx context.Context, s string) (*domain.Room, error) {
		return nil, errors.New("boom")
	}}
	svc := NewRoomService(repo)
	_, err := svc.GetRoom(context.Background(), "id")
	assert.Error(t, err)
}

// --- JoinRoom ----------------------------------------------------------------

func TestJoinRoom_PasswordAndClosedRooms(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()

	pw := "secret"
	room, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, RoomSettings{
		Name: "PW Room", Password: &pw,
	})
	require.NoError(t, err)

	// Wrong password.
	_, err = svc.JoinRoom(ctx, room.InviteCode, "wrong")
	assert.Error(t, err)

	// Correct password.
	joined, err := svc.JoinRoom(ctx, room.InviteCode, "secret")
	require.NoError(t, err)
	assert.Equal(t, room.ID, joined.ID)

	// No-password room allows empty string.
	open, err := svc.CreateRoom(ctx, "o2", domain.RoomTypePrivate, validSettings())
	require.NoError(t, err)
	_, err = svc.JoinRoom(ctx, open.InviteCode, "")
	require.NoError(t, err)

	// Closed room is gone.
	require.NoError(t, svc.CloseRoom(ctx, open.ID, "o2"))
	_, err = svc.JoinRoom(ctx, open.InviteCode, "")
	assert.Error(t, err)
}

// --- UpdateSettings ----------------------------------------------------------

func TestUpdateSettings_FullPatch(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()
	room, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, validSettings())
	require.NoError(t, err)

	newName := "New Name"
	newPw := "newpass"
	updated, err := svc.UpdateSettings(ctx, room.ID, "o", RoomSettings{
		Name:       newName,
		Password:   &newPw,
		Language:   "fa",
		MaxPlayers: 6, MaxRounds: 5, RoundTime: 120,
		WordSource:       domain.WordSourceCustom,
		CustomCategories: validCustomCats(),
	})
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.True(t, updated.HasPassword)
	assert.Equal(t, "fa", updated.Language)
	assert.Equal(t, 6, updated.MaxPlayers)
	assert.Equal(t, 5, updated.MaxRounds)
	assert.Equal(t, 120, updated.RoundTime)
	assert.Equal(t, domain.WordSourceCustom, updated.WordSource)
	assert.Len(t, updated.CustomCategories, 1)
}

func TestUpdateSettings_ClearsPasswordAndCustomWords(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()
	pw := "secret"
	s := validSettings()
	s.Password = &pw
	s.WordSource = domain.WordSourceCustom
	s.CustomCategories = validCustomCats()
	room, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, s)
	require.NoError(t, err)
	assert.True(t, room.HasPassword)
	assert.Len(t, room.CustomCategories, 1)

	empty := ""
	// Clear password + switch back to default words.
	updated, err := svc.UpdateSettings(ctx, room.ID, "o", RoomSettings{
		Password:   &empty,
		WordSource: domain.WordSourceDefault,
	})
	require.NoError(t, err)
	assert.False(t, updated.HasPassword)
	assert.Empty(t, updated.PasswordHash)
	assert.Nil(t, updated.CustomCategories)
	assert.Equal(t, domain.WordSourceDefault, updated.WordSource)
}

func TestUpdateSettings_Errors(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()
	room, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, validSettings())
	require.NoError(t, err)

	// Not owner.
	_, err = svc.UpdateSettings(ctx, room.ID, "someone-else", RoomSettings{Name: "x"})
	assert.Error(t, err)

	// Bad name length.
	_, err = svc.UpdateSettings(ctx, room.ID, "o", RoomSettings{Name: "x"})
	assert.Error(t, err)

	// Invalid language.
	_, err = svc.UpdateSettings(ctx, room.ID, "o", RoomSettings{Language: "xx"})
	assert.Error(t, err)

	// Invalid word source.
	_, err = svc.UpdateSettings(ctx, room.ID, "o", RoomSettings{WordSource: "nope"})
	assert.Error(t, err)

	// Short password.
	short := "12"
	_, err = svc.UpdateSettings(ctx, room.ID, "o", RoomSettings{Password: &short})
	assert.Error(t, err)

	// Nonexistent room.
	_, err = svc.UpdateSettings(ctx, "missing", "o", RoomSettings{Name: "ok"})
	assert.Error(t, err)

	// Settings locked after game start.
	require.NoError(t, svc.StartGame(ctx, room.ID, "o", domain.RoomMinPlayers))
	_, err = svc.UpdateSettings(ctx, room.ID, "o", RoomSettings{Name: "Nope"})
	assert.Error(t, err)
}

func TestUpdateSettings_SaveError(t *testing.T) {
	room := &domain.Room{ID: "id", OwnerID: "o", State: domain.RoomStateLobby, Name: "n"}
	repo := &fakeRoomRepo{
		getByIDFn: func(ctx context.Context, s string) (*domain.Room, error) { return room, nil },
		saveFn:    func(ctx context.Context, r *domain.Room) error { return errors.New("save fail") },
	}
	svc := NewRoomService(repo)
	_, err := svc.UpdateSettings(context.Background(), "id", "o", RoomSettings{Name: "New"})
	assert.Error(t, err)
}

func TestUpdateSettings_CustomWithoutCategoriesRejected(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()
	room, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, validSettings())
	require.NoError(t, err)
	// Switching to custom without providing any categories.
	_, err = svc.UpdateSettings(ctx, room.ID, "o", RoomSettings{WordSource: domain.WordSourceCustom})
	assert.Error(t, err)
}

// --- Close / Start / Leave ---------------------------------------------------

func TestCloseRoom(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()
	room, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, validSettings())
	require.NoError(t, err)

	assert.Error(t, svc.CloseRoom(ctx, room.ID, "not-owner"))
	assert.Error(t, svc.CloseRoom(ctx, "missing", "o"))

	require.NoError(t, svc.CloseRoom(ctx, room.ID, "o"))
	got, err := svc.GetRoom(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.RoomStateClosed, got.State)
}

func TestCloseRoom_SaveError(t *testing.T) {
	room := &domain.Room{ID: "id", OwnerID: "o", State: domain.RoomStateLobby}
	repo := &fakeRoomRepo{
		getByIDFn: func(ctx context.Context, s string) (*domain.Room, error) { return room, nil },
		saveFn:    func(ctx context.Context, r *domain.Room) error { return errors.New("save fail") },
	}
	svc := NewRoomService(repo)
	assert.Error(t, svc.CloseRoom(context.Background(), "id", "o"))
}

func TestStartGame(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()
	room, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, validSettings())
	require.NoError(t, err)

	// Not enough players.
	assert.Error(t, svc.StartGame(ctx, room.ID, "o", 1))
	// Not owner.
	assert.Error(t, svc.StartGame(ctx, room.ID, "x", domain.RoomMinPlayers))
	// Missing room.
	assert.Error(t, svc.StartGame(ctx, "missing", "o", domain.RoomMinPlayers))

	require.NoError(t, svc.StartGame(ctx, room.ID, "o", domain.RoomMinPlayers))
	got, err := svc.GetRoom(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.RoomStatePlaying, got.State)
	assert.Equal(t, 1, got.CurrentRound)

	// Already started.
	assert.Error(t, svc.StartGame(ctx, room.ID, "o", domain.RoomMinPlayers))
}

func TestStartGame_CustomReValidates(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()
	s := validSettings()
	s.WordSource = domain.WordSourceCustom
	s.CustomCategories = validCustomCats()
	room, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, s)
	require.NoError(t, err)
	require.NoError(t, svc.StartGame(ctx, room.ID, "o", domain.RoomMinPlayers))
}

func TestStartGame_SaveError(t *testing.T) {
	room := &domain.Room{ID: "id", OwnerID: "o", State: domain.RoomStateLobby, MinPlayers: 2}
	repo := &fakeRoomRepo{
		getByIDFn: func(ctx context.Context, s string) (*domain.Room, error) { return room, nil },
		saveFn:    func(ctx context.Context, r *domain.Room) error { return errors.New("save fail") },
	}
	svc := NewRoomService(repo)
	assert.Error(t, svc.StartGame(context.Background(), "id", "o", 2))
}

func TestLeaveRoom(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()
	room, err := svc.CreateRoom(ctx, "o", domain.RoomTypePrivate, validSettings())
	require.NoError(t, err)

	// Non-owner leaving leaves owner unchanged.
	after, err := svc.LeaveRoom(ctx, room.ID, "player-2", "")
	require.NoError(t, err)
	assert.Equal(t, "o", after.OwnerID)
	assert.Equal(t, domain.RoomStateLobby, after.State)

	// Owner leaves with no new owner → closed.
	after, err = svc.LeaveRoom(ctx, room.ID, "o", "")
	require.NoError(t, err)
	assert.Equal(t, domain.RoomStateClosed, after.State)

	// Owner leaves transferring ownership.
	room2, _ := svc.CreateRoom(ctx, "o2", domain.RoomTypePrivate, validSettings())
	after, err = svc.LeaveRoom(ctx, room2.ID, "o2", "new-owner")
	require.NoError(t, err)
	assert.Equal(t, "new-owner", after.OwnerID)
	assert.Equal(t, domain.RoomStateLobby, after.State)

	// Leaving a closed room is a no-op.
	after, err = svc.LeaveRoom(ctx, room.ID, "o", "")
	require.NoError(t, err)
	assert.Equal(t, domain.RoomStateClosed, after.State)

	// Missing room.
	_, err = svc.LeaveRoom(ctx, "missing", "u", "")
	assert.Error(t, err)
}

func TestLeaveRoom_SaveError(t *testing.T) {
	room := &domain.Room{ID: "id", OwnerID: "o", State: domain.RoomStateLobby}
	repo := &fakeRoomRepo{
		getByIDFn: func(ctx context.Context, s string) (*domain.Room, error) { return room, nil },
		saveFn:    func(ctx context.Context, r *domain.Room) error { return errors.New("save fail") },
	}
	svc := NewRoomService(repo)
	_, err := svc.LeaveRoom(context.Background(), "id", "o", "")
	assert.Error(t, err)
}

// --- ValidateCustomCategories / normalization --------------------------------

func TestValidateCustomCategories(t *testing.T) {
	svc, _ := newTestSvc(t)

	assert.Error(t, svc.ValidateCustomCategories(nil))
	assert.Error(t, svc.ValidateCustomCategories([]domain.CustomCategory{}))
	assert.Error(t, svc.ValidateCustomCategories([]domain.CustomCategory{{Name: "Empty", Words: map[int][]string{}}}))

	// Too few words.
	assert.Error(t, svc.ValidateCustomCategories([]domain.CustomCategory{{
		Name: "Few", Words: map[int][]string{1: {"a", "b"}},
	}}))

	// Enough words — ok.
	assert.NoError(t, svc.ValidateCustomCategories(validCustomCats()))

	// Too many categories.
	many := make([]domain.CustomCategory, 21)
	for i := range many {
		many[i] = domain.CustomCategory{
			Name:  "C" + string(rune('A'+i%26)),
			Words: map[int][]string{1: {"w" + string(rune('a'+i%26)) + "1", "w" + string(rune('a'+i%26)) + "2"}},
		}
	}
	// We need at least MinCustomWords total words — bulk it up.
	many[0].Words[1] = []string{"w1", "w2", "w3", "w4", "w5"}
	assert.Error(t, svc.ValidateCustomCategories(many))
}

func TestNormalizeCustomCategories_DedupesAndCleans(t *testing.T) {
	cats := []domain.CustomCategory{
		{
			Name: "  Animals  ", // whitespace trimmed
			Words: map[int][]string{
				1: {"Cat", " cat ", "CAT", " dog ", "a"}, // "a" too short → dropped; duplicates collapsed
				2: {"  big  cat  "},                        // whitespace collapsed
				9: {"bad tier"},                            // invalid tier dropped
			},
		},
		{Name: "", Words: map[int][]string{1: {"x"}}}, // empty name dropped
		{Name: "animals", Words: map[int][]string{1: {"zebra"}}}, // duplicate name → dropped
		{
			Name: "Places",
			Words: map[int][]string{3: {strings.Repeat("a", 31)}}, // too long → dropped
		},
	}
	got := normalizeCustomCategories(cats)
	require.Len(t, got, 1)
	assert.Equal(t, "Animals", got[0].Name)
	assert.Equal(t, []string{"cat", "dog"}, got[0].Words[1])
	assert.Equal(t, []string{"big cat"}, got[0].Words[2])
	assert.NotContains(t, got[0].Words, 9)
}

func TestTotalCustomWords(t *testing.T) {
	assert.Equal(t, 0, totalCustomWords(nil))
	assert.Equal(t, 3, totalCustomWords([]domain.CustomCategory{
		{Words: map[int][]string{1: {"a", "b"}, 2: {"c"}}},
	}))
}

// --- defaultSettings coverage ------------------------------------------------

func TestDefaultSettings(t *testing.T) {
	// Empty settings with ownerLocale "fa" — zero size fields fall back to the
	// Drawo defaults; MinPlayers is clamped to the floor, MaxPlayers to the
	// default capacity, and both respect the min≤max invariant.
	got := defaultSettings(RoomSettings{}, "fa")
	assert.Equal(t, "Drawo Room", got.Name)
	assert.Equal(t, "fa", got.Language)
	assert.Equal(t, domain.RoomMinPlayers, got.MinPlayers)
	assert.Equal(t, domain.RoomDefaultPlayers, got.MaxPlayers)
	assert.Equal(t, domain.RoomDefaultRounds, got.MaxRounds)
	assert.Equal(t, domain.RoomDefaultRoundTime, got.RoundTime)
	assert.Equal(t, domain.WordSourceDefault, got.WordSource)

	// When MaxPlayers is below MinPlayers (or the floor), it snaps up so the
	// configuration is never unplayable. Explicitly under-min rounds/roundTime
	// also get floored.
	gotLow := defaultSettings(RoomSettings{MaxPlayers: 1, MaxRounds: 0, RoundTime: 10}, "en")
	assert.Equal(t, domain.RoomMinPlayers, gotLow.MinPlayers)
	assert.Equal(t, domain.RoomMinPlayers, gotLow.MaxPlayers)
	assert.Equal(t, domain.RoomDefaultRounds, gotLow.MaxRounds)
	assert.Equal(t, domain.RoomMinRoundTime, gotLow.RoundTime)

	// Unknown language falls back. Round time just under a step snaps to the
	// nearest lower step after being floored to the minimum (30 → 30 since
	// 30 is a multiple of the 10s step).
	got2 := defaultSettings(RoomSettings{Language: "xx", RoundTime: 33}, "")
	assert.Equal(t, "en", got2.Language)
	assert.Equal(t, 30, got2.RoundTime)
}

// --- Guest token tests -------------------------------------------------------

func TestSanitizeNickname(t *testing.T) {
	assert.Equal(t, "", sanitizeNickname("  \t\n"))
	assert.Equal(t, "Ali Reza", sanitizeNickname("  Ali   Reza  "))
	// Control characters (like NUL) are stripped entirely; surrounding letters join.
	assert.Equal(t, "HelloWorld", sanitizeNickname("Hello\u0000World"))
	assert.Equal(t, "a b", sanitizeNickname("a   b")) // collapsed whitespace
}

func TestGenerateGuestToken(t *testing.T) {
	tok := generateGuestToken()
	assert.Len(t, tok, guestTokenLen)
	for _, r := range tok {
		assert.Contains(t, guestTokenAlphabet, string(r))
	}
	assert.NotEqual(t, tok, generateGuestToken())
}

func TestIssueGuestToken_InvalidRoomID(t *testing.T) {
	svc, _ := newTestSvc(t)
	// Direct repo miss via the service returns a friendly error.
	_, err := svc.IssueGuestToken(context.Background(), "does-not-exist", "Alice")
	assert.Error(t, err)
}

func TestIssueGuestToken(t *testing.T) {
	svc, repo := newTestSvc(t)
	ctx := context.Background()
	room, err := svc.CreateRoom(ctx, "owner-1", domain.RoomTypePrivate, RoomSettings{
		Name: "Guest Room", Language: "en", MaxPlayers: 4, MaxRounds: 2, RoundTime: 40,
		WordSource: domain.WordSourceDefault,
	})
	require.NoError(t, err)

	// Missing room id.
	_, err = svc.IssueGuestToken(ctx, "", "Alice")
	assert.Error(t, err)

	// Nickname too short/long.
	_, err = svc.IssueGuestToken(ctx, room.ID, " ")
	assert.Error(t, err)
	_, err = svc.IssueGuestToken(ctx, room.ID, strings.Repeat("x", 21))
	assert.Error(t, err)

	// Happy path — token is saved, guest id is namespaced, room bound, nickname sanitized.
	g, err := svc.IssueGuestToken(ctx, room.ID, "  Alice  B ")
	require.NoError(t, err)
	assert.True(t, domain.IsGuestID(g.GuestID))
	assert.Equal(t, room.ID, g.RoomID)
	assert.Equal(t, "Alice B", g.Nickname)
	assert.Len(t, g.Token, guestTokenLen)
	assert.True(t, g.ExpiresAt.After(g.CreatedAt))

	stored, err := repo.GetGuest(ctx, g.Token)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, g.GuestID, stored.GuestID)
	assert.Equal(t, "Alice B", stored.Nickname)

	// Closed room rejects new guest tokens.
	err = svc.CloseRoom(ctx, room.ID, "owner-1")
	require.NoError(t, err)
	closed, err := svc.GetRoom(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.RoomStateClosed, closed.State)
	_, err = svc.IssueGuestToken(ctx, room.ID, "Bob")
	assert.Error(t, err)

	// Nonexistent room errors.
	_, err = svc.IssueGuestToken(ctx, "missing", "Bob")
	assert.Error(t, err)
}

func TestValidateGuestToken(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()
	room, err := svc.CreateRoom(ctx, "owner-1", domain.RoomTypePrivate, RoomSettings{
		Name: "Guest Room", Language: "en", MaxPlayers: 4, MaxRounds: 2, RoundTime: 40,
		WordSource: domain.WordSourceDefault,
	})
	require.NoError(t, err)
	g, err := svc.IssueGuestToken(ctx, room.ID, "Alice")
	require.NoError(t, err)

	// Empty token rejected.
	_, err = svc.ValidateGuestToken(ctx, "  ")
	assert.Error(t, err)

	// Bogus token rejected.
	_, err = svc.ValidateGuestToken(ctx, "does-not-exist")
	assert.Error(t, err)

	// Valid token resolves.
	got, err := svc.ValidateGuestToken(ctx, g.Token)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, g.GuestID, got.GuestID)
	assert.Equal(t, g.RoomID, got.RoomID)
	assert.Equal(t, "Alice", got.Nickname)

	// Closing the room invalidates and deletes the token.
	err = svc.CloseRoom(ctx, room.ID, "owner-1")
	require.NoError(t, err)
	_, err = svc.ValidateGuestToken(ctx, g.Token)
	assert.Error(t, err)
}
