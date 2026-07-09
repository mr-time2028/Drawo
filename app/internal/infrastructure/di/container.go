// Package di wires the application's dependencies together.
// This is the "Glue" of the project where infrastructure meets logic.
package di

import (
	"context"
	"drawo/config"
	"drawo/internal/core/ports/repositories"
	"drawo/internal/core/ports/services"
	"drawo/internal/infrastructure/cache"
	"drawo/internal/infrastructure/database"
	"drawo/internal/infrastructure/websocket"
	"drawo/pkg/logger"
)

// Container holds all application dependencies.
type Container struct {
	Config   config.Config
	DB       *database.Connection
	Cache    repositories.CacheRepository
	Sessions repositories.SessionRepository
	Limiter  services.RateLimiter
	Hub      *websocket.Hub
	Services Services
}

// Services groups all high-level application services.
type Services struct {
	Auth services.AuthService
	User services.UserService
	Room services.RoomService
}

// NewContainer builds the full dependency graph for the application.
func NewContainer(cfg config.Config) (*Container, error) {
	// Initialize logging first so all subsequent errors are tracked correctly.
	logger.Init(cfg.Log)

	// Establish relational database connectivity.
	dbConn, err := database.NewConnection(cfg.Database)
	if err != nil {
		return nil, err
	}

	// Establish non-relational (cache/redis) connectivity.
	cacheClient, err := cache.NewClient(cfg.Cache)
	if err != nil {
		return nil, err
	}

	// 1. Initialize Repositories (Persist data)
	userRepo := repositories.NewUserRepo(dbConn.DB)
	profileRepo := repositories.NewProfileRepo(dbConn.DB)
	_ = repositories.NewFriendshipRepo(dbConn.DB)
	_ = repositories.NewFriendRequestRepo(dbConn.DB)
	_ = repositories.NewGameHistoryRepo(dbConn.DB)
	_ = repositories.NewReportRepo(dbConn.DB)
	_ = repositories.NewAchievementRepo(dbConn.DB)
	_ = repositories.NewPlayerStatisticRepo(dbConn.DB)
	_ = repositories.NewUserSettingsRepo(dbConn.DB)

	// 2. Initialize Specialized Cache Repositories
	sessionRepo := repositories.NewSessionRepo(cacheClient)
	roomRepo := repositories.NewRoomRepo(cacheClient)

	// 3. Initialize Services (Business Logic)
	rateLimiter := services.NewRateLimiter(cacheClient)
	
	authSvc := services.NewAuthService(cfg, userRepo, profileRepo, sessionRepo, rateLimiter)
	userSvc := services.NewUserService() 
	roomSvc := services.NewRoomService(roomRepo)
	
	// WebSocket Hub manages active room goroutines locally and coordinates discovery via Redis.
	hub := websocket.NewHub(roomRepo)

	return &Container{
		Config:   cfg,
		DB:       dbConn,
		Cache:    cacheClient,
		Sessions: sessionRepo,
		Limiter:  rateLimiter,
		Hub:      hub,
		Services: Services{
			Auth: authSvc,
			User: userSvc,
			Room: roomSvc,
		},
	}, nil
}

// Health returns the status of all critical infrastructure components.
func (c *Container) Health() map[string]error {
	ctx := context.Background()
	res := make(map[string]error)
	if c.DB != nil {
		res["database"] = c.DB.Health(ctx)
	}
	if c.Cache != nil {
		res["cache"] = c.Cache.Health(ctx)
	}
	return res
}
