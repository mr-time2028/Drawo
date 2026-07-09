// Package repositories defines data persistence contracts and implementations.
package repositories

import (
	"context"
	"fmt"
	"time"

	"drawo/internal/core/domain"
)

// OTPRepository defines the contract for storing and verifying temporary codes.
type OTPRepository interface {
	// Set saves an OTP in the cache with a TTL.
	Set(ctx context.Context, otp *domain.OTP) error
	// Get retrieves an OTP by identifier and type.
	Get(ctx context.Context, identifier string, otpType domain.OTPType) (string, error)
	// Delete removes an OTP after successful verification or expiration.
	Delete(ctx context.Context, identifier string, otpType domain.OTPType) error
}

// otpRepo implements OTPRepository using the application's CacheRepository.
type otpRepo struct {
	cache CacheRepository
}

// NewOTPRepo creates a new modular OTP repository.
func NewOTPRepo(cache CacheRepository) OTPRepository {
	return &otpRepo{cache: cache}
}

// otpKey returns a unique cache key for a specific OTP.
func (r *otpRepo) otpKey(id string, t domain.OTPType) string {
	return fmt.Sprintf("otp:%s:%s", t, id)
}

func (r *otpRepo) Set(ctx context.Context, otp *domain.OTP) error {
	ttl := time.Until(otp.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("otp already expired")
	}
	return r.cache.Set(ctx, r.otpKey(otp.Identifier, otp.Type), otp.Code, ttl)
}

func (r *otpRepo) Get(ctx context.Context, identifier string, otpType domain.OTPType) (string, error) {
	return r.cache.Get(ctx, r.otpKey(identifier, otpType))
}

func (r *otpRepo) Delete(ctx context.Context, identifier string, otpType domain.OTPType) error {
	return r.cache.Delete(ctx, r.otpKey(identifier, otpType))
}
