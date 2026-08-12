package middlewares

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	appErrors "drawo/pkg/errors"
)

// Recovery recovers from panics in HTTP handlers, logs them in a simple
// plain-text format to stderr (standard library log.Printf), and returns a
// generic 500 response to the client.
//
// We intentionally use the standard logger here rather than a structured
// logger, per project convention for unexpected bugs: internal errors should
// produce a straightforward timestamped line in the terminal that is easy to
// grep and read during local development.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID, _ := c.Get(appErrors.RequestIDKey)
				log.Printf(
					"[PANIC RECOVERED] method=%s path=%s request_id=%v client_ip=%s panic=%v",
					c.Request.Method,
					c.Request.URL.Path,
					requestID,
					c.ClientIP(),
					rec,
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"message": "internal server error",
				})
			}
		}()

		c.Next()
	}
}
