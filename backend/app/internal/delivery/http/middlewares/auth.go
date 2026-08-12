// Package middlewares contains Gin-compatible authentication and security guards.
package middlewares

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"drawo/internal/infrastructure/di"
	appErrors "drawo/pkg/errors"
	"drawo/pkg/security"
)

const (
	ContextUserID      = "userID"
	ContextIsSuperuser = "isSuperuser"
)

// RequireAuth ensures the request has a valid JWT and an active Redis session.
//
// Expected failure modes return 401 with a safe message:
//   - missing or malformed Authorization header
//   - bad JWT signature or expired token
//   - session missing (revoked / logged out / expired)
//   - user not found (deleted account between requests)
//
// Unexpected errors (Redis down, DB down, network blip) are logged to stderr
// via appErrors.RespondError (standard log.Printf) and returned as 500 with a
// generic message — internal details never leak to the client.
func RequireAuth(container *di.Container) gin.HandlerFunc {
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
			status, body := appErrors.New(appErrors.ErrUnauthorized, "authentication required").Response()
			c.AbortWithStatusJSON(status, body)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// 2. Parse and Validate JWT Signature
		claims, err := jwt.ParseAccessToken(tokenStr)
		if err != nil {
			status, body := appErrors.New(appErrors.ErrUnauthorized, "invalid or expired token").Response()
			c.AbortWithStatusJSON(status, body)
			return
		}

		// 3. Check Redis Session
		session, err := container.Sessions.Get(c.Request.Context(), claims.SessionID)
		if err != nil {
			// Unexpected Redis/infrastructure error — log and 500.
			appErrors.RespondError(c, err)
			c.Abort()
			return
		}
		if session == nil {
			// Expected: session was revoked/expired → 401.
			status, body := appErrors.New(appErrors.ErrUnauthorized, "session no longer active").Response()
			c.AbortWithStatusJSON(status, body)
			return
		}

		// 4. Fetch User for admin flag
		user, err := container.Services.User.GetProfile(c.Request.Context(), claims.UserID)
		if err != nil {
			// If the service returned ErrNotFound (AppError), treat as auth
			// failure (user was deleted). Any other error (DB down, etc.) is
			// an unexpected 500 and RespondError will log it.
			var appErr *appErrors.AppError
			if errors.As(err, &appErr) && errors.Is(appErr.Err, appErrors.ErrNotFound) {
				status, body := appErrors.New(appErrors.ErrUnauthorized, "user account not found").Response()
				c.AbortWithStatusJSON(status, body)
				return
			}
			appErrors.RespondError(c, err)
			c.Abort()
			return
		}

		// 5. Inject into Context for controllers
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextIsSuperuser, user.User.IsSuperuser)

		c.Next()
	}
}

// OptionalAuth works like RequireAuth but does NOT abort when no Authorization
// header is present. Controllers can inspect `c.Get(ContextUserID)` — an empty
// string means the request is anonymous. Used on endpoints where guests are
// allowed (e.g. private-room invite join).
func OptionalAuth(container *di.Container) gin.HandlerFunc {
	jwt := security.NewJWTManager(
		container.Config.App.SecretKey,
		container.Config.Auth.Issuer,
		container.Config.Auth.AccessTokenExpiry,
		container.Config.Auth.RefreshTokenExpiry,
	)
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwt.ParseAccessToken(tokenStr)
		if err != nil {
			c.Next()
			return
		}
		session, err := container.Sessions.Get(c.Request.Context(), claims.SessionID)
		if err != nil || session == nil {
			c.Next()
			return
		}
		user, err := container.Services.User.GetProfile(c.Request.Context(), claims.UserID)
		if err != nil {
			c.Next()
			return
		}
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
			status, body := appErrors.New(appErrors.ErrForbidden, "administrator permissions required").Response()
			c.AbortWithStatusJSON(status, body)
			return
		}
		c.Next()
	}
}
