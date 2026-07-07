package http

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"drawo/config"
	"drawo/internal/infrastructure/di"
)

func TestNewServer(t *testing.T) {
	cfg := config.ServerConfig{Host: "localhost", Port: "8080"}
	container := &di.Container{}
	server := NewServer(cfg, container)
	
	assert.NotNil(t, server)
	assert.Equal(t, "localhost:8080", server.httpServer.Addr)
}
