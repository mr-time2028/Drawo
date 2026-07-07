package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"drawo/internal/core/domain"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate all tables
	err = db.AutoMigrate(
		&domain.User{},
		&domain.Profile{},
		&domain.Friendship{},
		&domain.FriendRequest{},
		&domain.GameHistory{},
		&domain.Round{},
		&domain.Score{},
		&domain.Report{},
		&domain.Achievement{},
		&domain.PlayerStatistic{},
		&domain.UserSettings{},
	)
	require.NoError(t, err)

	return db
}

func setupBrokenDB(t *testing.T) *gorm.DB {
    // A DB where tables don't exist
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
    return db
}

func TestUserRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepo(db)

	user := &domain.User{
		ID:       "user-1",
		Username: "alice",
	}

	// Test Insert
	err := repo.Insert(user)
	assert.NoError(t, err)

	// Test Exists
	exists, err := repo.Exists("alice")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Test GetByID
	fetched, err := repo.GetByID("user-1")
	assert.NoError(t, err)
	assert.Equal(t, "alice", fetched.Username)

	// Test GetByUsername
	fetched, err = repo.GetByUsername("alice")
	assert.NoError(t, err)
	assert.Equal(t, "user-1", fetched.ID)

	// Test Update
	user.Username = "alice-updated"
	err = repo.Update(user)
	assert.NoError(t, err)
	
	fetched, _ = repo.GetByID("user-1")
	assert.Equal(t, "alice-updated", fetched.Username)
	
	// Test Not Found
	f, err := repo.GetByID("none")
	assert.NoError(t, err)
	assert.Nil(t, f)
	
	f, err = repo.GetByUsername("none")
	assert.NoError(t, err)
	assert.Nil(t, f)

    // Test Errors (broken DB)
    brokenRepo := NewUserRepo(setupBrokenDB(t))
    _, err = brokenRepo.GetByID("1")
    assert.Error(t, err)
    _, err = brokenRepo.GetByUsername("1")
    assert.Error(t, err)
    _, err = brokenRepo.Exists("1")
    assert.Error(t, err)
}

func TestProfileRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProfileRepo(db)

	profile := &domain.Profile{
		UserID: "user-1",
		Email:  "alice@example.com",
	}

	err := repo.Insert(profile)
	assert.NoError(t, err)

	fetched, err := repo.GetByUserID("user-1")
	assert.NoError(t, err)
	assert.Equal(t, "alice@example.com", fetched.Email)

	profile.Email = "new@example.com"
	err = repo.Update(profile)
	assert.NoError(t, err)
	
	// Test Not Found
	f, err := repo.GetByUserID("none")
	assert.Error(t, err)
	assert.Nil(t, f)

    // Test Errors
    brokenRepo := NewProfileRepo(setupBrokenDB(t))
    assert.Error(t, brokenRepo.Insert(profile))
    assert.Error(t, brokenRepo.Update(profile))
}

func TestFriendshipRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFriendshipRepo(db)
	ctx := context.Background()

	f := &domain.Friendship{UserID: "u1", FriendID: "u2"}
	assert.NoError(t, repo.AddFriend(ctx, f))

	list, err := repo.ListFriends(ctx, "u1")
	assert.NoError(t, err)
	assert.Len(t, list, 1)

	assert.NoError(t, repo.RemoveFriend(ctx, "u1", "u2"))
    
    // Test Errors
    brokenRepo := NewFriendshipRepo(setupBrokenDB(t))
    assert.Error(t, brokenRepo.AddFriend(ctx, f))
    _, err = brokenRepo.ListFriends(ctx, "u1")
    assert.Error(t, err)
}

func TestFriendRequestRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFriendRequestRepo(db)
	ctx := context.Background()

	req := &domain.FriendRequest{ID: "r1", FromID: "u1", ToID: "u2", Status: "pending"}
	assert.NoError(t, repo.CreateRequest(ctx, req))

	fetched, err := repo.GetByID(ctx, "r1")
	assert.NoError(t, err)
	assert.Equal(t, "u1", fetched.FromID)

	list, err := repo.ListPending(ctx, "u2")
	assert.NoError(t, err)
	assert.Len(t, list, 1)

	req.Status = "accepted"
	assert.NoError(t, repo.UpdateRequest(ctx, req))
	
	// Test Not Found
	f, err := repo.GetByID(ctx, "none")
	assert.Error(t, err)
	assert.Nil(t, f)

    // Test Errors
    brokenRepo := NewFriendRequestRepo(setupBrokenDB(t))
    assert.Error(t, brokenRepo.CreateRequest(ctx, req))
    _, err = brokenRepo.ListPending(ctx, "u1")
    assert.Error(t, err)
    assert.Error(t, brokenRepo.UpdateRequest(ctx, req))
}

func TestGameHistoryRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGameHistoryRepo(db)
	ctx := context.Background()

	hist := &domain.GameHistory{ID: "g1", RoomID: "r1", StartedAt: time.Now(), EndedAt: time.Now()}
	rounds := []domain.Round{{ID: "rnd1", GameHistoryID: "g1"}}
	scores := []domain.Score{{ID: "s1", GameHistoryID: "g1", UserID: "u1", Points: 100}}

	assert.NoError(t, repo.SaveGameSummary(ctx, hist, rounds, scores))

	g, r, s, err := repo.GetGameSummary(ctx, "g1")
	assert.NoError(t, err)
	assert.NotNil(t, g)
	assert.Len(t, r, 1)
	assert.Len(t, s, 1)

	list, err := repo.ListUserGames(ctx, "u1", domain.Paging{Limit: 10, Offset: 0})
	assert.NoError(t, err)
	assert.NotNil(t, list)
	
	// Test Not Found
	_, _, _, err = repo.GetGameSummary(ctx, "none")
	assert.Error(t, err)

    // Test Transaction Fail
    brokenRepo := NewGameHistoryRepo(setupBrokenDB(t))
    assert.Error(t, brokenRepo.SaveGameSummary(ctx, hist, rounds, scores))
}

func TestStatsRepository(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	
	repRepo := NewReportRepo(db)
	assert.NoError(t, repRepo.InsertReport(ctx, &domain.Report{ID: "rep1"}))
	reps, err := repRepo.ListReports(ctx, domain.Paging{Limit: 10})
	assert.NoError(t, err)
	assert.Len(t, reps.Items, 1)

	achRepo := NewAchievementRepo(db)
	assert.NoError(t, achRepo.UnlockAchievement(ctx, &domain.Achievement{ID: "a1", UserID: "u1"}))
	achs, err := achRepo.ListUserAchievements(ctx, "u1")
	assert.NoError(t, err)
	assert.Len(t, achs, 1)

	statRepo := NewPlayerStatisticRepo(db)
	assert.NoError(t, statRepo.UpsertStats(ctx, &domain.PlayerStatistic{UserID: "u1", TotalGames: 5}))
	s, err := statRepo.GetStats(ctx, "u1")
	assert.NoError(t, err)
	assert.Equal(t, int64(5), s.TotalGames)

	setRepo := NewUserSettingsRepo(db)
	assert.NoError(t, setRepo.SaveSettings(ctx, &domain.UserSettings{UserID: "u1", Theme: "dark"}))
	sets, err := setRepo.GetSettings(ctx, "u1")
	assert.NoError(t, err)
	assert.Equal(t, "dark", sets.Theme)

    // Test Errors
    brokenDB := setupBrokenDB(t)
    br := NewReportRepo(brokenDB)
    assert.Error(t, br.InsertReport(ctx, &domain.Report{}))
    
    ba := NewAchievementRepo(brokenDB)
    _, err = ba.ListUserAchievements(ctx, "1")
    assert.Error(t, err)
}
