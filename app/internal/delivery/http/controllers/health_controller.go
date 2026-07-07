// Package controllers contains Gin HTTP handlers.
//
// Controllers are thin adapters. They should:
//   - Parse and validate request input.
//   - Call application 
//   - Map service errors to HTTP responses.
//
// Controllers must NOT contain business logic.
package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"drawo/internal/infrastructure/di"
)

// HealthController exposes health and readiness endpoints.
type HealthController struct {
	Container *di.Container
}

// NewHealthController creates a health controller.
func NewHealthController(container *di.Container) *HealthController {
	return &HealthController{Container: container}
}

// Ping returns a simple 200 OK. Use it for load balancer health checks.
func (ctrl *HealthController) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Health reports the status of every infrastructure dependency.
func (ctrl *HealthController) Health(c *gin.Context) {
	results := ctrl.Container.Health()

	status := http.StatusOK
	deps := make(map[string]string, len(results))
	for name, err := range results {
		if err != nil {
			status = http.StatusServiceUnavailable
			deps[name] = "unhealthy: " + err.Error()
		} else {
			deps[name] = "healthy"
		}
	}

	c.JSON(status, gin.H{
		"status":      deps,
		"request_id":  c.GetString("X-Request-ID"),
	})
}
