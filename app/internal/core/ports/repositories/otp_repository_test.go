package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/internal/core/domain"
)

func TestOTPRepository(t *testing.T) {
	mc := setupMemoryCache(t)
	repo := NewOTPRepo(mc)
	ctx := context.Background()

	otp := &domain.OTP{
		Identifier: "test@example.com",
		Type:       domain.OTPEmail,
		Code:       "123456",
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	}

	// 1. Set
	err := repo.Set(ctx, otp)
	require.NoError(t, err)

	// 2. Get Success
	code, err := repo.Get(ctx, "test@example.com", domain.OTPEmail)
	require.NoError(t, err)
	assert.Equal(t, "123456", code)

	// 3. Delete
	err = repo.Delete(ctx, "test@example.com", domain.OTPEmail)
	assert.NoError(t, err)

	// 4. Get Fail
	_, err = repo.Get(ctx, "test@example.com", domain.OTPEmail)
	assert.Error(t, err)

	// 5. Set Expired
	otp.ExpiresAt = time.Now().Add(-1 * time.Minute)
	err = repo.Set(ctx, otp)
	assert.Error(t, err)
}
