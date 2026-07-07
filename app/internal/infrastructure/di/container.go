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

type Container struct {
	Config   config.Config
	DB       *database.Connection
	Cache    repositories.CacheRepository
	Hub      *websocket.Hub
	Services Services
}

type Services struct {
	Auth services.AuthService
	User services.UserService
	Room services.RoomService
}

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

	_ = repositories.NewUserRepo(dbConn.DB)
	_ = repositories.NewProfileRepo(dbConn.DB)
	_ = repositories.NewFriendshipRepo(dbConn.DB)
	_ = repositories.NewFriendRequestRepo(dbConn.DB)
	_ = repositories.NewGameHistoryRepo(dbConn.DB)
	_ = repositories.NewReportRepo(dbConn.DB)
	_ = repositories.NewAchievementRepo(dbConn.DB)
	_ = repositories.NewPlayerStatisticRepo(dbConn.DB)
	_ = repositories.NewUserSettingsRepo(dbConn.DB)

	roomRepo := repositories.NewRoomRepo(cacheClient)
	hub := websocket.NewHub(roomRepo)

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
