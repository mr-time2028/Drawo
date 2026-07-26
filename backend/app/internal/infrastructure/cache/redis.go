package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"drawo/config"
	"drawo/internal/core/ports/repositories"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(cfg config.CacheConfig) *RedisClient {
	return &RedisClient{
		client: redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
			Password: cfg.Password,
			DB:       cfg.DB,
		}),
	}
}

func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrCacheMiss
	}
	return val, err
}

func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisClient) Exists(ctx context.Context, keys ...string) (bool, error) {
	count, err := r.client.Exists(ctx, keys...).Result()
	return count > 0, err
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

func (r *RedisClient) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

var _ repositories.CacheRepository = (*RedisClient)(nil)
