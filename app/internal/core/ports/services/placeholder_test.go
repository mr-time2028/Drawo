package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"drawo/internal/core/domain"
)

func TestAuthService_Placeholder(t *testing.T) {
	svc := NewAuthService()
	ctx := context.Background()

	u, err := svc.Register(ctx, "user", "pass")
	assert.Error(t, err)
	assert.Nil(t, u)

	tp, err := svc.Login(ctx, "user", "pass")
	assert.Error(t, err)
	assert.Nil(t, tp)

	tp, err = svc.Refresh(ctx, "token")
	assert.Error(t, err)
	assert.Nil(t, tp)

	err = svc.Logout(ctx, "a", "r")
	assert.Error(t, err)
}

func TestUserService_Placeholder(t *testing.T) {
	svc := NewUserService()
	ctx := context.Background()

	up, err := svc.GetProfile(ctx, "1")
	assert.Error(t, err)
	assert.Nil(t, up)

	p, err := svc.UpdateProfile(ctx, "1", domain.Profile{})
	assert.Error(t, err)
	assert.Nil(t, p)
}
