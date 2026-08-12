package websocket

import (
	"context"

	"github.com/gin-gonic/gin"

	"drawo/internal/core/domain"
	"drawo/internal/infrastructure/di"
)

// Register attaches realtime WebSocket routes to the main Gin engine.
//
// The endpoint is still mounted on the same HTTP server because WebSocket starts
// as an HTTP Upgrade request, but the route lives in delivery/websocket to keep
// protocol boundaries clear.
func Register(router *gin.Engine, container *di.Container) {
	// Adapt repositories.UserRepository (no ctx on GetByID — legacy signature)
	// to the realtime.UserLookup narrow interface (ctx-aware). We drop ctx
	// here; the underlying GORM repo doesn't use it.
	userLookup := userLookupAdapter{repo: container.Repos.User}
	ctrl := NewController(container.Config, container.Hub, container.Sessions, container.Services.Room, userLookup)
	router.GET("/api/v1/ws", ctrl.Connect)
}

type userLookupAdapter struct {
	repo interface {
		GetByID(id string) (*domain.User, error)
	}
}

func (a userLookupAdapter) GetByID(_ context.Context, id string) (*domain.User, error) {
	return a.repo.GetByID(id)
}
