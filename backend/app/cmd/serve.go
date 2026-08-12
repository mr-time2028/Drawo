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
type server struct {
	httpServer *http.Server
}

// newServer creates a configured Gin server with all routes registered.
func newServer(cfg config.ServerConfig, container *di.Container) *server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	httpRoutes.Register(router, container)
	wsRoutes.Register(router, container)

	return &server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
			Handler: router,
			// Generous timeouts so WebSocket upgrades don't hit the default
			// zero-value (which causes i/o timeout errors like
			// "read tcp [::1]:8080->[::1]:xxxxx: i/o timeout" on slow
			// connections or during the auth handshake).
			ReadHeaderTimeout: 20 * time.Second,
			ReadTimeout:       0, // 0 = no deadline; long-lived WS connections require this.
			WriteTimeout:      0,
			IdleTimeout:       120 * time.Second,
		},
	}
}

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
