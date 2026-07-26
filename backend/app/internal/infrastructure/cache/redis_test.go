package cache

import (
	"context"
	"testing"
	"time"

	"drawo/config"
	"github.com/stretchr/testify/assert"
)

func TestRedisClient_Failures(t *testing.T) {
	cfg := config.CacheConfig{
		Host: "localhost",
		Port: "12345", // Wrong port
	}
	client := NewRedisClient(cfg)
	assert.NotNil(t, client)
	defer client.Close()

	ctx := context.Background()

	// All these should fail but coverage should be hit
	assert.Error(t, client.Health(ctx))
	assert.Error(t, client.Set(ctx, "k", "v", time.Second))
	_, err := client.Get(ctx, "k")
	assert.Error(t, err)
	assert.Error(t, client.Delete(ctx, "k"))
	_, err = client.Exists(ctx, "k")
	assert.Error(t, err)
}
