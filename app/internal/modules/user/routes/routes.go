package routes

import (
	"drawo/internal/modules/user/controllers"
	"github.com/gin-gonic/gin"
)

func Routes(router *gin.Engine) {
	_ = controllers.New()
}
