// Package database provides relational database connectivity for the application.
//
// Responsibility:
//   - Open and configure a database connection using GORM dialectors based on config.
//   - Implement repositories.HealthReporter so the server can report DB health.
//   - Provide a factory registry so switching between relational databases (PostgreSQL, MySQL, SQLite, etc.)
//     requires minimal code changes and no database-specific logic leaking into repositories or application
//
// Why this architecture?
//
//	The application layer depends on interfaces (UserRepository, etc.).
//	Concrete repositories depend on GORM's *gorm.DB abstraction rather than raw driver connections.
//	By decoupling driver initialization through DialectorFactory, switching databases is purely a configuration step.
package database

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"drawo/config"
	"drawo/internal/core/ports/repositories"
	"drawo/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connection is the shared GORM database handle.
type Connection struct {
	DB *gorm.DB
}

// DialectorFactory creates a gorm.Dialector given a database configuration.
type DialectorFactory func(cfg config.DatabaseConfig) (gorm.Dialector, error)

// dialectorFactories holds registered database driver factories.
var dialectorFactories = map[string]DialectorFactory{
	"postgres": func(cfg config.DatabaseConfig) (gorm.Dialector, error) {
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			cfg.Host,
			cfg.User,
			cfg.Password,
			cfg.Name,
			cfg.Port,
			cfg.SSLMode,
		)
		return postgres.Open(dsn), nil
	},
}

// RegisterDriver registers a custom DialectorFactory for a database driver name
// (e.g., "mysql", "sqlite", "sqlserver"). This allows extending database support
// without modifying existing core infrastructure logic.
func RegisterDriver(name string, factory DialectorFactory) {
	dialectorFactories[strings.ToLower(name)] = factory
}

// NewConnection opens a relational database connection using the configured driver.
func NewConnection(cfg config.DatabaseConfig) (*Connection, error) {
	driver := strings.ToLower(cfg.Driver)
	if driver == "" {
		driver = "postgres"
	}

	factory, ok := dialectorFactories[driver]
	if !ok {
		return nil, fmt.Errorf("unsupported database driver: %q (supported drivers can be registered via RegisterDriver)", cfg.Driver)
	}

	dialector, err := factory(cfg)
	if err != nil {
		return nil, fmt.Errorf("create dialector for driver %s: %w", driver, err)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database (%s): %w", driver, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql db: %w", err)
	}

	// Connection pool configurations.
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database (%s): %w", driver, err)
	}

	logger.L.Info("connected to relational database", slog.String("driver", driver), slog.String("host", cfg.Host), slog.String("database", cfg.Name))

	return &Connection{DB: db}, nil
}

// Health verifies the database is reachable.
func (c *Connection) Health(ctx context.Context) error {
	if c.DB == nil {
		return fmt.Errorf("database connection not initialized")
	}
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Compile-time check: Connection implements repositories.HealthReporter.
var _ repositories.HealthReporter = (*Connection)(nil)
