package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/internal/core/domain"
)

func TestSessionRepository_SingleDevicePolicy(t *testing.T) {
	mc := setupMemoryCache(t)
	repo := NewSessionRepo(mc)
	ctx := context.Background()
	userID := "hamid-123"

	// 1. Hamid logs in from "Laptop"
	laptopSession := &domain.Session{
		ID:        "laptop-uuid",
		UserID:    userID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	err := repo.Set(ctx, laptopSession)
	require.NoError(t, err)

	// Verify laptop session is active
	f, _ := repo.Get(ctx, "laptop-uuid")
	assert.NotNil(t, f)

	// 2. Hamid logs in from "Phone"
	phoneSession := &domain.Session{
		ID:        "phone-uuid",
		UserID:    userID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	err = repo.Set(ctx, phoneSession)
	require.NoError(t, err)

	// 3. THE CRITICAL CHECK:
	// The Laptop session must now be GONE (deleted by Single Device Policy)
	f, _ = repo.Get(ctx, "laptop-uuid")
	assert.Nil(t, f, "Laptop session should have been invalidated by Phone login")

	// The Phone session must be active
	f, _ = repo.Get(ctx, "phone-uuid")
	assert.NotNil(t, f)
}

func TestSessionRepository_BasicCRUD(t *testing.T) {
	mc := setupMemoryCache(t)
	repo := NewSessionRepo(mc)
	ctx := context.Background()

	sess := &domain.Session{
		ID:        "s1",
		UserID:    "u1",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Set
	assert.NoError(t, repo.Set(ctx, sess))
	// Get
	f, _ := repo.Get(ctx, "s1")
	assert.Equal(t, "u1", f.UserID)
	// Delete
	assert.NoError(t, repo.Delete(ctx, "s1"))
	f, _ = repo.Get(ctx, "s1")
	assert.Nil(t, f)
	
	// DeleteAllForUser
	repo.Set(ctx, sess)
	assert.NoError(t, repo.DeleteAllForUser(ctx, "u1"))
	f, _ = repo.Get(ctx, "s1")
	assert.Nil(t, f)
}
