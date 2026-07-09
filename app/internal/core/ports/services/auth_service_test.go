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

type failLimiter struct{}
func (f *failLimiter) Allow(ctx context.Context, k string, l int, w time.Duration) (bool, error) {
    return false, assert.AnError
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

    // Logout already invalid
    err = svc.Logout(ctx, "invalid")
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

    // Failed Limiter
    fs := NewAuthService(cfg, userRepo, profileRepo, sessionRepo, &failLimiter{})
    _, err = fs.Login(ctx, "u", "p", "", "")
    assert.Error(t, err)
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

func TestAuthService_Register_BrokenDB(t *testing.T) {
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    ur := repositories.NewUserRepo(db)
    pr := repositories.NewProfileRepo(db)
    svc := NewAuthService(config.Get(), ur, pr, nil, nil)
    
    _, err := svc.Register(context.Background(), "u", "p")
    assert.Error(t, err)
}
