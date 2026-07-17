package services

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/internal/infrastructure/cache"
)

type mockStorage struct {
	mock.Mock
}
func (m *mockStorage) Upload(ctx context.Context, b, n string, r io.Reader, s int64, c string) (string, error) {
	args := m.Called(ctx, b, n, r, s, c); return args.String(0), args.Error(1)
}
func (m *mockStorage) Delete(ctx context.Context, b, n string) error { return m.Called(ctx, b, n).Error(0) }
func (m *mockStorage) GetURL(ctx context.Context, b, n string) (string, error) {
	args := m.Called(ctx, b, n); return args.String(0), args.Error(1)
}

func setupAdminDeps(t *testing.T) (config.Config, repositories.AdminRepository, repositories.UserRepository, repositories.ProfileRepository, repositories.SessionRepository, *mockStorage) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&domain.Song{}, &domain.GlobalSetting{}, &domain.User{}, &domain.Profile{})
	
    cfg := config.Get()
	cfg.App.Storage.BucketName = "b"

    cacheClient, _ := cache.NewClient(config.CacheConfig{Driver: "memory"})

	return cfg, repositories.NewAdminRepo(db), repositories.NewUserRepo(db), repositories.NewProfileRepo(db), repositories.NewSessionRepo(cacheClient), new(mockStorage)
}

func TestAdminService(t *testing.T) {
	cfg, aRepo, uRepo, pRepo, sRepo, mStore := setupAdminDeps(t)
	svc := NewAdminService(cfg, aRepo, uRepo, pRepo, sRepo, mStore)
	ctx := context.Background()

	// Songs
	mStore.On("Upload", mock.Anything, "b", mock.Anything, mock.Anything, int64(5), "audio/mpeg").Return("k", nil)
	s, _ := svc.UploadSong(ctx, "S", domain.SongTypeLanding, strings.NewReader("data"), 5)
	assert.NotNil(t, s)
	svc.ListSongs(ctx, domain.SongTypeLanding)
	svc.ToggleSong(ctx, s.ID, false)
    mStore.On("Delete", mock.Anything, "b", "k").Return(nil)
	svc.DeleteSong(ctx, s.ID)

	// Moderation
	uRepo.Insert(&domain.User{ID: "u1", Username: "u"})
	svc.SearchUsers(ctx, "u")
	svc.BanUser(ctx, "u1")
	svc.UnbanUser(ctx, "u1")

	// Settings
	svc.UpdateGlobalSetting(ctx, "k", "v")
}
