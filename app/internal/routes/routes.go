package routes

import (
	UserRoutes "drawo/internal/modules/user/routes"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	UserRoutes.Routes(router)
}
