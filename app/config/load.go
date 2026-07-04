// Package config (continued): loading logic.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Load reads configuration from defaults, a .env file, and environment variables.
//
// Priority (highest to lowest):
//   1. Environment variables (e.g., DRAWO_SERVER_PORT)
//   2. .env file in the working directory
//   3. Built-in defaults
//
// The .env file is optional in production (env vars can be injected directly),
// but required in development so the app knows where Postgres and Redis are.
func Load() error {
	setDefaults()

	// Load .env file if it exists. godotenv does not error when the file is missing,
	// which lets the same binary run in containers that use only env vars.
	_ = godotenv.Load(".env")

	// Allow environment variables to override everything.
	viper.SetEnvPrefix("DRAWO")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.Unmarshal(&configurations); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
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
