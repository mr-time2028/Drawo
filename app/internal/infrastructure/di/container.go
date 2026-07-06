// Package di wires the application's dependencies together.
package di

import (
	"context"

	"drawo/config"
	"drawo/internal/core/ports"
	"drawo/internal/infrastructure/cache"
	"drawo/internal/infrastructure/database"
	"drawo/internal/infrastructure/websocket"
	"drawo/internal/repositories"
	"drawo/internal/services"
	"drawo/pkg/logger"
)

// Container holds all application dependencies.
type Container struct {
	Config   config.Config
	DB       *database.Connection
	Cache    ports.CacheRepository
	Hub      *websocket.Hub
	Services Services
}

// Services groups all application services for easy access.
type Services struct {
	Auth ports.AuthService
	User ports.UserService
	Room ports.RoomService
}

// NewContainer builds the full dependency graph.
func NewContainer(cfg config.Config) (*Container, error) {
	logger.Init(cfg.Log)

	dbConn, err := database.NewConnection(cfg.Database)
	if err != nil {
		return nil, err
	}

	cacheClient, err := cache.NewClient(cfg.Cache)
	if err != nil {
		return nil, err
	}

	// Persistent repositories receive the relational GORM DB handle.
	userRepo := repositories.NewUserRepo(dbConn.DB)
	_ = repositories.NewProfileRepo(dbConn.DB)
	_ = repositories.NewFriendRepo(dbConn.DB)
	_ = repositories.NewGameHistoryRepo(dbConn.DB)
	_ = repositories.NewStatsRepo(dbConn.DB)

	// Ephemeral room discovery repository receives the non-relational CacheRepository.
	roomRepo := repositories.NewRoomRepo(cacheClient)

	// WebSocket Hub manages active room goroutines locally and coordinates via roomRepo.
	hub := websocket.NewHub(roomRepo)

	// Suppress unused warning while placeholder auth/user services are wired in later phases.
	_ = userRepo

	return &Container{
		Config: cfg,
		DB:     dbConn,
		Cache:  cacheClient,
		Hub:    hub,
		Services: Services{
			Auth: services.NewAuthService(),
			User: services.NewUserService(),
			Room: services.NewRoomService(roomRepo),
		},
	}, nil
}

// Health returns the status of all infrastructure components.
func (c *Container) Health() map[string]error {
	ctx := context.Background()
	return map[string]error{
		"database": c.DB.Health(ctx),
		"cache":    c.Cache.Health(ctx),
	}
}
