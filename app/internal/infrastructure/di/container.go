// Package di wires the application's dependencies together.
//
// Responsibility:
//   - Construct infrastructure components (DB, Redis, logger).
//   - Construct repositories and services.
//   - Expose a single container that HTTP handlers can depend on.
//
// Why dependency injection?
//   Without DI, services create their own repositories with `repository.New()`.
//   That makes unit testing impossible and hides dependencies. With a container,
//   every dependency is explicit and can be replaced (e.g., with a mock).
//
// Pattern:
//   This is a simple manual DI container. For a larger project we could use
//   wire or fx, but manual wiring is easier to read and debug while learning Go.
package di

import (
	"context"

	"drawo/config"
	"drawo/internal/core/ports"
	"drawo/internal/infrastructure/database"
	"drawo/internal/infrastructure/redis"
	"drawo/internal/repositories"
	"drawo/internal/services"
	"drawo/pkg/logger"
)

// Container holds all application dependencies.
//
// Controllers receive *Container and pull only what they need.
// This keeps the constructor signature small and avoids "constructor hell".
type Container struct {
	Config   config.Config
	DB       *database.Connection
	Redis    *redis.Client
	Services Services
}

// Services groups all application services for easy access.
type Services struct {
	Auth ports.AuthService
	User ports.UserService
}

// NewContainer builds the full dependency graph.
func NewContainer(cfg config.Config) (*Container, error) {
	logger.Init(cfg.Log)

	dbConn, err := database.NewConnection(cfg.Database)
	if err != nil {
		return nil, err
	}

	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		return nil, err
	}

	// Repositories receive the GORM DB handle.
	userRepo := repositories.NewUserRepo(dbConn.DB)
	_ = repositories.NewProfileRepo(dbConn.DB)
	_ = repositories.NewRoomRepo(dbConn.DB)

	// Services receive repositories. Currently they are placeholders.
	// In later phases we will inject real repositories into real services.
	_ = userRepo

	return &Container{
		Config: cfg,
		DB:     dbConn,
		Redis:  redisClient,
		Services: Services{
			Auth: services.NewAuthService(),
			User: services.NewUserService(),
		},
	}, nil
}

// Health returns the status of all infrastructure components.
func (c *Container) Health() map[string]error {
	ctx := context.Background()
	return map[string]error{
		"database": c.DB.Health(ctx),
		"redis":    c.Redis.Health(ctx),
	}
}
