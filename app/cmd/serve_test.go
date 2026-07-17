package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"drawo/config"
	"drawo/internal/infrastructure/di"
)

func TestNewServer(t *testing.T) {
	cfg := config.ServerConfig{Host: "localhost", Port: "8080"}
	container := &di.Container{}
	server := newServer(cfg, container)

	assert.NotNil(t, server)
	assert.Equal(t, "localhost:8080", server.httpServer.Addr)
}

func TestHTTPServerHealth(t *testing.T) {
	config.Load()
	cfg := config.Get()
	cfg.Database.Driver = "postgres"
	cfg.Cache.Driver = "memory"

	// Full container initialization needs a reachable database. Keep the old
	// behavior: skip this integration-style health test when DB is not available.
	container, _ := di.NewContainer(cfg)
	if container == nil {
		t.Skip("Skipping HTTP test because container could not be initialized without DB")
	}

	server := newServer(config.ServerConfig{Port: "0"}, container)

	req, _ := http.NewRequest("GET", "/health/ping", nil)
	w := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")

	req, _ = http.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(w, req)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable)
}
