package serve

import (
	"drawo/internal/routes"
	"drawo/pkg/config"
	"drawo/pkg/database"
	"drawo/pkg/static"
	"drawo/pkg/websocket"
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

	// register routes
	routes.RegisterRoutes(router)

	// load static files
	static.LoadStatic(GetRouter())

	// start ws hub
	websocket.NewHub()
	hub := websocket.GetHub()
	go hub.Run()

	// start application
	err := GetRouter().Run(fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port))
	if err != nil {
		log.Fatal("Failed to serve the application")
	}
}
