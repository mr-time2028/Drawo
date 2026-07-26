package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/config"
	"drawo/internal/infrastructure/cache"
)

func TestRateLimiter_Allow(t *testing.T) {
	mc, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	defer mc.Close()

	limiter := NewRateLimiter(mc)
	ctx := context.Background()
	key := "login:user-1"
	limit := 3
	window := 100 * time.Millisecond

	// 1. First 3 requests allowed
	for i := 0; i < limit; i++ {
		allowed, err := limiter.Allow(ctx, key, limit, window)
		assert.NoError(t, err)
		assert.True(t, allowed)
	}

	// 2. 4th blocked
	allowed, err := limiter.Allow(ctx, key, limit, window)
	assert.NoError(t, err)
	assert.False(t, allowed)

	// 3. Wait for window
	time.Sleep(window + 10*time.Millisecond)

	allowed, err = limiter.Allow(ctx, key, limit, window)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

type failCache struct {
	cache.MemoryClient // inherit most methods
}

func (f *failCache) Get(ctx context.Context, key string) (string, error) { return "", assert.AnError }
func (f *failCache) Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	return assert.AnError
}

func TestRateLimiter_Failures(t *testing.T) {
	limiter := NewRateLimiter(&failCache{})
	_, err := limiter.Allow(context.Background(), "k", 1, time.Hour)
	assert.Error(t, err)
}
