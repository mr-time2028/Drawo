package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"drawo/config"
	"drawo/internal/infrastructure/database"
)

var allFlag bool

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply all pending migrations to the database",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		mgr := database.NewMigrationManager(cfg)
		return mgr.Migrate()
	},
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply only the last pending migration to the database",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		mgr := database.NewMigrationManager(cfg)
		return mgr.MigrateUp()
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback the last migration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		mgr := database.NewMigrationManager(cfg)
		return mgr.MigrateDown(allFlag)
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Migration status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		mgr := database.NewMigrationManager(cfg)
		return mgr.Status()
	},
}

var migrateForceCmd = &cobra.Command{
	Use:   "force <version>",
	Short: "Force migration to version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid migration version: %s", args[0])
		}
		cfg := config.Get()
		mgr := database.NewMigrationManager(cfg)
		return mgr.Force(uint(version))
	},
}

var migrateGoToVersionCmd = &cobra.Command{
	Use:     "go_to <version>",
	Aliases: []string{"goto"},
	Short:   "Migrate to a specific version",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid migration version: %s", args[0])
		}
		cfg := config.Get()
		mgr := database.NewMigrationManager(cfg)
		return mgr.GoToVersion(uint(version))
	},
}

var generateMigrationCmd = &cobra.Command{
	Use:   "generate_migration <moduleName> <fileName>",
	Short: "Generate an empty SQL file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		mgr := database.NewMigrationManager(cfg)
		return mgr.GenerateMigration(args[0], args[1])
	},
}

func init() {
	migrateDownCmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Rollback all migrations")

	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateForceCmd)
	migrateCmd.AddCommand(migrateGoToVersionCmd)

	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(generateMigrationCmd)
}
