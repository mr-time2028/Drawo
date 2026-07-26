// Package services defines application use cases and business logic.
package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/pkg/errors"
)

// UserService defines the contract for managing user data and contact verification.
type UserService interface {
	// GetProfile retrieves the full profile and account details for a user.
	GetProfile(ctx context.Context, userID string) (*domain.UserWithProfile, error)
	// UpdateProfile updates non-critical fields (avatar, sounds, locale).
	UpdateProfile(ctx context.Context, userID string, updates domain.Profile) (*domain.Profile, error)
	// ChangeUsername allows a user to rename their account if the new name is unique.
	ChangeUsername(ctx context.Context, userID, newUsername string) error
	// RequestVerification generates and sends a 6-digit code to Email or Phone.
	RequestVerification(ctx context.Context, userID string, otpType domain.OTPType) error
	// ConfirmVerification checks a code and marks the contact as verified in DB.
	ConfirmVerification(ctx context.Context, userID, code string, otpType domain.OTPType) error
}

type userService struct {
	userRepo    repositories.UserRepository
	profileRepo repositories.ProfileRepository
	otpRepo     repositories.OTPRepository
	otpSvc      OTPService
}

// NewUserService creates a new user management service.
func NewUserService(
	u repositories.UserRepository,
	p repositories.ProfileRepository,
	o repositories.OTPRepository,
	os OTPService,
) UserService {
	return &userService{
		userRepo:    u,
		profileRepo: p,
		otpRepo:     o,
		otpSvc:      os,
	}
}

func (s *userService) GetProfile(ctx context.Context, userID string) (*domain.UserWithProfile, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return nil, errors.New(errors.ErrNotFound, "user not found")
	}

	profile, err := s.profileRepo.GetByUserID(userID)
	if err != nil || profile == nil {
		return nil, errors.New(errors.ErrNotFound, "profile not found")
	}

	return &domain.UserWithProfile{
		User:    *user,
		Profile: *profile,
	}, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID string, updates domain.Profile) (*domain.Profile, error) {
	profile, err := s.profileRepo.GetByUserID(userID)
	if err != nil || profile == nil {
		return nil, errors.New(errors.ErrNotFound, "profile not found")
	}

	// Update only allowed fields
	profile.AvatarURL = updates.AvatarURL
	profile.Locale = updates.Locale
	profile.Theme = updates.Theme
	profile.BackgroundSound = updates.BackgroundSound
	profile.ToolSound = updates.ToolSound
	profile.UpdatedAt = time.Now()

	if err := s.profileRepo.Update(profile); err != nil {
		return nil, errors.New(errors.ErrInternalServer, "failed to update profile")
	}

	return profile, nil
}

func (s *userService) ChangeUsername(ctx context.Context, userID, newUsername string) error {
	// 1. Check uniqueness
	exists, err := s.userRepo.Exists(newUsername)
	if err != nil {
		return errors.New(errors.ErrInternalServer, "database error")
	}
	if exists {
		return errors.New(errors.ErrConflict, "username already taken")
	}

	// 2. Fetch user
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return errors.New(errors.ErrNotFound, "user not found")
	}

	// 3. Update
	user.Username = newUsername
	user.UpdatedAt = time.Now()
	return s.userRepo.Update(user)
}

func (s *userService) RequestVerification(ctx context.Context, userID string, otpType domain.OTPType) error {
	profile, err := s.profileRepo.GetByUserID(userID)
	if err != nil || profile == nil {
		return errors.New(errors.ErrNotFound, "profile not found")
	}

	identifier := ""
	if otpType == domain.OTPEmail {
		identifier = profile.Email
	} else {
		identifier = profile.Phone
	}

	if identifier == "" {
		return errors.New(errors.ErrBadRequest, "contact information missing")
	}

	// Generate 6-digit code
	code := generate6DigitCode()
	otp := &domain.OTP{
		Identifier: identifier,
		Type:       otpType,
		Code:       code,
		ExpiresAt:  time.Now().Add(10 * time.Minute),
	}

	// Save to Redis
	if err := s.otpRepo.Set(ctx, otp); err != nil {
		return errors.New(errors.ErrInternalServer, "failed to store verification code")
	}

	// Send via provider
	return s.otpSvc.SendCode(ctx, otp)
}

func (s *userService) ConfirmVerification(ctx context.Context, userID, code string, otpType domain.OTPType) error {
	profile, err := s.profileRepo.GetByUserID(userID)
	if err != nil || profile == nil {
		return errors.New(errors.ErrNotFound, "profile not found")
	}

	identifier := ""
	if otpType == domain.OTPEmail {
		identifier = profile.Email
	} else {
		identifier = profile.Phone
	}

	// Retrieve from Redis
	storedCode, err := s.otpRepo.Get(ctx, identifier, otpType)
	if err != nil || storedCode != code {
		return errors.New(errors.ErrUnauthorized, "invalid or expired verification code")
	}

	// Mark as verified in DB
	if otpType == domain.OTPEmail {
		profile.EmailVerified = true
	} else {
		profile.PhoneVerified = true
	}
	profile.UpdatedAt = time.Now()

	if err := s.profileRepo.Update(profile); err != nil {
		return errors.New(errors.ErrInternalServer, "failed to update verification status")
	}

	// Cleanup
	_ = s.otpRepo.Delete(ctx, identifier, otpType)

	return nil
}

func generate6DigitCode() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	// Convert bytes to a simple 6-digit numeric string for easy user entry
	return fmt.Sprintf("%06d", int(b[0])%10*100000+int(b[1])%100*1000+int(b[2])%1000)
}
