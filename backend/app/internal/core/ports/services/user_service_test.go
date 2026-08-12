package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/internal/infrastructure/cache"
)

func setupUserDeps(t *testing.T) (repositories.UserRepository, repositories.ProfileRepository, repositories.OTPRepository, OTPService) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	db.AutoMigrate(&domain.User{}, &domain.Profile{})

	userRepo := repositories.NewUserRepo(db)
	profileRepo := repositories.NewProfileRepo(db)

	cacheClient, _ := cache.NewClient(config.CacheConfig{Driver: "memory"})
	otpRepo := repositories.NewOTPRepo(cacheClient)
	otpSvc := NewMockOTPService()

	return userRepo, profileRepo, otpRepo, otpSvc
}

func setupBrokenUserRepo(t *testing.T) (repositories.UserRepository, repositories.ProfileRepository, repositories.OTPRepository) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	// Don't migrate -> will cause errors
	userRepo := repositories.NewUserRepo(db)
	profileRepo := repositories.NewProfileRepo(db)
	cacheClient, _ := cache.NewClient(config.CacheConfig{Driver: "memory"})
	otpRepo := repositories.NewOTPRepo(cacheClient)
	return userRepo, profileRepo, otpRepo
}

func TestUserService_Profile(t *testing.T) {
	uRepo, pRepo, oRepo, oSvc := setupUserDeps(t)
	svc := NewUserService(uRepo, pRepo, oRepo, oSvc)
	ctx := context.Background()

	userID := "u1"
	uRepo.Insert(&domain.User{ID: userID, Username: "alice"})
	pRepo.Insert(&domain.Profile{UserID: userID, Email: "alice@test.com"})

	// 1. Get Profile
	up, err := svc.GetProfile(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "alice", up.User.Username)

	// 2. Update Profile
	updates := domain.Profile{AvatarURL: "new_url", Locale: "en"}
	p, err := svc.UpdateProfile(ctx, userID, updates)
	require.NoError(t, err)
	assert.Equal(t, "new_url", p.AvatarURL)

	// 3. Change Username
	err = svc.ChangeUsername(ctx, userID, "new_alice")
	require.NoError(t, err)

	// 4. Change Username Fail (Duplicate)
	uRepo.Insert(&domain.User{ID: "u2", Username: "taken"})
	err = svc.ChangeUsername(ctx, userID, "taken")
	assert.Error(t, err)
}

func TestUserService_Verification(t *testing.T) {
	uRepo, pRepo, oRepo, oSvc := setupUserDeps(t)
	svc := NewUserService(uRepo, pRepo, oRepo, oSvc)
	ctx := context.Background()

	userID := "u1"
	pRepo.Insert(&domain.Profile{UserID: userID, Email: "test@test.com", Phone: "12345"})

	// 1. Request Email
	err := svc.RequestVerification(ctx, userID, domain.OTPEmail)
	assert.NoError(t, err)
	code, _ := oRepo.Get(ctx, "test@test.com", domain.OTPEmail)

	// 2. Confirm Email
	err = svc.ConfirmVerification(ctx, userID, code, domain.OTPEmail)
	assert.NoError(t, err)

	// 3. Request Phone
	err = svc.RequestVerification(ctx, userID, domain.OTPPhone)
	assert.NoError(t, err)
	code2, _ := oRepo.Get(ctx, "12345", domain.OTPPhone)
	err = svc.ConfirmVerification(ctx, userID, code2, domain.OTPPhone)
	assert.NoError(t, err)
}

func TestUserService_Errors(t *testing.T) {
	uRepo, pRepo, oRepo, oSvc := setupUserDeps(t)
	svc := NewUserService(uRepo, pRepo, oRepo, oSvc)
	ctx := context.Background()

	// NotFound cases
	_, err := svc.GetProfile(ctx, "ghost")
	assert.Error(t, err)
	_, err = svc.UpdateProfile(ctx, "ghost", domain.Profile{})
	assert.Error(t, err)
	err = svc.ChangeUsername(ctx, "ghost", "new")
	assert.Error(t, err)
	err = svc.RequestVerification(ctx, "ghost", domain.OTPEmail)
	assert.Error(t, err)
	err = svc.ConfirmVerification(ctx, "ghost", "1", domain.OTPEmail)
	assert.Error(t, err)

	// Missing contact
	pRepo.Insert(&domain.Profile{UserID: "no_email"})
	err = svc.RequestVerification(ctx, "no_email", domain.OTPEmail)
	assert.Error(t, err)
	err = svc.RequestVerification(ctx, "no_email", domain.OTPPhone)
	assert.Error(t, err)

	// Database failures
	bu, bp, bo := setupBrokenUserRepo(t)
	bs := NewUserService(bu, bp, bo, oSvc)
	err = bs.ChangeUsername(ctx, "1", "fail")
	assert.Error(t, err)
}
