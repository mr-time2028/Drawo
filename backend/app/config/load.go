// Package config (continued): loading logic.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Load reads configuration from defaults, a .env file, and environment variables.
//
// Priority (highest to lowest):
//  1. Environment variables (e.g., DRAWO_SERVER_PORT)
//  2. .env file in the repository root, or current working directory
//  3. Built-in defaults
//
// The .env file is optional in production (env vars can be injected directly),
// but required in development so the app knows where backing services are.
func Load() error {
	setDefaults()

	loadDotEnv()

	// Allow environment variables to override everything.
	viper.SetEnvPrefix("DRAWO")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.Unmarshal(&configurations); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	// Backwards compatibility: if cache configuration matches defaults but redis configuration was explicitly set,
	// mirror redis config to cache config.
	if configurations.Cache.Host == "localhost" && configurations.Redis.Host != "localhost" {
		configurations.Cache = CacheConfig(configurations.Redis)
	}

	return nil
}

// MustLoad is like Load but panics on error. Use it in main().
func MustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

// GetEnv returns an environment variable or the given default.
// It is a small helper for values that are not part of the typed Config struct.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loadDotEnv loads local development environment files.
//
// Docker Compose reads the repository-root .env automatically. For local Go
// execution, developers usually run `cd app && go run . serve`; in that case
// the app should still use the single repository-root .env rather than requiring
// a duplicated app/.env file.
func loadDotEnv() {
	// Load .env in the current working directory when present. This supports
	// running from the repository root and keeps existing tests deterministic.
	_ = godotenv.Load(".env")

	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	// When running from /app, also load ../.env from the repository root.
	// Do not walk parents for package tests such as /app/config, otherwise tests
	// would accidentally consume the developer's real root .env.
	if filepath.Base(cwd) == "app" {
		_ = godotenv.Load(filepath.Join("..", ".env"))
	}
}
