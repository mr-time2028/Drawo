package routes

import (
	"drawo/internal/middlewares"
	UserRoutes "drawo/internal/modules/user/routes"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.Use(middlewares.CORSMiddleware())
	router.HandleMethodNotAllowed = true

	UserRoutes.Routes(router)
}
