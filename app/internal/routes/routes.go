package routes

import (
	"drawo/internal/middlewares"
	tokenRoutes "drawo/internal/modules/token/routes"
	userRoutes "drawo/internal/modules/user/routes"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.Use(middlewares.CORSMiddleware())
	router.HandleMethodNotAllowed = true

	userRoutes.Routes(router)
	tokenRoutes.Routes(router)
}
