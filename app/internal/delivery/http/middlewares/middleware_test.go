package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMiddlewares(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(RequestID())
	router.Use(Logger())
	router.Use(Recovery())
	router.Use(CORS())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

    router.GET("/panic", func(c *gin.Context) {
        panic("oops")
    })

	t.Run("RequestID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
		assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	})

    t.Run("Recovery", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/panic", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

    t.Run("CORS", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("OPTIONS", "/test", nil)
        req.Header.Set("Origin", "http://localhost:3000")
        req.Header.Set("Access-Control-Request-Method", "GET")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}
