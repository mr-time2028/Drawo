package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"drawo/config"
	"drawo/internal/core/ports"
	"drawo/pkg/logger"

	"github.com/redis/go-redis/v9"
)

// RedisClient implements ports.CacheRepository using go-redis.
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a Redis-backed ports.CacheRepository.
func NewRedisClient(cfg config.CacheConfig) (ports.CacheRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	logger.L.Info("connected to non-relational store", slog.String("driver", "redis"), slog.String("host", cfg.Host))

	return &RedisClient{client: client}, nil
}

// Set stores a key-value pair with an optional TTL.
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves the string representation of a stored value.
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found: %s", key)
	}
	return val, err
}

// Delete deletes one or more keys.
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if one or more keys exist.
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (bool, error) {
	count, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Health verifies Redis is reachable.
func (r *RedisClient) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis connection gracefully.
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Compile-time check: RedisClient implements ports.CacheRepository.
var _ ports.CacheRepository = (*RedisClient)(nil)
