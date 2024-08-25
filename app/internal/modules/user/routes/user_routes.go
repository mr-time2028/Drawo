package routes

import (
	"drawo/internal/modules/user/controllers"
	"github.com/gin-gonic/gin"
)

func Routes(router *gin.Engine) {
	userController := controllers.New()

	router.POST("/users/register/", userController.Register)
	router.POST("/users/login/", userController.Login)
}
