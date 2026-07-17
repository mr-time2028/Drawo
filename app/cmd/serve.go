package cmd

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
	"github.com/spf13/cobra"

	"drawo/config"
	httpRoutes "drawo/internal/delivery/http/routes"
	wsRoutes "drawo/internal/delivery/websocket"
	"drawo/internal/infrastructure/di"
	"drawo/pkg/logger"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Drawo HTTP server",
	Long:  "Loads configuration, connects to infrastructure, and starts the API server.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return serve()
	},
}

func serve() error {
	cfg := config.Get()

	// Initialize the dependency container. This also opens DB and cache store.
	container, err := di.NewContainer(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if container.Cache != nil {
			_ = container.Cache.Close()
		}
	}()

	logger.L.Info("drawo server initialized",
		slog.String("app", cfg.App.Name),
		slog.String("env", cfg.Log.Level),
	)

	// Start the HTTP server. This blocks until shutdown.
	server := newServer(cfg.Server, container)
	return server.run()
}

// server wraps the Gin engine and net/http server for the `serve` command.
//
// This lives in cmd because starting/stopping the HTTP process is application
// bootstrap behavior. The delivery layer still owns controllers, middleware,
// and route registration; cmd only wires those adapters into an executable
// server process.
type server struct {
	httpServer *http.Server
}

// newServer creates a configured Gin server with all routes registered.
func newServer(cfg config.ServerConfig, container *di.Container) *server {
	// Release mode hides stack traces from HTTP responses.
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	httpRoutes.Register(router, container)
	wsRoutes.Register(router, container)

	return &server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
			Handler: router,
		},
	}
}

// run starts the server and blocks until it is shut down.
//
// It listens for SIGINT/SIGTERM and performs a graceful shutdown with a
// 5-second timeout.
func (s *server) run() error {
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
