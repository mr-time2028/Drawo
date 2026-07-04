package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetEnv clears all DRAWO_* environment variables so tests are deterministic.
func resetEnv(t *testing.T) {
	t.Helper()
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(parts[0], "DRAWO_") {
			_ = os.Unsetenv(parts[0])
		}
	}
}

// TestLoadDefaults verifies that default configuration values are applied
// when no .env file or environment variables are provided.
func TestLoadDefaults(t *testing.T) {
	resetEnv(t)

	err := Load()
	require.NoError(t, err)

	cfg := Get()
	assert.Equal(t, "Drawo", cfg.App.Name)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "drawo", cfg.Database.Name)
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 15*time.Minute, cfg.Auth.AccessTokenExpiry)
}

// TestLoadEnvOverride verifies that environment variables override defaults.
func TestLoadEnvOverride(t *testing.T) {
	resetEnv(t)
	t.Setenv("DRAWO_SERVER_PORT", "9999")
	t.Setenv("DRAWO_APP_NAME", "TestDrawo")

	err := Load()
	require.NoError(t, err)

	cfg := Get()
	assert.Equal(t, "9999", cfg.Server.Port)
	assert.Equal(t, "TestDrawo", cfg.App.Name)
}

// TestLoadDotEnv verifies that a .env file is loaded and overrides defaults.
func TestLoadDotEnv(t *testing.T) {
	resetEnv(t)

	// Create a temporary .env file.
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	content := `DRAWO_SERVER_PORT=7777
DRAWO_LOG_LEVEL=debug`
	require.NoError(t, os.WriteFile(envPath, []byte(content), 0600))

	// Change to the temp directory so godotenv.Load(".env") finds our file.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	err = Load()
	require.NoError(t, err)

	cfg := Get()
	assert.Equal(t, "7777", cfg.Server.Port)
	assert.Equal(t, "debug", cfg.Log.Level)
}
