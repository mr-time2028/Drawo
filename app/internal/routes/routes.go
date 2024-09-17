package routes

import (
	"drawo/internal/middlewares"
	roomRoutes "drawo/internal/modules/room/routes"
	tokenRoutes "drawo/internal/modules/token/routes"
	userRoutes "drawo/internal/modules/user/routes"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.Use(middlewares.CORSMiddleware())
	router.HandleMethodNotAllowed = true

	roomRoutes.Routes(router)
	tokenRoutes.Routes(router)
	userRoutes.Routes(router)
}
