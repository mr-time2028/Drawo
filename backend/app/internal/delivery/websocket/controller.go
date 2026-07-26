// Package websocket exposes Drawo's realtime WebSocket delivery adapter.
//
// WebSocket starts with an HTTP Upgrade request, but after the upgrade it is a
// separate realtime protocol. Keeping this adapter outside delivery/http makes
// the project easier to navigate: HTTP REST routes stay in delivery/http and
// realtime WebSocket routes stay here.
package websocket

import (
	"github.com/gin-gonic/gin"

	"drawo/config"
	"drawo/internal/core/ports/repositories"
	"drawo/internal/realtime"
)

// Controller is intentionally thin: all protocol, authentication, heartbeat,
// re-authentication, and room dispatch rules live in internal/realtime.
type Controller struct {
	handler *realtime.Handler
}

func NewController(cfg config.Config, hub *realtime.Hub, sessions repositories.SessionRepository) *Controller {
	return &Controller{handler: realtime.NewHandler(cfg, hub, sessions)}
}

// Connect upgrades GET /api/v1/ws to a WebSocket connection.
// The first client frame must be auth, then join.
func (ctrl *Controller) Connect(c *gin.Context) {
	ctrl.handler.ServeHTTP(c.Writer, c.Request)
}
