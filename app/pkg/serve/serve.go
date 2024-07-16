package serve

import (
	"drawo/pkg/config"
	"drawo/pkg/database"
	"drawo/pkg/static"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
)

var router *gin.Engine

func Init() {
	router = gin.Default()
}

func GetRouter() *gin.Engine {
	return router
}

func Serve() {
	// initial configs
	config.SetConfig()
	cfg := config.GetConfig()

	// connect to the database
	database.Connect()

	// initial router
	Init()

	// load static files
	static.LoadStatic(GetRouter())

	// start application
	err := GetRouter().Run(fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port))
	if err != nil {
		log.Fatal("Failed to serve the application")
	}
}
