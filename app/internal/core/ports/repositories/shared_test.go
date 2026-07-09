package repositories

import (
	"context"
	"time"
	"testing"
    "github.com/stretchr/testify/assert"
)

// setupMemoryCache provides a shared mock for all repository unit tests.
func setupMemoryCache(t *testing.T) CacheRepository {
    return &mockCache{items: make(map[string]string)}
}

type mockCache struct {
    items map[string]string
}
func (m *mockCache) Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
    m.items[key] = val.(string); return nil 
}
func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
    v, ok := m.items[key]; if !ok { return "", assert.AnError }; return v, nil
}
func (m *mockCache) Delete(ctx context.Context, keys ...string) error {
    for _, k := range keys { delete(m.items, k) }; return nil
}
func (m *mockCache) Exists(ctx context.Context, keys ...string) (bool, error) {
    _, ok := m.items[keys[0]]; return ok, nil
}
func (m *mockCache) Close() error { return nil }
func (m *mockCache) Health(ctx context.Context) error { return nil }
