package cache

import (
	"fmt"

	"drawo/config"
	"drawo/internal/core/ports/repositories"
)

type Driver func(cfg config.CacheConfig) (repositories.CacheRepository, error)

var drivers = make(map[string]Driver)

func RegisterDriver(name string, driver Driver) {
	drivers[name] = driver
}

func NewClient(cfg config.CacheConfig) (repositories.CacheRepository, error) {
	driver, ok := drivers[cfg.Driver]
	if !ok {
		return nil, fmt.Errorf("unsupported cache driver: %s", cfg.Driver)
	}
	return driver(cfg)
}

var ErrCacheMiss = fmt.Errorf("key not found in cache")

func init() {
	RegisterDriver("redis", func(cfg config.CacheConfig) (repositories.CacheRepository, error) {
		return NewRedisClient(cfg), nil
	})
	RegisterDriver("memory", func(cfg config.CacheConfig) (repositories.CacheRepository, error) {
		return NewMemoryClient(), nil
	})
}
