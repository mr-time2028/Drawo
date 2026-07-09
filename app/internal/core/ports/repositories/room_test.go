package repositories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"drawo/internal/core/domain"
)

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
