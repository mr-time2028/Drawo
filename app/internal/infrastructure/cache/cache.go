// Package cache provides non-relational database connectivity for caching, sessions, and real-time coordination.
//
// Responsibility:
//   - Provide a factory registry so switching between caching/non-relational stores (Redis, in-memory, etc.)
//     requires minimal code changes and no technology-specific logic leaking into services or controllers.
//   - Return implementations of ports.CacheRepository based on configuration.
package cache

import (
	"fmt"
	"strings"

	"drawo/config"
	"drawo/internal/core/ports"
)

// Factory creates a ports.CacheRepository given a cache configuration.
type Factory func(cfg config.CacheConfig) (ports.CacheRepository, error)

var factories = map[string]Factory{
	"redis":  NewRedisClient,
	"memory": NewMemoryClient,
}

// RegisterDriver registers a custom Factory for a non-relational / cache driver name
// (e.g., "memcached"). This allows extending caching support without modifying core logic.
func RegisterDriver(name string, factory Factory) {
	factories[strings.ToLower(name)] = factory
}

// NewClient creates a ports.CacheRepository from configuration using the configured driver.
func NewClient(cfg config.CacheConfig) (ports.CacheRepository, error) {
	driver := strings.ToLower(cfg.Driver)
	if driver == "" {
		driver = "redis"
	}

	factory, ok := factories[driver]
	if !ok {
		return nil, fmt.Errorf("unsupported cache driver: %q (supported drivers can be registered via RegisterDriver)", cfg.Driver)
	}

	return factory(cfg)
}
