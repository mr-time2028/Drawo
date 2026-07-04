// Package database provides PostgreSQL connectivity for the application.
//
// Responsibility:
//   - Open and configure a GORM connection.
//   - Implement ports.HealthReporter so the server can report DB health.
//   - Provide a getter for repositories to share the same *gorm.DB.
//
// Why a package-level DB variable?
//   GORM's *gorm.DB is safe for concurrent use and designed to be reused.
//   Repositories receive it through dependency injection instead of importing
//   this package directly.
package database

import (
	"context"
	"fmt"
	"log/slog"

	"drawo/config"
	"drawo/internal/core/ports"
	"drawo/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connection is the shared GORM database handle.
type Connection struct {
	DB *gorm.DB
}

// NewConnection opens a PostgreSQL connection using the provided config.
func NewConnection(cfg config.DatabaseConfig) (*Connection, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		cfg.Host,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.Port,
		cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql db: %w", err)
	}

	// Sanity checks.
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	logger.L.Info("connected to PostgreSQL", slog.String("host", cfg.Host), slog.String("database", cfg.Name))

	return &Connection{DB: db}, nil
}

// Health verifies the database is reachable.
func (c *Connection) Health(ctx context.Context) error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Compile-time check: Connection implements ports.HealthReporter.
var _ ports.HealthReporter = (*Connection)(nil)
