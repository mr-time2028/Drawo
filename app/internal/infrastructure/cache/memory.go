package cache

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"drawo/config"
	"drawo/internal/core/ports"
	"drawo/pkg/logger"
)

type memoryItem struct {
	value     interface{}
	expiresAt time.Time
}

func (m *memoryItem) isExpired() bool {
	if m.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(m.expiresAt)
}

// MemoryClient implements ports.CacheRepository using an in-memory sync.Map.
// Useful for local development, testing, or deployments without external caching engines.
type MemoryClient struct {
	items  sync.Map
	closed chan struct{}
}

// NewMemoryClient creates an in-memory ports.CacheRepository.
func NewMemoryClient(cfg config.CacheConfig) (ports.CacheRepository, error) {
	logger.L.Info("initialized non-relational store", slog.String("driver", "memory"))
	m := &MemoryClient{
		closed: make(chan struct{}),
	}
	go m.cleanupLoop()
	return m, nil
}

func (m *MemoryClient) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-m.closed:
			return
		case <-ticker.C:
			now := time.Now()
			m.items.Range(func(key, val interface{}) bool {
				item, ok := val.(*memoryItem)
				if ok && !item.expiresAt.IsZero() && now.After(item.expiresAt) {
					m.items.Delete(key)
				}
				return true
			})
		}
	}
}

// Set stores a key-value pair with an optional TTL.
func (m *MemoryClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	m.items.Store(key, &memoryItem{value: value, expiresAt: exp})
	return nil
}

// Get retrieves the string representation of a stored value.
func (m *MemoryClient) Get(ctx context.Context, key string) (string, error) {
	val, ok := m.items.Load(key)
	if !ok {
		return "", fmt.Errorf("key not found: %s", key)
	}
	item, ok := val.(*memoryItem)
	if !ok || item.isExpired() {
		m.items.Delete(key)
		return "", fmt.Errorf("key not found: %s", key)
	}
	return fmt.Sprintf("%v", item.value), nil
}

// Delete deletes one or more keys.
func (m *MemoryClient) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		m.items.Delete(key)
	}
	return nil
}

// Exists checks if one or more keys exist.
func (m *MemoryClient) Exists(ctx context.Context, keys ...string) (bool, error) {
	count := 0
	for _, key := range keys {
		if val, ok := m.items.Load(key); ok {
			item, ok := val.(*memoryItem)
			if ok && !item.isExpired() {
				count++
			} else {
				m.items.Delete(key)
			}
		}
	}
	return count > 0, nil
}

// Health always reports healthy for memory client.
func (m *MemoryClient) Health(ctx context.Context) error {
	return nil
}

// Close stops the cleanup loop.
func (m *MemoryClient) Close() error {
	select {
	case <-m.closed:
	default:
		close(m.closed)
	}
	return nil
}

// Compile-time check: MemoryClient implements ports.CacheRepository.
var _ ports.CacheRepository = (*MemoryClient)(nil)
