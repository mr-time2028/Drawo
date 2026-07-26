package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"drawo/internal/core/ports/repositories"
)

type item struct {
	value      interface{}
	expiration int64
}

type MemoryClient struct {
	mu    sync.RWMutex
	items map[string]item
}

func NewMemoryClient() *MemoryClient {
	return &MemoryClient{
		items: make(map[string]item),
	}
}

func (m *MemoryClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}
	m.items[key] = item{
		value:      value,
		expiration: exp,
	}
	return nil
}

func (m *MemoryClient) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	it, ok := m.items[key]
	if !ok || (it.expiration > 0 && time.Now().UnixNano() > it.expiration) {
		return "", ErrCacheMiss
	}
	return fmt.Sprintf("%v", it.value), nil
}

func (m *MemoryClient) Delete(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.items, k)
	}
	return nil
}

func (m *MemoryClient) Exists(ctx context.Context, keys ...string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, k := range keys {
		it, ok := m.items[k]
		if ok && (it.expiration == 0 || time.Now().UnixNano() <= it.expiration) {
			return true, nil
		}
	}
	return false, nil
}

func (m *MemoryClient) Close() error                     { return nil }
func (m *MemoryClient) Health(ctx context.Context) error { return nil }

var _ repositories.CacheRepository = (*MemoryClient)(nil)
