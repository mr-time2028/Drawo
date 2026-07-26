package websocket

import (
	"github.com/gin-gonic/gin"

	"drawo/internal/infrastructure/di"
)

// Register attaches realtime WebSocket routes to the main Gin engine.
//
// The endpoint is still mounted on the same HTTP server because WebSocket starts
// as an HTTP Upgrade request, but the route lives in delivery/websocket to keep
// protocol boundaries clear.
func Register(router *gin.Engine, container *di.Container) {
	ctrl := NewController(container.Config, container.Hub, container.Sessions)
	router.GET("/api/v1/ws", ctrl.Connect)
}
