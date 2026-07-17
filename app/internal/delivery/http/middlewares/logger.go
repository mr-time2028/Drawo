package middlewares

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"drawo/pkg/logger"
)

// Logger logs every HTTP request with method, path, status, latency, and client IP.
//
// It runs after the request completes so it has access to the response status code.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		logger.WithContext(c.Request.Context()).LogAttrs(
			c.Request.Context(),
			level,
			"http request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("client_ip", c.ClientIP()),
			slog.Int("errors", len(c.Errors)),
		)
	}
}
