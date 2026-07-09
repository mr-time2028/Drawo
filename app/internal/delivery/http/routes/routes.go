// Package routes wires HTTP controllers to URL paths.
//
// Responsibility:
//   - Register routes and middleware groups.
//   - Keep routing centralized so it is easy to see the whole API surface.
package routes

import (
	"github.com/gin-gonic/gin"

	"drawo/internal/delivery/http/controllers"
	"drawo/internal/delivery/http/middlewares"
	"drawo/internal/infrastructure/di"
)

// Register adds all application routes to the provided Gin engine.
func Register(router *gin.Engine, container *di.Container) {
	// Global middleware.
	router.Use(middlewares.RequestID())
	router.Use(middlewares.Logger())
	router.Use(middlewares.Recovery())
	router.Use(middlewares.CORS())

	// Health endpoints (no auth).
	health := controllers.NewHealthController(container)
	router.GET("/health/ping", health.Ping)
	router.GET("/health", health.Health)

	// API v1 group.
	api := router.Group("/api/v1")
	{
		// Auth routes (Phase 4)
		authCtrl := controllers.NewAuthController(container.Services.Auth)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authCtrl.Register)
			auth.POST("/login", authCtrl.Login)
			auth.POST("/refresh", authCtrl.Refresh)
		}
	}
}
