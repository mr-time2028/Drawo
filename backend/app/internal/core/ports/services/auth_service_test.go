package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/internal/infrastructure/cache"
	"drawo/pkg/i18n"
)

func setupAuthDeps(t *testing.T) (config.Config, repositories.UserRepository, repositories.ProfileRepository, repositories.SessionRepository, RateLimiter) {
	cfg := config.Get()
	cfg.App.SecretKey = "test-secret-key-that-is-long-enough"
	cfg.Auth.AccessTokenExpiry = 1 * time.Hour
	cfg.Auth.RefreshTokenExpiry = 24 * time.Hour
	cfg.Auth.MaxLoginAttempts = 100 // High enough to avoid failures in loop tests

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&domain.User{}, &domain.Profile{})
	require.NoError(t, err)

	userRepo := repositories.NewUserRepo(db)
	profileRepo := repositories.NewProfileRepo(db)

	cacheClient, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)

	sessionRepo := repositories.NewSessionRepo(cacheClient)
	limiter := NewRateLimiter(cacheClient)

	// Init i18n for ban messages
	_ = i18n.Init("../../../../locales", "fa")

	return cfg, userRepo, profileRepo, sessionRepo, limiter
}

func TestAuthService_FullLifecycle(t *testing.T) {
	cfg, userRepo, profileRepo, sessionRepo, limiter := setupAuthDeps(t)
	svc := NewAuthService(cfg, userRepo, profileRepo, sessionRepo, limiter)
	ctx := context.Background()

	username := "hamid"
	password := "Secret123!"

	// 1. REGISTER
	user, err := svc.Register(ctx, username, password)
	require.NoError(t, err)
	assert.Equal(t, username, user.Username)

	// 2. LOGIN
	tokens, err := svc.Login(ctx, username, password, "1.2.3.4", "Mozilla/Test")
	require.NoError(t, err)
	require.NotNil(t, tokens)

	// 3. REFRESH
	newTokens, err := svc.Refresh(ctx, tokens.RefreshToken)
	require.NoError(t, err)
	assert.NotNil(t, newTokens)

	// 4. REUSE DETECTION
	_, err = svc.Refresh(ctx, tokens.RefreshToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reuse detected")

	// 5. SINGLE DEVICE POLICY
	_, _ = svc.Login(ctx, username, password, "1.1.1.1", "Dev1")
	tokens3, err := svc.Login(ctx, username, password, "2.2.2.2", "Dev2")
	require.NoError(t, err)

	// 6. LOGOUT
	err = svc.Logout(ctx, tokens3.AccessToken)
	assert.NoError(t, err)
}

func TestAuthService_Failures(t *testing.T) {
	cfg, userRepo, profileRepo, sessionRepo, limiter := setupAuthDeps(t)
	svc := NewAuthService(cfg, userRepo, profileRepo, sessionRepo, limiter)
	ctx := context.Background()

	// Register Conflict
	svc.Register(ctx, "dup", "pass")
	_, err := svc.Register(ctx, "dup", "pass")
	assert.Error(t, err)

	// Login Fail (Ghost User)
	_, err = svc.Login(ctx, "ghost", "pass", "", "")
	assert.Error(t, err)

	// Login Fail (Wrong Password)
	svc.Register(ctx, "wrong", "pass")
	_, err = svc.Login(ctx, "wrong", "fail", "", "")
	assert.Error(t, err)

	// Refresh Fail (Invalid Token)
	_, err = svc.Refresh(ctx, "invalid")
	assert.Error(t, err)

	// Refresh Fail (Revoked Session)
	user, _ := svc.Register(ctx, "hamid2", "pass")
	tks, err := svc.Login(ctx, "hamid2", "pass", "", "")
	require.NoError(t, err)
	sessionRepo.DeleteAllForUser(ctx, user.ID)
	_, err = svc.Refresh(ctx, tks.RefreshToken)
	assert.Error(t, err)
}

func TestAuthService_AccountStatusMessages(t *testing.T) {
	cfg, userRepo, profileRepo, sessionRepo, limiter := setupAuthDeps(t)
	svc := NewAuthService(cfg, userRepo, profileRepo, sessionRepo, limiter)
	ctx := context.Background()

	t.Run("BannedPersian", func(t *testing.T) {
		username := "banned_user"
		user, err := svc.Register(ctx, username, "pass")
		require.NoError(t, err)
		user.IsActive = false
		user.Status = domain.AccountStatusBanned
		require.NoError(t, userRepo.Update(user))

		_, err = svc.Login(ctx, username, "pass", "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "مسدود")
	})

	t.Run("SuspendedEnglish", func(t *testing.T) {
		username := "suspended_user"
		user, err := svc.Register(ctx, username, "pass")
		require.NoError(t, err)
		profile, err := profileRepo.GetByUserID(user.ID)
		require.NoError(t, err)
		profile.Locale = "en"
		require.NoError(t, profileRepo.Update(profile))
		user.Status = domain.AccountStatusSuspended
		require.NoError(t, userRepo.Update(user))

		_, err = svc.Login(ctx, username, "pass", "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "suspended")
	})

	t.Run("DeletedEnglish", func(t *testing.T) {
		username := "deleted_user"
		user, err := svc.Register(ctx, username, "pass")
		require.NoError(t, err)
		profile, err := profileRepo.GetByUserID(user.ID)
		require.NoError(t, err)
		profile.Locale = "en"
		require.NoError(t, profileRepo.Update(profile))
		user.Status = domain.AccountStatusDeleted
		require.NoError(t, userRepo.Update(user))

		_, err = svc.Login(ctx, username, "pass", "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deleted")
	})
}

func TestAuthService_RegisterRollsBackUserWhenProfileCreateFails(t *testing.T) {
	cfg := config.Get()
	cfg.App.SecretKey = "test-secret-key-that-is-long-enough"
	cfg.Auth.AccessTokenExpiry = time.Hour
	cfg.Auth.RefreshTokenExpiry = 24 * time.Hour
	cfg.Auth.MaxLoginAttempts = 100

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// Intentionally migrate only users. Profile creation will fail inside the
	// transaction, and the inserted user must be rolled back.
	require.NoError(t, db.AutoMigrate(&domain.User{}))

	userRepo := repositories.NewUserRepo(db)
	profileRepo := repositories.NewProfileRepo(db)
	cacheClient, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	defer cacheClient.Close()

	svc := NewAuthService(cfg, userRepo, profileRepo, repositories.NewSessionRepo(cacheClient), NewRateLimiter(cacheClient))
	_, err = svc.Register(context.Background(), "rollback_user", "Password123")
	require.Error(t, err)

	user, err := userRepo.GetByUsername("rollback_user")
	require.NoError(t, err)
	assert.Nil(t, user, "failed profile creation must not leave an orphaned user")
}
