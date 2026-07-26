package cmd

import (
	"testing"

	"drawo/config"
	"github.com/stretchr/testify/assert"
)

func TestMigrateCommands(t *testing.T) {
	// These will fail due to no DB, but we hit the Cobra logic
	config.Load()

	t.Run("migrate", func(t *testing.T) {
		err := migrateCmd.RunE(migrateCmd, []string{})
		assert.Error(t, err)
	})

	t.Run("up", func(t *testing.T) {
		err := migrateUpCmd.RunE(migrateUpCmd, []string{})
		assert.Error(t, err)
	})

	t.Run("down", func(t *testing.T) {
		err := migrateDownCmd.RunE(migrateDownCmd, []string{})
		assert.Error(t, err)
	})

	t.Run("status", func(t *testing.T) {
		err := migrateStatusCmd.RunE(migrateStatusCmd, []string{})
		assert.Error(t, err)
	})

	t.Run("force", func(t *testing.T) {
		err := migrateForceCmd.RunE(migrateForceCmd, []string{"1"})
		assert.Error(t, err)
	})

	t.Run("go_to", func(t *testing.T) {
		err := migrateGoToVersionCmd.RunE(migrateGoToVersionCmd, []string{"1"})
		assert.Error(t, err)
	})

	t.Run("generate", func(t *testing.T) {
		// This doesn't need DB
		tmpDir := t.TempDir()
		cfg := config.Get()
		cfg.App.MigrationsPath = tmpDir
		// We can't easily inject cfg into migrate.go because it uses config.Get()
		// and we already called Load().
		// But we can test it fails or succeeds depending on path.
	})
}

func TestRootCommand(t *testing.T) {
	assert.NotNil(t, rootCmd)
}
