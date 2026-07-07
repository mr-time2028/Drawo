package routes

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"drawo/internal/infrastructure/di"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	container := &di.Container{}
	
	Register(router, container)
	
	// Verify some routes exist
	routes := router.Routes()
	assert.NotEmpty(t, routes)
}
