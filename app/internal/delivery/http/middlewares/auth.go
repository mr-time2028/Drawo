// Package middlewares contains Gin-compatible authentication and security guards.
package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"

	"drawo/internal/infrastructure/di"
	"drawo/pkg/errors"
	"drawo/pkg/security"
)

const (
	ContextUserID      = "userID"
	ContextIsSuperuser = "isSuperuser"
)

// RequireAuth ensures the request has a valid JWT and an active Redis session.
func RequireAuth(container *di.Container) gin.HandlerFunc {
	// Initialize JWT manager once for the middleware
	jwt := security.NewJWTManager(
		container.Config.App.SecretKey,
		container.Config.Auth.Issuer,
		container.Config.Auth.AccessTokenExpiry,
		container.Config.Auth.RefreshTokenExpiry,
	)

	return func(c *gin.Context) {
		// 1. Extract Bearer Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(errors.New(errors.ErrUnauthorized, "authentication required").Response())
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// 2. Parse and Validate JWT Signature
		claims, err := jwt.ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(errors.New(errors.ErrUnauthorized, "invalid or expired token").Response())
			return
		}

		// 3. SECURITY CHECK: Check Redis Session (Single Device Policy)
		// This verifies that the session hasn't been revoked/kicked.
		session, err := container.Sessions.Get(c.Request.Context(), claims.SessionID)
		if err != nil || session == nil {
			c.AbortWithStatusJSON(errors.New(errors.ErrUnauthorized, "session no longer active").Response())
			return
		}

		// 4. Fetch User to check Admin status (Optimization: could be stored in Redis if needed)
		user, err := container.Services.User.GetProfile(c.Request.Context(), claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(errors.New(errors.ErrUnauthorized, "user account not found").Response())
			return
		}

		// 5. Inject into Context for use in controllers
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextIsSuperuser, user.User.IsSuperuser)

		c.Next()
	}
}

// RequireAdmin restricts the endpoint to superusers only.
// Must be used AFTER RequireAuth.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		isSuperuser, exists := c.Get(ContextIsSuperuser)
		if !exists || !isSuperuser.(bool) {
			c.AbortWithStatusJSON(errors.New(errors.ErrForbidden, "administrator permissions required").Response())
			return
		}
		c.Next()
	}
}
