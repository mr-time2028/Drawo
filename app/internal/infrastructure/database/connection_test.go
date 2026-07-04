package database

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"drawo/config"
)

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
