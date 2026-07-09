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
)

func setupAuthDeps(t *testing.T) (config.Config, repositories.UserRepository, repositories.ProfileRepository, repositories.SessionRepository, RateLimiter) {
	cfg := config.Get()
	cfg.App.SecretKey = "test-secret-key-that-is-long-enough"
	cfg.Auth.AccessTokenExpiry = 1 * time.Hour
	cfg.Auth.RefreshTokenExpiry = 24 * time.Hour
    cfg.Auth.MaxLoginAttempts = 10 

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	db.AutoMigrate(&domain.User{}, &domain.Profile{})

	userRepo := repositories.NewUserRepo(db)
	profileRepo := repositories.NewProfileRepo(db)

	cacheClient, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)

	sessionRepo := repositories.NewSessionRepo(cacheClient)
	limiter := NewRateLimiter(cacheClient)

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
	tokens3, _ := svc.Login(ctx, username, password, "2.2.2.2", "Dev2")

	// 6. LOGOUT
	err = svc.Logout(ctx, tokens3.AccessToken)
	assert.NoError(t, err)
}

func TestAuthService_Errors(t *testing.T) {
	cfg, userRepo, profileRepo, sessionRepo, limiter := setupAuthDeps(t)
	svc := NewAuthService(cfg, userRepo, profileRepo, sessionRepo, limiter)
	ctx := context.Background()

	// Register Conflict
	svc.Register(ctx, "dup", "pass")
	_, err := svc.Register(ctx, "dup", "pass")
	assert.Error(t, err)

	// Login Fail
	_, err = svc.Login(ctx, "ghost", "pass", "", "")
	assert.Error(t, err)

	// Refresh Fail (Invalid Token)
	_, err = svc.Refresh(ctx, "invalid")
	assert.Error(t, err)

	// Logout Invalid
	err = svc.Logout(ctx, "invalid")
	assert.NoError(t, err)
}

func TestAuthService_RateLimit(t *testing.T) {
	cfg, userRepo, profileRepo, sessionRepo, limiter := setupAuthDeps(t)
    cfg.Auth.MaxLoginAttempts = 1
	svc := NewAuthService(cfg, userRepo, profileRepo, sessionRepo, limiter)
	ctx := context.Background()

    svc.Login(ctx, "rate", "pass", "", "")
    _, err := svc.Login(ctx, "rate", "pass", "", "")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "too many failed attempts")
}
