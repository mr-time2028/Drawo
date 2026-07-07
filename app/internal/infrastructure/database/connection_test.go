package database

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"drawo/config"
)

type failDialector struct {
    gorm.Dialector
}
func (d *failDialector) Initialize(db *gorm.DB) error {
    return errors.New("open error")
}

func TestNewConnection_UnsupportedDriver(t *testing.T) {
	cfg := config.DatabaseConfig{
		Driver: "nonexistent-driver",
	}

	conn, err := NewConnection(cfg)
	assert.Nil(t, conn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database driver")
}

func TestRegisterDriver(t *testing.T) {
	mockErr := errors.New("mock dialector created")
	RegisterDriver("mockdb", func(cfg config.DatabaseConfig) (gorm.Dialector, error) {
		return nil, mockErr
	})

	cfg := config.DatabaseConfig{
		Driver: "mockdb",
	}

	conn, err := NewConnection(cfg)
	assert.Nil(t, conn)
	require.Error(t, err)
	assert.ErrorIs(t, err, mockErr)
}

func TestNewConnection_Success(t *testing.T) {
	RegisterDriver("test_sqlite", func(cfg config.DatabaseConfig) (gorm.Dialector, error) {
		return sqlite.Open(":memory:"), nil
	})

	cfg := config.DatabaseConfig{
		Driver: "test_sqlite",
	}

	conn, err := NewConnection(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, conn.DB)

	// Test Health Success
	assert.NoError(t, conn.Health(context.Background()))
}

func TestNewConnection_Failures(t *testing.T) {
    // Open Fail
	RegisterDriver("fail_open", func(cfg config.DatabaseConfig) (gorm.Dialector, error) {
		return &failDialector{}, nil
	})
    _, err := NewConnection(config.DatabaseConfig{Driver: "fail_open"})
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "open database")
    
    // Default to postgres if empty
    _, err = NewConnection(config.DatabaseConfig{Driver: ""})
    assert.Error(t, err) // Fails because postgres not running
}

func TestConnection_Health_Fail(t *testing.T) {
	// Case: Not initialized
	conn := &Connection{DB: nil}
	err := conn.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
    
    // Case: Underlyng DB error (use a closed DB)
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    sqlDB, _ := db.DB()
    sqlDB.Close()
    conn2 := &Connection{DB: db}
    assert.Error(t, conn2.Health(context.Background()))
}
