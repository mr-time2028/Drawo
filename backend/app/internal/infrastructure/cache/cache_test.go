package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/config"
	"drawo/internal/core/ports/repositories"
)

func TestNewClient_UnsupportedDriver(t *testing.T) {
	cfg := config.CacheConfig{
		Driver: "unknown-store",
	}

	client, err := NewClient(cfg)
	assert.Nil(t, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported cache driver")
}

func TestRegisterDriver(t *testing.T) {
	mockErr := fmt.Errorf("mock cache init error")
	RegisterDriver("mockstore", func(cfg config.CacheConfig) (repositories.CacheRepository, error) {
		return nil, mockErr
	})

	client, err := NewClient(config.CacheConfig{Driver: "mockstore"})
	assert.Nil(t, client)
	assert.Equal(t, mockErr, err)
}

func TestMemoryClient_Operations(t *testing.T) {
	client, err := NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	ctx := context.Background()

	// Verify Health
	assert.NoError(t, client.Health(ctx))

	// Test Set and Get
	err = client.Set(ctx, "session:123", "user_abc", 1*time.Minute)
	require.NoError(t, err)

	val, err := client.Get(ctx, "session:123")
	require.NoError(t, err)
	assert.Equal(t, "user_abc", val)

	// Test Exists
	exists, err := client.Exists(ctx, "session:123")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = client.Exists(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)

	// Test Delete
	err = client.Delete(ctx, "session:123")
	require.NoError(t, err)

	exists, err = client.Exists(ctx, "session:123")
	require.NoError(t, err)
	assert.False(t, exists)

	_, err = client.Get(ctx, "session:123")
	assert.Error(t, err)
}

func TestMemoryClient_TTL(t *testing.T) {
	client, err := NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	err = client.Set(ctx, "temp_key", "temp_val", 50*time.Millisecond)
	require.NoError(t, err)

	val, err := client.Get(ctx, "temp_key")
	require.NoError(t, err)
	assert.Equal(t, "temp_val", val)

	time.Sleep(100 * time.Millisecond)

	exists, err := client.Exists(ctx, "temp_key")
	require.NoError(t, err)
	assert.False(t, exists)

	_, err = client.Get(ctx, "temp_key")
	assert.Error(t, err)
}
