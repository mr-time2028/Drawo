// Package services defines the high-level business logic of Drawo.
package services

import (
	"context"
	"fmt"
	"time"

	"drawo/internal/core/ports/repositories"
)

// RateLimiter defines the contract for protecting resources from abuse.
type RateLimiter interface {
	// Allow checks if an action should be permitted based on the current window.
	// Returns true if allowed, false if limited.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// slidingWindowRateLimiter implements a high-precision rate limiter using Redis/Cache.
type slidingWindowRateLimiter struct {
	cache repositories.CacheRepository
}

// NewRateLimiter creates a new rate limiter service.
func NewRateLimiter(cache repositories.CacheRepository) RateLimiter {
	return &slidingWindowRateLimiter{cache: cache}
}

// Allow implements a sliding window algorithm.
//
// HOW IT WORKS:
//  1. We create a unique key for the user/action.
//  2. We store a counter.
//  3. If the count < Limit, we allow and record the new action.
func (r *slidingWindowRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	fullKey := fmt.Sprintf("rl:%s", key)

	// Check if we have a counter
	val, err := r.cache.Get(ctx, fullKey)
	count := 0
	if err == nil {
		fmt.Sscanf(val, "%d", &count)
	}

	if count >= limit {
		return false, nil // Rate limited!
	}

	// Increment and set with TTL of the window
	err = r.cache.Set(ctx, fullKey, fmt.Sprintf("%d", count+1), window)
	if err != nil {
		return false, err
	}

	return true, nil
}
