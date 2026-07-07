package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"drawo/internal/core/domain"
)

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

func TestRoomRepository(t *testing.T) {
    mc := setupMemoryCache(t)
    repo := NewRoomRepo(mc)
    ctx := context.Background()

    room := &domain.Room{ID: "r1", Name: "test", InviteCode: "INV1"}
    
    // Save
    assert.NoError(t, repo.Save(ctx, room))
    
    // GetByID
    f, err := repo.GetByID(ctx, "r1")
    assert.NoError(t, err)
    assert.Equal(t, "test", f.Name)
    
    // GetByInvite
    f, err = repo.GetByInviteCode(ctx, "inv1")
    assert.NoError(t, err)
    assert.Equal(t, "r1", f.ID)
    
    // Delete
    assert.NoError(t, repo.Delete(ctx, "r1", "INV1"))
    
    // ListPublic
    list, _ := repo.ListPublic(ctx, "en", domain.Paging{})
    assert.NotNil(t, list)
}
