// Package services implements the application use cases (business logic).
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/pkg/errors"
	"drawo/pkg/i18n"
	"drawo/pkg/security"
)

// AuthService handles authentication logic: Login, Register, Refresh, Logout.
type AuthService interface {
	Register(ctx context.Context, username, password string) (*domain.User, error)
	Login(ctx context.Context, username, password, ip, userAgent string) (*domain.TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*domain.TokenPair, error)
	Logout(ctx context.Context, accessToken string) error
}

// authService is the hardened implementation of AuthService.
type authService struct {
	cfg         config.Config
	userRepo    repositories.UserRepository
	profileRepo repositories.ProfileRepository
	sessionRepo repositories.SessionRepository
	limiter     RateLimiter
	jwt         *security.JWTManager
}

// NewAuthService creates a new hardened authentication service.
func NewAuthService(
	cfg config.Config,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	sessionRepo repositories.SessionRepository,
	limiter RateLimiter,
) AuthService {
	// Initialize the JWT manager with secret and durations from config.
	jwtMgr := security.NewJWTManager(
		cfg.App.SecretKey,
		cfg.Auth.Issuer,
		cfg.Auth.AccessTokenExpiry,
		cfg.Auth.RefreshTokenExpiry,
	)

	return &authService{
		cfg:         cfg,
		userRepo:    userRepo,
		profileRepo: profileRepo,
		sessionRepo: sessionRepo,
		limiter:     limiter,
		jwt:         jwtMgr,
	}
}

// Register creates a new user and an associated empty profile.
func (s *authService) Register(ctx context.Context, username, password string) (*domain.User, error) {
	// 1. Check if user already exists
	exists, err := s.userRepo.Exists(username)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServer, "database lookup failed")
	}
	if exists {
		return nil, errors.New(errors.ErrConflict, "username is already taken")
	}

	// 2. Hash the password securely
	hash, err := security.HashPassword(password)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServer, "failed to process password")
	}

	// 3. Create the user entity
	user := &domain.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: hash,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Insert(user); err != nil {
		return nil, errors.New(errors.ErrInternalServer, "failed to create user account")
	}

	// 4. Create the initial profile for the user
	profile := &domain.Profile{
		UserID:          user.ID,
		ReputationScore: 10000, // Default production starting reputation
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.profileRepo.Insert(profile); err != nil {
		// Note: In production this should use a transaction; for now we use granular repositories.
		return nil, errors.New(errors.ErrInternalServer, "account created but profile failed")
	}

	return user, nil
}

// Login verifies credentials and establishes a session (Single Device Policy enforced).
func (s *authService) Login(ctx context.Context, username, password, ip, userAgent string) (*domain.TokenPair, error) {
	// 1. Brute-force protection (Rate Limit)
	limitKey := fmt.Sprintf("login_attempt:%s", username)
	allowed, err := s.limiter.Allow(ctx, limitKey, s.cfg.Auth.MaxLoginAttempts, s.cfg.Auth.LoginLockoutDuration)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServer, "security check failed")
	}
	if !allowed {
		return nil, errors.New(errors.ErrTooManyRequests, "too many failed attempts, please wait")
	}

	// 2. Fetch user
	user, err := s.userRepo.GetByUsername(username)
	if err != nil || user == nil {
		return nil, errors.New(errors.ErrUnauthorized, "invalid username or password")
	}

	// 3. Verify Password
	if err := security.VerifyPassword(user.PasswordHash, password); err != nil {
		return nil, errors.New(errors.ErrUnauthorized, "invalid username or password")
	}

	// 4. CHECK BAN STATUS: If the user is deactivated, prevent login and show localized message.
	if !user.IsActive {
		// Fetch profile to see user's language preference
		profile, _ := s.profileRepo.GetByUserID(user.ID)
		lang := "fa" // Default to Persian as requested
		if profile != nil && profile.Locale != "" {
			lang = profile.Locale
		}

		// Use the i18n package to get the "account_banned" message
		msg := i18n.T(lang, "errors.account_banned")
		return nil, errors.New(errors.ErrForbidden, msg)
	}

	// 5. SUCCESS: Create Session and Tokens
	sessionID := uuid.New().String()
	tokenID := uuid.New().String() // Initial ID for rotation chain

	session := &domain.Session{
		ID:             sessionID,
		UserID:         user.ID,
		RefreshTokenID: tokenID,
		IP:             ip,
		UserAgent:      userAgent,
		ExpiresAt:      time.Now().Add(s.cfg.Auth.RefreshTokenExpiry),
		CreatedAt:      time.Now(),
	}

	// The SessionRepository.Set logic automatically enforces Single Device Policy (Kills old session).
	if err := s.sessionRepo.Set(ctx, session); err != nil {
		return nil, errors.New(errors.ErrInternalServer, "failed to establish session")
	}

	// 5. Generate JWTs
	acc, ref, err := s.jwt.GenerateTokenPair(user.ID, sessionID, tokenID)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServer, "token generation failed")
	}

	return &domain.TokenPair{
		AccessToken:  acc,
		RefreshToken: ref,
		ExpiresIn:    int64(s.cfg.Auth.AccessTokenExpiry.Seconds()),
	}, nil
}

// Refresh handles token rotation and detects potential theft/reuse.
func (s *authService) Refresh(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	// 1. Parse and validate the Refresh Token
	claims, err := s.jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New(errors.ErrUnauthorized, "invalid or expired refresh token")
	}

	// 2. Fetch session details from Redis
	session, err := s.sessionRepo.Get(ctx, claims.SessionID)
	if err != nil || session == nil {
		// Session was likely revoked (Logout, Ban, or Single Device Kick)
		return nil, errors.New(errors.ErrUnauthorized, "session no longer active")
	}

	// 3. HARDENING: Rotation & Reuse Detection
	// If the token ID in the JWT doesn't match the one stored in Redis,
	// it means this token was already used or is part of a stolen chain.
	if session.RefreshTokenID != claims.TokenID {
		// REUSE DETECTED: This is a high-security risk.
		// Kill the entire session immediately to block both the real user and the hacker.
		_ = s.sessionRepo.Delete(ctx, session.ID)
		return nil, errors.New(errors.ErrUnauthorized, "security alert: token reuse detected")
	}

	// 4. ROTATION: Generate a brand new Refresh Token ID
	newTokenID := uuid.New().String()
	session.RefreshTokenID = newTokenID
	session.ExpiresAt = time.Now().Add(s.cfg.Auth.RefreshTokenExpiry) // Sliding window session

	if err := s.sessionRepo.Set(ctx, session); err != nil {
		return nil, errors.New(errors.ErrInternalServer, "failed to rotate session")
	}

	// 5. Generate new tokens
	acc, ref, err := s.jwt.GenerateTokenPair(session.UserID, session.ID, newTokenID)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServer, "token generation failed")
	}

	return &domain.TokenPair{
		AccessToken:  acc,
		RefreshToken: ref,
		ExpiresIn:    int64(s.cfg.Auth.AccessTokenExpiry.Seconds()),
	}, nil
}

// Logout terminates the session and revokes all tokens instantly.
func (s *authService) Logout(ctx context.Context, accessToken string) error {
	// 1. Parse token (even if expired, we want the SID to clean up Redis)
	claims, _ := s.jwt.ParseAccessToken(accessToken)
	if claims == nil {
		return nil // Already invalid
	}

	// 2. Remove from Redis
	return s.sessionRepo.Delete(ctx, claims.SessionID)
}
