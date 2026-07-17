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
		&domain.Category{},
		&domain.Word{},
		&domain.BadWord{},
		&domain.Song{},
		&domain.GlobalSetting{},
	)
	require.NoError(t, err)
	return db
}

func TestUserRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepo(db)
	user := &domain.User{ID: "u1", Username: "alice"}
	require.NoError(t, repo.Insert(user))
	exists, _ := repo.Exists("alice")
	assert.True(t, exists)
	fetched, _ := repo.GetByID("u1")
	assert.Equal(t, "alice", fetched.Username)
	fetched, _ = repo.GetByUsername("alice")
	assert.Equal(t, "u1", fetched.ID)
	user.Username = "alice2"
	assert.NoError(t, repo.Update(user))
	db.Create(&domain.Profile{UserID: "u1", Email: "alice@test.com"})
	results, _ := repo.SearchUsers("alice")
	assert.Len(t, results, 1)
}

func TestProfileRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProfileRepo(db)
	p := &domain.Profile{UserID: "u1", Email: "a@a.com"}
	repo.Insert(p)
	f, _ := repo.GetByUserID("u1")
	assert.Equal(t, "a@a.com", f.Email)
	p.Theme = "dark"
	repo.Update(p)
}

func TestContentRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContentRepo(db)
	ctx := context.Background()
	cat := &domain.Category{ID: "c1", GroupID: "cg1", Name: "N", Language: "en"}
	repo.InsertCategory(ctx, cat)
	list, _ := repo.ListCategories(ctx, "en")
	assert.Len(t, list, 1)
	word := &domain.Word{ID: "w1", GroupID: "wg1", CategoryID: "c1", Text: "T", Language: "en"}
	repo.InsertWord(ctx, word)
	words, _ := repo.GetRandomWordGroups(ctx, "c1", "en", 1)
	assert.Len(t, words, 1)
	trans, _ := repo.GetTranslation(ctx, "wg1", "en")
	assert.Equal(t, "T", trans.Text)
	repo.InsertBadWord(ctx, &domain.BadWord{Text: "B", Language: "en"})
	bws, _ := repo.ListBadWords(ctx, "en")
	assert.Len(t, bws, 1)
}

func TestAdminRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAdminRepo(db)
	ctx := context.Background()
	song := &domain.Song{ID: "s1", Title: "T", Type: domain.SongTypeLanding}
	repo.SaveSong(ctx, song)
	s, _ := repo.GetSongByID(ctx, "s1")
	assert.Equal(t, "T", s.Title)
	repo.ListSongs(ctx, domain.SongTypeLanding)
	repo.UpdateSetting(ctx, "k", "v")
	val, _ := repo.GetSetting(ctx, "k")
	assert.Equal(t, "v", val)
	repo.UpdateSetting(ctx, "k", "v2")
	repo.DeleteSong(ctx, "s1")
}

func TestEphemeralRepositories(t *testing.T) {
	mc := NewMockCache(t)
	ctx := context.Background()

	// Session
	repoS := NewSessionRepo(mc)
	sess := &domain.Session{ID: "s1", UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)}
	repoS.Set(ctx, sess)
	repoS.Get(ctx, "s1")
	repoS.Delete(ctx, "s1")
	repoS.DeleteAllForUser(ctx, "u1")

	// Room
	repoR := NewRoomRepo(mc)
	room := &domain.Room{ID: "r1", Name: "R"}
	repoR.Save(ctx, room)
	repoR.GetByID(ctx, "r1")
	repoR.GetByInviteCode(ctx, "I")
	repoR.Delete(ctx, "r1", "I")
	repoR.ListPublic(ctx, "en", domain.Paging{})

	// OTP
	repoO := NewOTPRepo(mc)
	otp := &domain.OTP{Identifier: "i", Type: domain.OTPEmail, Code: "1", ExpiresAt: time.Now().Add(time.Hour)}
	repoO.Set(ctx, otp)
	repoO.Get(ctx, "i", domain.OTPEmail)
	repoO.Delete(ctx, "i", domain.OTPEmail)
}

func TestMiscRepositories(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	NewAchievementRepo(db).UnlockAchievement(ctx, &domain.Achievement{})
	NewAchievementRepo(db).ListUserAchievements(ctx, "1")
	NewGameHistoryRepo(db).SaveGameSummary(ctx, &domain.GameHistory{}, nil, nil)
	NewGameHistoryRepo(db).GetGameSummary(ctx, "1")
	NewGameHistoryRepo(db).ListUserGames(ctx, "1", domain.Paging{})
	NewPlayerStatisticRepo(db).UpsertStats(ctx, &domain.PlayerStatistic{})
	NewPlayerStatisticRepo(db).GetStats(ctx, "1")
	NewReportRepo(db).InsertReport(ctx, &domain.Report{})
	NewReportRepo(db).ListReports(ctx, domain.Paging{})
	NewUserSettingsRepo(db).SaveSettings(ctx, &domain.UserSettings{})
	NewUserSettingsRepo(db).GetSettings(ctx, "1")
}
