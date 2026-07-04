// Package redis provides Redis connectivity for caching, sessions, and real-time coordination.
//
// Responsibility:
//   - Open a go-redis connection.
//   - Implement ports.HealthReporter.
//
// Why Redis?
//   PostgreSQL is great for durable data, but too slow for per-second
//   operations like rate limiting, session lookup, and WebSocket presence.
//   Redis handles these with sub-millisecond latency.
package redis

import (
	"context"
	"fmt"

	"drawo/config"
	"drawo/internal/core/ports"

	"github.com/redis/go-redis/v9"
)

// Client wraps go-redis.Client to add health reporting.
type Client struct {
	redis.Cmdable
	client *redis.Client
}

// NewClient creates a Redis client from configuration.
func NewClient(cfg config.RedisConfig) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Client{Cmdable: client, client: client}, nil
}

// Health verifies Redis is reachable.
func (c *Client) Health(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection gracefully.
func (c *Client) Close() error {
	return c.client.Close()
}

// Compile-time check: Client implements ports.HealthReporter.
var _ ports.HealthReporter = (*Client)(nil)
