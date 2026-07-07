package database

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drawo/config"
)

func TestMigrationManager_GetMigrationURL(t *testing.T) {
	cfg := config.Config{
		Database: config.DatabaseConfig{
			User:     "user",
			Password: "pass",
			Host:     "localhost",
			Port:     "5432",
			Name:     "db",
			SSLMode:  "disable",
		},
	}
	mgr := NewMigrationManager(cfg)
	expected := "postgres://user:pass@localhost:5432/db?sslmode=disable"
	assert.Equal(t, expected, mgr.GetMigrationURL())
}

func TestMigrationManager_Validate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "migrations-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := config.Config{
		App: config.AppConfig{
			MigrationsPath: tempDir,
		},
	}
	mgr := NewMigrationManager(cfg)

	// Case 1: Empty directory
	err = mgr.Validate()
	assert.NoError(t, err)

	// Case 2: Matching up/down, not empty
	upFile := filepath.Join(tempDir, "001_test.up.sql")
	downFile := filepath.Join(tempDir, "001_test.down.sql")
	require.NoError(t, os.WriteFile(upFile, []byte("SELECT 1;"), 0644))
	require.NoError(t, os.WriteFile(downFile, []byte("SELECT 2;"), 0644))
	err = mgr.Validate()
	assert.NoError(t, err)

	// Case 3: Missing down file
	upFile2 := filepath.Join(tempDir, "002_test.up.sql")
	require.NoError(t, os.WriteFile(upFile2, []byte("SELECT 3;"), 0644))
	err = mgr.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is missing a matching .down.sql file")

	// Case 4: Empty up file
	require.NoError(t, os.Remove(upFile2))
	require.NoError(t, os.WriteFile(upFile2, []byte("   "), 0644))
	downFile2 := filepath.Join(tempDir, "002_test.down.sql")
	require.NoError(t, os.WriteFile(downFile2, []byte("SELECT 4;"), 0644))
	err = mgr.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is empty")
    
    // Case 5: Dir doesn't exist
    mgr.cfg.App.MigrationsPath = "/non/existent/path"
    assert.Error(t, mgr.Validate())
}

func TestMigrationManager_GenerateMigration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "migrations-gen-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := config.Config{
		App: config.AppConfig{
			MigrationsPath: tempDir,
		},
	}
	mgr := NewMigrationManager(cfg)

	// Generate first migration
	err = mgr.GenerateMigration("user", "init")
	assert.NoError(t, err)

	upFile := filepath.Join(tempDir, "000001_user_init.up.sql")
	downFile := filepath.Join(tempDir, "000001_user_init.down.sql")
	assert.FileExists(t, upFile)
	assert.FileExists(t, downFile)

	// Generate second migration
	err = mgr.GenerateMigration("auth", "login")
	assert.NoError(t, err)

	upFile2 := filepath.Join(tempDir, "000002_auth_login.up.sql")
	assert.FileExists(t, upFile2)

	// Attempt duplicate
	err = mgr.GenerateMigration("user", "init")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestMigrationManager_Helpers(t *testing.T) {
	mgr := &MigrationManager{}
	
	// Friendly Name
	assert.Equal(t, "user_init", mgr.getFriendlyName("000001_user_init.up.sql"))
	assert.Equal(t, "plain", mgr.getFriendlyName("plain"))
	assert.Equal(t, "", mgr.getFriendlyName(""))

	// File listing and applied logic needs a real dir
	tempDir, _ := os.MkdirTemp("", "migrations-helper-test")
	defer os.RemoveAll(tempDir)
	mgr.cfg.App.MigrationsPath = tempDir

	// Mix valid and invalid files
	os.WriteFile(filepath.Join(tempDir, "000001_test.up.sql"), []byte("sql"), 0644)
    os.WriteFile(filepath.Join(tempDir, "000001_test.down.sql"), []byte("sql"), 0644)
	os.WriteFile(filepath.Join(tempDir, "not_a_migration.sql"), []byte("sql"), 0644)
    os.WriteFile(filepath.Join(tempDir, "bad_version_test.up.sql"), []byte("sql"), 0644)
	os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	
	files, err := mgr.getAllMigrationFiles(tempDir)
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, uint(1), files[0].Version)

	applied, err := mgr.getAppliedMigrations(0, 1)
	assert.NoError(t, err)
	assert.Len(t, applied, 1)

	fileName, err := mgr.getMigrationFileName(tempDir, 1)
	assert.NoError(t, err)
	assert.Contains(t, fileName, "000001")
    
    _, err = mgr.getMigrationFileName(tempDir, 99)
    assert.Error(t, err)
}

func TestMigrationManager_Commands(t *testing.T) {
    // We can't run real migrate up/down because it needs Postgres, 
    // but we can test that they return error when DB is missing.
    cfg := config.Get()
    mgr := NewMigrationManager(cfg)
    
    assert.Error(t, mgr.Migrate())
    assert.Error(t, mgr.MigrateUp())
    assert.Error(t, mgr.MigrateDown(false))
    assert.Error(t, mgr.MigrateDown(true))
    assert.Error(t, mgr.Status())
    assert.Error(t, mgr.Force(1))
    assert.Error(t, mgr.GoToVersion(1))
    
    // Test NewMigrator fail (invalid path)
    mgr.cfg.App.MigrationsPath = "\x00"
    _, err := mgr.NewMigrator()
    assert.Error(t, err)
}
