// Package di wires the application's dependencies together.
package di

import (
	"context"
	"fmt"

	"drawo/config"
	"drawo/internal/core/ports/repositories"
	"drawo/internal/core/ports/services"
	"drawo/internal/infrastructure/cache"
	"drawo/internal/infrastructure/database"
	"drawo/internal/infrastructure/storage"
	"drawo/internal/realtime"
	"drawo/pkg/logger"
)

// Container holds all application dependencies.
type Container struct {
	Config   config.Config
	DB       *database.Connection
	Cache    repositories.CacheRepository
	Sessions repositories.SessionRepository
	OTPs     repositories.OTPRepository
	Limiter  services.RateLimiter
	Hub      *realtime.Hub
	Services Services
}

type Services struct {
	Auth    services.AuthService
	User    services.UserService
	Room    services.RoomService
	Content services.ContentService
	Admin   services.AdminService
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

	// -------------------------------------------------------------------------
	// DYNAMIC STORAGE SWITCHING:
	// -------------------------------------------------------------------------
	// Based on the 'cfg.App.Storage.Driver' value in .env, we choose the provider.
	var storageProvider repositories.FileStorage
	switch cfg.App.Storage.Driver {
	case "minio":
		storageProvider, err = storage.NewMinioProvider(cfg.App.Storage)
		if err != nil {
			return nil, fmt.Errorf("failed to init minio: %w", err)
		}
	case "local":
		storageProvider = storage.NewLocalStorageProvider(cfg.App.Storage)
	default:
		// Fallback to local for safety during development
		storageProvider = storage.NewLocalStorageProvider(cfg.App.Storage)
	}
	// -------------------------------------------------------------------------

	// 1. Initialize Repositories
	userRepo := repositories.NewUserRepo(dbConn.DB)
	profileRepo := repositories.NewProfileRepo(dbConn.DB)
	_ = repositories.NewFriendshipRepo(dbConn.DB)
	_ = repositories.NewFriendRequestRepo(dbConn.DB)
	_ = repositories.NewGameHistoryRepo(dbConn.DB)
	reportRepo := repositories.NewReportRepo(dbConn.DB)
	_ = repositories.NewAchievementRepo(dbConn.DB)
	_ = repositories.NewPlayerStatisticRepo(dbConn.DB)
	_ = repositories.NewUserSettingsRepo(dbConn.DB)
	contentRepo := repositories.NewContentRepo(dbConn.DB)
	adminRepo := repositories.NewAdminRepo(dbConn.DB)
	reputationRepo := repositories.NewReputationRepo(dbConn.DB)

	// 2. Initialize specialized cache-based repositories
	sessionRepo := repositories.NewSessionRepo(cacheClient)
	roomRepo := repositories.NewRoomRepo(cacheClient)
	otpRepo := repositories.NewOTPRepo(cacheClient)

	// 3. Initialize Services
	rateLimiter := services.NewRateLimiter(cacheClient)
	otpSvc := services.NewMockOTPService()
	contentSvc := services.NewContentService(contentRepo, profileRepo, 100)

	authSvc := services.NewAuthService(cfg, userRepo, profileRepo, sessionRepo, rateLimiter)
	userSvc := services.NewUserService(userRepo, profileRepo, otpRepo, otpSvc)
	roomSvc := services.NewRoomService(roomRepo)
	adminSvc := services.NewAdminService(cfg, adminRepo, userRepo, profileRepo, sessionRepo, storageProvider, contentRepo, reportRepo, reputationRepo)

	hub := realtime.NewHubWithDependencies(roomRepo, contentRepo, profileRepo, reputationRepo)

	return &Container{
		Config:   cfg,
		DB:       dbConn,
		Cache:    cacheClient,
		Sessions: sessionRepo,
		OTPs:     otpRepo,
		Limiter:  rateLimiter,
		Hub:      hub,
		Services: Services{
			Auth:    authSvc,
			User:    userSvc,
			Room:    roomSvc,
			Content: contentSvc,
			Admin:   adminSvc,
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
