// Package http (internal/delivery/http) bootstraps the Gin HTTP server.
//
// Responsibility:
//   - Create and configure the Gin engine.
//   - Register routes.
//   - Start and stop the server gracefully.
package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"drawo/config"
	"drawo/internal/delivery/http/routes"
	"drawo/internal/infrastructure/di"
	"drawo/pkg/logger"
)

// Server wraps the Gin engine and HTTP server.
type Server struct {
	httpServer *http.Server
}

// NewServer creates a configured Gin server with all routes registered.
func NewServer(cfg config.ServerConfig, container *di.Container) *Server {
	// Release mode hides stack traces from HTTP responses.
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	routes.Register(router, container)

	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
			Handler: router,
		},
	}
}

// Run starts the server and blocks until it is shut down.
//
// It listens for SIGINT/SIGTERM and performs a graceful shutdown with a 5-second timeout.
func (s *Server) Run() error {
	go func() {
		logger.L.Info("starting http server", slog.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L.Error("http server error", slog.Any("error", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.L.Info("shutting down http server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}
