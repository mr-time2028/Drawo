// Package middlewares contains Gin middleware for cross-cutting HTTP concerns.
//
// Responsibility:
//   - Attach request IDs and user IDs to the Gin context.
//   - Log requests and recover from panics.
//   - Apply CORS and rate limiting.
package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"drawo/pkg/logger"
)

// RequestIDHeader is the HTTP header used to accept or return a request ID.
const RequestIDHeader = "X-Request-ID"

// RequestID attaches a unique request ID to every incoming request.
//
// If the client provides one in X-Request-ID, we reuse it; otherwise we generate
// a UUID. The request ID is stored in both the Gin context and the standard
// context so logger.WithContext can find it.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(RequestIDHeader, requestID)
		c.Header(RequestIDHeader, requestID)

		ctx := logger.ContextWithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
