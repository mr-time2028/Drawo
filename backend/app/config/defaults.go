package config

import (
	"time"

	"github.com/spf13/viper"
)

// setDefaults registers default values for every configuration key.
//
// Defaults are overridden by (in order of precedence):
//  1. Environment variables (e.g., DRAWO_SERVER_PORT)
//  2. .env file
//  3. Explicit values set in tests before calling Load()
//
// We use the mapstructure keys here because Viper binds env vars to those keys.
func setDefaults() {
	// Application
	viper.SetDefault("app.name", "Drawo")
	viper.SetDefault("app.domain", "http://localhost:8080")
	viper.SetDefault("app.secretKey", "change-me-in-production")
	viper.SetDefault("app.migrationsPath", "migrations")
	viper.SetDefault("app.modules", []string{"user", "auth", "room", "game", "admin"})
	viper.SetDefault("app.storage.driver", "local")
	viper.SetDefault("app.storage.endpoint", "localhost:9000")
	viper.SetDefault("app.storage.accessKey", "admin")
	viper.SetDefault("app.storage.secretKey", "change-me-minio-password")
	viper.SetDefault("app.storage.useSSL", false)
	viper.SetDefault("app.storage.bucketName", "drawo")
	viper.SetDefault("app.storage.region", "us-east-1")
	viper.SetDefault("app.storage.uploadDir", "uploads")

	// Server
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", "8080")

	// Database
	viper.SetDefault("database.driver", "postgres")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("database.name", "drawo")
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "123456")
	viper.SetDefault("database.sslMode", "disable")

	// Cache (abstracted non-relational storage)
	viper.SetDefault("cache.driver", "redis")
	viper.SetDefault("cache.host", "localhost")
	viper.SetDefault("cache.port", "6379")
	viper.SetDefault("cache.password", "")
	viper.SetDefault("cache.db", 0)

	// Redis (backward compatibility)
	viper.SetDefault("redis.driver", "redis")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// Auth
	viper.SetDefault("auth.issuer", "drawo")
	viper.SetDefault("auth.audience", "drawo")
	viper.SetDefault("auth.accessTokenExpiry", 15*time.Minute)
	viper.SetDefault("auth.refreshTokenExpiry", 7*24*time.Hour)
	viper.SetDefault("auth.maxLoginAttempts", 5)
	viper.SetDefault("auth.loginLockoutDuration", time.Minute)
	viper.SetDefault("auth.refreshTokenFamilySize", 10)

	// Log
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
}
