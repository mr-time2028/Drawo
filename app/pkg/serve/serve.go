package serve

import (
	"drawo/internal/routes"
	"drawo/pkg/config"
	"drawo/pkg/database"
	"drawo/pkg/routing"
	"drawo/pkg/static"
	"drawo/pkg/websocket"
	"fmt"
	"log"
)

func Serve() {
	// initial configs
	config.SetConfig()
	cfg := config.Get()

	// connect to the database
	database.Connect()

	// initial router
	routing.Init()
	router := routing.Get()

	// register routes
	routes.RegisterRoutes(router)

	// load static files
	static.LoadStatic(router)

	// start ws hub
	websocket.NewHub()
	hub := websocket.GetHub()
	go hub.Run()

	// start application
	err := router.Run(fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port))
	if err != nil {
		log.Fatal("Failed to serve the application")
	}
}
