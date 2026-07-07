package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"drawo/internal/infrastructure/di"
    "drawo/config"
)

func TestHealthController(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	cfg := config.Get()
	cfg.Cache.Driver = "memory"
    cfg.Database.Driver = "postgres"
    // Use a realish but non-connecting config
    cfg.Database.Host = "localhost"
	
    // We try to create a container, even if it has errors we use the struct if possible
    // but NewContainer returns (nil, err) on fail. 
    // Let's manually create a dummy container for the controller.
	container := &di.Container{}
	ctrl := NewHealthController(container)

	t.Run("Ping", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		ctrl.Ping(c)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ok")
	})

	t.Run("Health", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		ctrl.Health(c)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
