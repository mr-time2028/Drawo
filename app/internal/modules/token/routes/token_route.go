package routes

import (
	"drawo/internal/modules/token/controllers"
	"github.com/gin-gonic/gin"
)

func Routes(router *gin.Engine) {
	tokenController := controllers.New()

	router.POST("/tokens/get_access_token/", tokenController.GetAccessTokenByRefreshToken)
	router.POST("/tokens/verify_access_token/", tokenController.VerifyAccessToken)
}
