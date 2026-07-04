package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"

	"drawo/config"
	"drawo/internal/delivery/http"
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
	// Load configuration from env vars and optional config file.
	if err := config.Load(); err != nil {
		return err
	}
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
	server := http.NewServer(cfg.Server, container)
	return server.Run()
}
