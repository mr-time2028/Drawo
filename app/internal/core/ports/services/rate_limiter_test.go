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
	// Use memory client for isolation
	mc, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	defer mc.Close()

	limiter := NewRateLimiter(mc)
	ctx := context.Background()
	key := "login:user-1"
	limit := 3
	window := 100 * time.Millisecond

	// 1. First 3 requests should be allowed
	for i := 0; i < limit; i++ {
		allowed, err := limiter.Allow(ctx, key, limit, window)
		assert.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}

	// 2. 4th request should be blocked
	allowed, err := limiter.Allow(ctx, key, limit, window)
	assert.NoError(t, err)
	assert.False(t, allowed, "4th request should be blocked")

	// 3. Wait for window to pass
	time.Sleep(window + 10*time.Millisecond)

	// 4. Should be allowed again after window (Memory fallback uses simple TTL)
	allowed, err = limiter.Allow(ctx, key, limit, window)
	assert.NoError(t, err)
	assert.True(t, allowed, "Request after window should be allowed")
}
