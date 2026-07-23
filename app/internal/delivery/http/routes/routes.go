// Package routes wires HTTP controllers to URL paths.
//
// Responsibility:
//   - Register routes and middleware groups.
//   - Keep routing centralized so it is easy to see the whole API surface.
package routes

import (
	"os"

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

	// Serve locally stored uploads when local storage is selected.
	// The storage provider returns URLs like /uploads/<bucket>/<object>.
	if container != nil && container.Config.App.Storage.Driver == "local" {
		_ = os.MkdirAll(container.Config.App.Storage.UploadDirectory, 0755)
		router.Static("/uploads", container.Config.App.Storage.UploadDirectory)
	}

	// Health endpoints (no auth).
	health := controllers.NewHealthController(container)
	router.GET("/health/ping", health.Ping)
	router.GET("/health", health.Health)

	// API v1 group.
	api := router.Group("/api/v1")
	{
		// Auth routes
		authCtrl := controllers.NewAuthController(container.Services.Auth)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authCtrl.Register)
			auth.POST("/login", authCtrl.Login)
			auth.POST("/refresh", authCtrl.Refresh)
			auth.POST("/logout", authCtrl.Logout)
		}

		// User and profile routes
		userCtrl := controllers.NewUserController(container.Services.User)
		user := api.Group("/user")
		user.Use(middlewares.RequireAuth(container))
		{
			user.GET("/profile", userCtrl.GetProfile)
			user.PATCH("/profile", userCtrl.UpdateProfile)
			user.POST("/profile/username", userCtrl.ChangeUsername)
			user.POST("/profile/verify/request", userCtrl.RequestVerification)
			user.POST("/profile/verify/confirm", userCtrl.ConfirmVerification)
		}

		// Admin routes
		adminCtrl := controllers.NewAdminController(container.Services.Admin)
		admin := api.Group("/admin")
		admin.Use(middlewares.RequireAuth(container), middlewares.RequireAdmin())
		{
			// Song management
			admin.POST("/songs", adminCtrl.UploadSong)
			admin.GET("/songs", adminCtrl.ListSongs)
			admin.PATCH("/songs/:id/toggle", adminCtrl.ToggleSong)
			admin.DELETE("/songs/:id", adminCtrl.DeleteSong)

			// User management
			admin.GET("/users/search", adminCtrl.SearchUsers)
			admin.POST("/users/:id/ban", adminCtrl.BanUser)
			admin.POST("/users/:id/unban", adminCtrl.UnbanUser)

			// Bad word management
			admin.POST("/bad-words", adminCtrl.CreateBadWord)
			admin.GET("/bad-words", adminCtrl.ListBadWords)
			admin.DELETE("/bad-words/:id", adminCtrl.DeleteBadWord)

			// Report moderation
			admin.GET("/reports", adminCtrl.ListReports)
			admin.GET("/reports/:id", adminCtrl.GetReport)
			admin.POST("/reports/:id/confirm", adminCtrl.ConfirmReport)
			admin.POST("/reports/:id/reject", adminCtrl.RejectReport)

			// Settings
			admin.PATCH("/settings/:key", adminCtrl.UpdateSetting)
		}
	}
}
