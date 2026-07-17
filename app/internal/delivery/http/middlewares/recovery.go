package middlewares

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"drawo/pkg/logger"
)

// Recovery recovers from panics in HTTP handlers and returns a 500 response.
//
// Gin has a built-in recovery middleware, but this one also logs the panic with
// the request ID so panics can be traced back to a specific request.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.WithContext(c.Request.Context()).Error(
					"panic recovered in http handler",
					slog.Any("error", err),
					slog.String("path", c.Request.URL.Path),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"message": "internal server error",
				})
			}
		}()

		c.Next()
	}
}
