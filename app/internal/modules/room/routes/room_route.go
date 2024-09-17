package routes

import (
	"drawo/internal/middlewares"
	"drawo/internal/modules/room/controllers"
	"github.com/gin-gonic/gin"
)

func Routes(router *gin.Engine) {
	roomController := controllers.New()

	authGroup := router.Group("/rooms", middlewares.AuthMiddleware())
	authGroup.POST("/create_private_room", roomController.CreatePrivateRoom)
}
