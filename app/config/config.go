// Package config holds the application configuration structures and the global accessor.
//
// Responsibility:
//   - Define strongly-typed configuration structs.
//   - Provide a thread-safe getter for the loaded configuration.
//
// Why this design?
//   We load configuration once at startup and treat it as read-only afterwards.
//   A package-level variable is sufficient because the config is immutable after
//   initialization, avoiding the need to pass a Config object through every call.
//   For testing, Load() can be called again with different environment variables.
package config

import "time"

// Config is the root configuration container for the entire application.
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Cache    CacheConfig    `mapstructure:"cache"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Log      LogConfig      `mapstructure:"log"`
}

// AppConfig contains application-level settings.
type AppConfig struct {
	Name           string   `mapstructure:"name"`
	Domain         string   `mapstructure:"domain"`
	SecretKey      string   `mapstructure:"secretKey"`
	MigrationsPath string   `mapstructure:"migrationsPath"`
	Modules        []string `mapstructure:"modules"`
	Storage        StorageConfig `mapstructure:"storage"`
}

// StorageConfig handles the dynamic configuration for file storage (MinIO/S3/Local).
type StorageConfig struct {
	Driver          string `mapstructure:"driver"` // "minio", "s3", or "local"
	Endpoint        string `mapstructure:"endpoint"`
	AccessKey       string `mapstructure:"accessKey"`
	SecretKey       string `mapstructure:"secretKey"`
	UseSSL          bool   `mapstructure:"useSSL"`
	BucketName      string `mapstructure:"bucketName"`
	Region          string `mapstructure:"region"`
	UploadDirectory string `mapstructure:"uploadDir"` // Used for local driver
}

// ServerConfig contains the HTTP server binding.
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

// DatabaseConfig contains relational database connection details.
type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Name     string `mapstructure:"name"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"sslMode"`
}

// CacheConfig contains non-relational database / caching layer connection details.
type CacheConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// RedisConfig contains Redis connection details for backward compatibility.
type RedisConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// AuthConfig contains JWT and security settings.
type AuthConfig struct {
	Issuer                 string        `mapstructure:"issuer"`
	Audience               string        `mapstructure:"audience"`
	AccessTokenExpiry      time.Duration `mapstructure:"accessTokenExpiry"`
	RefreshTokenExpiry     time.Duration `mapstructure:"refreshTokenExpiry"`
	MaxLoginAttempts       int           `mapstructure:"maxLoginAttempts"`
	LoginLockoutDuration   time.Duration `mapstructure:"loginLockoutDuration"`
	RefreshTokenFamilySize int           `mapstructure:"refreshTokenFamilySize"`
}

// LogConfig contains logging settings.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// configurations is the loaded, immutable config instance.
var configurations Config

// Get returns the loaded configuration.
// It panics if called before Load() because running without config is unsafe.
func Get() Config {
	return configurations
}
