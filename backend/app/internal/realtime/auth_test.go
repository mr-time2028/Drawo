package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/internal/infrastructure/cache"
	"drawo/pkg/security"
)

func TestAuthenticator(t *testing.T) {
	cfg := config.Get()
	cfg.App.SecretKey = "authenticator-secret"
	cfg.Auth.Issuer = "drawo"
	cfg.Auth.AccessTokenExpiry = time.Hour
	cfg.Auth.RefreshTokenExpiry = time.Hour

	cacheClient, err := cache.NewClient(config.CacheConfig{Driver: "memory"})
	require.NoError(t, err)
	defer cacheClient.Close()

	sessions := repositories.NewSessionRepo(cacheClient)
	authenticator := NewAuthenticator(cfg, sessions)
	jwt := security.NewJWTManager(cfg.App.SecretKey, cfg.Auth.Issuer, cfg.Auth.AccessTokenExpiry, cfg.Auth.RefreshTokenExpiry)

	access, refresh, err := jwt.GenerateTokenPair("u1", "s1", "t1")
	require.NoError(t, err)

	_, err = authenticator.AuthenticateAccessToken(context.Background(), "not-a-token")
	assert.Error(t, err)

	_, err = authenticator.AuthenticateAccessToken(context.Background(), refresh)
	assert.Error(t, err)

	_, err = authenticator.AuthenticateAccessToken(context.Background(), access)
	assert.Error(t, err)

	require.NoError(t, sessions.Set(context.Background(), &domain.Session{ID: "s1", UserID: "different-user", ExpiresAt: time.Now().Add(time.Hour)}))
	_, err = authenticator.AuthenticateAccessToken(context.Background(), access)
	assert.Error(t, err)

	require.NoError(t, sessions.Set(context.Background(), &domain.Session{ID: "s1", UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)}))
	ctx, err := authenticator.AuthenticateAccessToken(context.Background(), access)
	require.NoError(t, err)
	assert.Equal(t, "u1", ctx.UserID)
	assert.True(t, authenticator.SessionActive(context.Background(), ctx))

	require.NoError(t, sessions.Delete(context.Background(), "s1"))
	assert.False(t, authenticator.SessionActive(context.Background(), ctx))
}
