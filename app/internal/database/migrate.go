package database

import (
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"os/exec"
	"strings"
)

func GenerateMigration(moduleName, migrationFileName string) error {
	// 1. Determine migration directory
	migrationDir := getMigrationPath()

	// 2. Check if migration with this name already exists
	existsMigrationFile := checkMigrationExists(
		migrationDir,
		moduleName,
		migrationFileName,
	)

	if existsMigrationFile {
		return fmt.Errorf(
			"migration %s:%s already exists",
			moduleName,
			migrationFileName,
		)
	}

	// 3. Generate migration files
	cmd := exec.Command(
		"migrate",
		"create",
		"-ext", "sql",
		"-dir", migrationDir,
		fmt.Sprintf("%s_%s", moduleName, migrationFileName),
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"generating migration %s:%s module failed: %w",
			moduleName,
			migrationFileName,
			err,
		)
	}

	fmt.Printf(
		"migration %s:%s successfully generated.\n",
		moduleName,
		migrationFileName,
	)

	return nil
}

func Migrate() error {
	if err := validateMigrations(); err != nil {
		return err
	}

	m, err := NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	var currentVersion uint
	if v, _, err := m.Version(); err == nil {
		currentVersion = v
	}

	err = m.Up()

	switch {
	case err == nil:
		newVersion, _, versionErr := m.Version()
		if versionErr != nil {
			return fmt.Errorf("get current migration version: %w", versionErr)
		}

		files, filesErr := getAppliedMigrations(
			"migrations",
			currentVersion,
			newVersion,
		)

		fmt.Println("Migrations applied successfully.")

		if filesErr == nil && len(files) > 0 {
			fmt.Println("Applied migrations:")

			for _, file := range files {
				fmt.Printf("  ✓ %s\n", file)
			}
		}

		return nil

	case errors.Is(err, migrate.ErrNoChange):
		fmt.Println("Database is already up to date.")
		return nil

	default:
		return fmt.Errorf("apply migrations: %w", err)
	}
}

func MigrateUp() error {
	if err := validateMigrations(); err != nil {
		return err
	}

	m, err := NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	var currentVersion uint
	if v, _, err := m.Version(); err == nil {
		currentVersion = v
	}

	err = m.Steps(1)

	switch {
	case err == nil:
		newVersion, _, versionErr := m.Version()
		if versionErr != nil {
			return fmt.Errorf("get current migration version: %w", versionErr)
		}

		files, filesErr := getAppliedMigrations(
			"migrations",
			currentVersion,
			newVersion,
		)

		fmt.Println("Migration applied successfully.")

		if filesErr == nil && len(files) > 0 {
			fmt.Println("Applied migrations:")

			for _, file := range files {
				fmt.Printf("  ✓ %s\n", file)
			}
		}
	case errors.Is(err, migrate.ErrNoChange),
		strings.Contains(err.Error(), "file does not exist"):
		fmt.Println("Database is already up to date.")
		return nil

	default:
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func MigrateDown() error {
	if err := validateMigrations(); err != nil {
		return err
	}

	m, err := NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	var currentVersion uint

	if v, _, err := m.Version(); err == nil {
		currentVersion = v
	}

	err = m.Steps(-1)

	switch {
	case err == nil:
		files, filesErr := getAppliedMigrations(
			"migrations",
			currentVersion-1,
			currentVersion,
		)

		fmt.Println("Migration rollback completed successfully.")

		if filesErr == nil && len(files) > 0 {
			fmt.Println("Rolled back migration:")

			for _, file := range files {
				fmt.Printf("  ✓ %s\n", file)
			}
		}

		return nil

	case errors.Is(err, migrate.ErrNoChange),
		strings.Contains(err.Error(), "file does not exist"):
		fmt.Println("No migrations to roll back.")
		return nil

	default:
		return fmt.Errorf("rollback migration: %w", err)
	}
}

func MigrateDownAll() error {
	if err := validateMigrations(); err != nil {
		return err
	}

	m, err := NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	err = m.Down()

	switch {
	case err == nil:
		dir := getMigrationPath()
		files, filesErr := getAllMigrationFiles(dir)

		fmt.Println("All migrations rolled back successfully.")

		if filesErr == nil && len(files) > 0 {
			fmt.Println("Rolled back migrations:")

			for i := len(files) - 1; i >= 0; i-- {
				fmt.Printf("  ✓ %s\n", files[i].Name)
			}
		}

		return nil

	case errors.Is(err, migrate.ErrNoChange),
		strings.Contains(err.Error(), "file does not exist"):
		fmt.Println("No migrations to roll back, Database is already at version 0.")
		return nil

	default:
		return fmt.Errorf("rollback all migrations: %w", err)
	}
}

func MigrateStatus() error {
	if err := validateMigrations(); err != nil {
		return err
	}

	m, err := NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	version, dirty, err := m.Version()

	switch {
	case errors.Is(err, migrate.ErrNilVersion):
		fmt.Println("Current version: NONE")
		fmt.Println("Status: DIRTY")
		return nil

	case err != nil:
		return fmt.Errorf("migration status: %w", err)
	}

	migrationDir := getMigrationPath()
	migrationFileName, _ := getMigrationFileName(
		migrationDir,
		uint(version),
	)

	fmt.Printf("Version : %d\n", version)

	if migrationFileName != "" {
		fmt.Printf(
			"Name    : %s\n",
			getMigrationName(migrationFileName),
		)
	}

	if dirty {
		fmt.Println("Status  : DIRTY")
	} else {
		fmt.Println("Status  : CLEAN")
	}

	return nil
}

func MigrateForce(version uint) error {
	if err := validateMigrations(); err != nil {
		return err
	}

	m, err := NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Force(int(version)); err != nil {
		return fmt.Errorf(
			"force migration version %d: %w",
			version,
			err,
		)
	}

	migrationFileName, _ := getMigrationFileName(
		"migrations",
		version,
	)

	if migrationFileName != "" {
		fmt.Printf(
			"Database forced to version %d (%s).\n",
			version,
			getMigrationName(migrationFileName),
		)
	} else {
		fmt.Printf(
			"Database forced to version %d.\n",
			version,
		)
	}

	return nil
}

func MigrateGoToVersion(version uint) error {
	if err := validateMigrations(); err != nil {
		return err
	}

	m, err := NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	err = m.Migrate(version)

	migrationFileName, _ := getMigrationFileName(
		"migrations",
		version,
	)

	displayName := getMigrationName(migrationFileName)

	switch {
	case err == nil:
		if displayName != "" {
			fmt.Printf(
				"Database migrated successfully to version %d (%s).\n",
				version,
				displayName,
			)
		} else {
			fmt.Printf(
				"Database migrated successfully to version %d.\n",
				version,
			)
		}
		return nil

	case errors.Is(err, migrate.ErrNoChange):
		if displayName != "" {
			fmt.Printf(
				"Database is already at version %d (%s).\n",
				version,
				displayName,
			)
		} else {
			fmt.Printf(
				"Database is already at version %d.\n",
				version,
			)
		}
		return nil

	default:
		return fmt.Errorf(
			"migrate to version %d: %w",
			version,
			err,
		)
	}
}
