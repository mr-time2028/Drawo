package cmd

import (
	"drawo/internal/database"
	"fmt"
	"github.com/spf13/cobra"
	"strconv"
)

var allFlag bool

func init() {
	// generate migrations
	rootCmd.AddCommand(generateMigrationCmd)

	// migrate all migration files
	rootCmd.AddCommand(migrateCmd)

	// migrate last pending migration file
	migrateCmd.AddCommand(migrateUpCmd)

	// rollback migrations
	migrateDownCmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Rollback all migrations.")
	migrateCmd.AddCommand(migrateDownCmd)

	// status migrations
	migrateCmd.AddCommand(migrateStatusCmd)

	// force migrations
	migrateCmd.AddCommand(migrateForceCmd)

	// migrate to a specific version
	migrateCmd.AddCommand(migrateGoToVersionCmd)
}

var generateMigrationCmd = &cobra.Command{
	Use:   "generate_migration <moduleName> <fileName>",
	Short: "Generate an empty SQL file",
	Long:  `Arguments: (moduleName, fileName), Job: generate an empty sql file with the given fileName with the given moduleName`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("requires exactly 2 arguments: <moduleName> <fileName>")
		}
		moduleName := args[0]
		fileName := args[1]
		return database.GenerateMigration(moduleName, fileName)
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply all pending migrations to the database.",
	Long:  `Generate migration files using 'generate_migration' command and migrate them to the database using 'migrate' command.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return database.Migrate()
	},
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply only the last pending migration to the database.",
	Long:  `Same as the 'migrate' command but it only migrate the last pending migration to the database.'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return database.MigrateUp()
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down [--all]",
	Short: "Rollback the last migration",
	Long:  `Rollback the last migration. if you pass --all flag it will rollback all migrations from the beginning of the migration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch {
		case allFlag:
			return database.MigrateDownAll()
		default:
			return database.MigrateDown()
		}
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Migration status",
	Long:  `Determines the current status of the migration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return database.MigrateStatus()
	},
}

var migrateForceCmd = &cobra.Command{
	Use:   "force <version>",
	Short: "Force migration to version",
	Long: `Force version to a specific version given by user,
			it's not change the database schema, just move the current migration version.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid migration version: %s", args[0])
		}

		return database.MigrateForce(uint(version))
	},
}

var migrateGoToVersionCmd = &cobra.Command{
	Use:   "go_to <version>",
	Short: "Migration To Version",
	Long: `Migrate database to a specific version.
			it moves the current migration version and also change the database schema.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("requires exactly 1 arguments: <version>")
		}
		version := args[0]
		v, err := strconv.ParseUint(version, 10, 64)
		if err != nil {
			return err
		}
		return database.MigrateGoToVersion(uint(v))
	},
}
