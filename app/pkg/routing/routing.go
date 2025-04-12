package routing

import "github.com/gin-gonic/gin"

var router *gin.Engine

func Init() {
	router = gin.Default()
}

func Get() *gin.Engine {
	return router
}
