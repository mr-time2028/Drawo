package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"drawo/internal/infrastructure/di"
)

func TestHealthController(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("Ping", func(t *testing.T) {
        container := &di.Container{}
	    ctrl := NewHealthController(container)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		ctrl.Ping(c)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ok")
	})

	t.Run("Health_Success", func(t *testing.T) {
        container := &di.Container{}
	    ctrl := NewHealthController(container)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		ctrl.Health(c)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
