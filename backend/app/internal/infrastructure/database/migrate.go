package database

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"drawo/config"
)

// MigrationFile represents a metadata about a migration file.
type MigrationFile struct {
	Version uint
	Name    string
}

// MigrationManager handles database migrations with detailed reporting.
type MigrationManager struct {
	cfg config.Config
}

// NewMigrationManager creates a new migration manager.
func NewMigrationManager(cfg config.Config) *MigrationManager {
	return &MigrationManager{cfg: cfg}
}

// GetMigrationURL returns the database URL for golang-migrate.
func (m *MigrationManager) GetMigrationURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		m.cfg.Database.User,
		m.cfg.Database.Password,
		m.cfg.Database.Host,
		m.cfg.Database.Port,
		m.cfg.Database.Name,
		m.cfg.Database.SSLMode,
	)
}

// NewMigrator initializes a golang-migrate instance.
func (m *MigrationManager) NewMigrator() (*migrate.Migrate, error) {
	return migrate.New(
		"file://"+m.cfg.App.MigrationsPath,
		m.GetMigrationURL(),
	)
}

// GenerateMigration creates an empty pair of up/down migration files.
func (m *MigrationManager) GenerateMigration(moduleName, migrationFileName string) error {
	migrationDir := m.cfg.App.MigrationsPath

	if err := os.MkdirAll(migrationDir, 0755); err != nil {
		return fmt.Errorf("create migration directory: %w", err)
	}

	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return err
	}

	maxVersion := uint(0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}
		version, err := strconv.ParseUint(parts[0], 10, 64)
		if err == nil && uint(version) > maxVersion {
			maxVersion = uint(version)
		}
	}
	nextVersion := maxVersion + 1

	namePart := fmt.Sprintf("%s_%s", moduleName, migrationFileName)
	matches, _ := filepath.Glob(filepath.Join(migrationDir, fmt.Sprintf("*_%s.up.sql", namePart)))
	if len(matches) > 0 {
		return fmt.Errorf("migration %s already exists", namePart)
	}

	versionStr := fmt.Sprintf("%06d", nextVersion)
	upName := fmt.Sprintf("%s_%s.up.sql", versionStr, namePart)
	downName := fmt.Sprintf("%s_%s.down.sql", versionStr, namePart)

	if err := os.WriteFile(filepath.Join(migrationDir, upName), []byte("-- Up migration\n"), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(migrationDir, downName), []byte("-- Down migration\n"), 0644); err != nil {
		return err
	}

	fmt.Printf("migration %s:%s successfully generated.\n", moduleName, migrationFileName)
	return nil
}

// Migrate applies all pending migrations and reports progress.
func (m *MigrationManager) Migrate() error {
	if err := m.Validate(); err != nil {
		return err
	}

	mgr, err := m.NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer mgr.Close()

	var currentVersion uint
	if v, _, err := mgr.Version(); err == nil {
		currentVersion = v
	}

	err = mgr.Up()

	switch {
	case err == nil:
		newVersion, _, versionErr := mgr.Version()
		if versionErr != nil {
			return fmt.Errorf("get current migration version: %w", versionErr)
		}

		files, _ := m.getAppliedMigrations(currentVersion, newVersion)
		fmt.Println("Migrations applied successfully.")
		if len(files) > 0 {
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

// MigrateUp applies the single next pending migration.
func (m *MigrationManager) MigrateUp() error {
	if err := m.Validate(); err != nil {
		return err
	}

	mgr, err := m.NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer mgr.Close()

	var currentVersion uint
	if v, _, err := mgr.Version(); err == nil {
		currentVersion = v
	}

	err = mgr.Steps(1)

	switch {
	case err == nil:
		newVersion, _, _ := mgr.Version()
		files, _ := m.getAppliedMigrations(currentVersion, newVersion)
		fmt.Println("Migration applied successfully.")
		if len(files) > 0 {
			fmt.Println("Applied migrations:")
			for _, file := range files {
				fmt.Printf("  ✓ %s\n", file)
			}
		}
		return nil

	case errors.Is(err, migrate.ErrNoChange), strings.Contains(fmt.Sprint(err), "file does not exist"):
		fmt.Println("Database is already up to date.")
		return nil

	default:
		return fmt.Errorf("apply migration: %w", err)
	}
}

// MigrateDown rolls back the last migration or all of them.
func (m *MigrationManager) MigrateDown(all bool) error {
	if err := m.Validate(); err != nil {
		return err
	}

	mgr, err := m.NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer mgr.Close()

	var currentVersion uint
	if v, _, err := mgr.Version(); err == nil {
		currentVersion = v
	}

	var errMigrate error
	if all {
		errMigrate = mgr.Down()
	} else {
		errMigrate = mgr.Steps(-1)
	}

	switch {
	case errMigrate == nil:
		if all {
			dir := m.cfg.App.MigrationsPath
			files, _ := m.getAllMigrationFiles(dir)
			fmt.Println("All migrations rolled back successfully.")
			if len(files) > 0 {
				fmt.Println("Rolled back migrations:")
				for i := len(files) - 1; i >= 0; i-- {
					fmt.Printf("  ✓ %s\n", files[i].Name)
				}
			}
		} else {
			files, _ := m.getAppliedMigrations(currentVersion-1, currentVersion)
			fmt.Println("Migration rollback completed successfully.")
			if len(files) > 0 {
				fmt.Println("Rolled back migration:")
				for _, file := range files {
					fmt.Printf("  ✓ %s\n", file)
				}
			}
		}
		return nil

	case errors.Is(errMigrate, migrate.ErrNoChange), strings.Contains(fmt.Sprint(errMigrate), "file does not exist"):
		if all {
			fmt.Println("No migrations to roll back, Database is already at version 0.")
		} else {
			fmt.Println("No migrations to roll back.")
		}
		return nil

	default:
		return fmt.Errorf("rollback migration: %w", errMigrate)
	}
}

// Status returns the current migration status and name.
func (m *MigrationManager) Status() error {
	if err := m.Validate(); err != nil {
		return err
	}

	mgr, err := m.NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer mgr.Close()

	version, dirty, err := mgr.Version()

	switch {
	case errors.Is(err, migrate.ErrNilVersion):
		fmt.Println("Current version: NONE")
		fmt.Println("Status: CLEAN (Version 0)")
		return nil

	case err != nil:
		return fmt.Errorf("migration status: %w", err)
	}

	migrationDir := m.cfg.App.MigrationsPath
	fileName, _ := m.getMigrationFileName(migrationDir, uint(version))

	fmt.Printf("Version : %d\n", version)
	if fileName != "" {
		fmt.Printf("Name    : %s\n", m.getFriendlyName(fileName))
	}

	if dirty {
		fmt.Println("Status  : DIRTY")
	} else {
		fmt.Println("Status  : CLEAN")
	}

	return nil
}

// Force sets the migration version with friendly reporting.
func (m *MigrationManager) Force(version uint) error {
	mgr, err := m.NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer mgr.Close()

	if err := mgr.Force(int(version)); err != nil {
		return fmt.Errorf("force migration version %d: %w", version, err)
	}

	fileName, _ := m.getMigrationFileName(m.cfg.App.MigrationsPath, version)
	if fileName != "" {
		fmt.Printf("Database forced to version %d (%s).\n", version, m.getFriendlyName(fileName))
	} else {
		fmt.Printf("Database forced to version %d.\n", version)
	}

	return nil
}

// GoToVersion migrates to a specific version with friendly reporting.
func (m *MigrationManager) GoToVersion(version uint) error {
	if err := m.Validate(); err != nil {
		return err
	}

	mgr, err := m.NewMigrator()
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer mgr.Close()

	err = mgr.Migrate(version)
	fileName, _ := m.getMigrationFileName(m.cfg.App.MigrationsPath, version)
	displayName := m.getFriendlyName(fileName)

	switch {
	case err == nil:
		if displayName != "" {
			fmt.Printf("Database migrated successfully to version %d (%s).\n", version, displayName)
		} else {
			fmt.Printf("Database migrated successfully to version %d.\n", version)
		}
		return nil

	case errors.Is(err, migrate.ErrNoChange):
		if displayName != "" {
			fmt.Printf("Database is already at version %d (%s).\n", version, displayName)
		} else {
			fmt.Printf("Database is already at version %d.\n", version)
		}
		return nil

	default:
		return fmt.Errorf("migrate to version %d: %w", version, err)
	}
}

// Validate ensures migration file integrity.
func (m *MigrationManager) Validate() error {
	dir := m.cfg.App.MigrationsPath
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	upFiles := make(map[string]bool)
	downFiles := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			upFiles[strings.TrimSuffix(name, ".up.sql")] = true
		} else if strings.HasSuffix(name, ".down.sql") {
			downFiles[strings.TrimSuffix(name, ".down.sql")] = true
		}
	}

	for base := range upFiles {
		if !downFiles[base] {
			return fmt.Errorf("migration %q is missing a matching .down.sql file", base+".up.sql")
		}
		content, _ := os.ReadFile(filepath.Join(dir, base+".up.sql"))
		if len(strings.TrimSpace(string(content))) == 0 {
			return fmt.Errorf("migration file %q is empty", base+".up.sql")
		}
	}
	return nil
}

// Helpers

func (m *MigrationManager) getAllMigrationFiles(dir string) ([]MigrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var migrations []MigrationFile
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			continue
		}
		version, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			continue
		}
		migrations = append(migrations, MigrationFile{
			Version: uint(version),
			Name:    name,
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (m *MigrationManager) getAppliedMigrations(fromVersion uint, toVersion uint) ([]string, error) {
	allFiles, err := m.getAllMigrationFiles(m.cfg.App.MigrationsPath)
	if err != nil {
		return nil, err
	}

	var applied []string
	for _, migration := range allFiles {
		if migration.Version > fromVersion && migration.Version <= toVersion {
			applied = append(applied, migration.Name)
		}
	}
	return applied, nil
}

func (m *MigrationManager) getMigrationFileName(migrationDir string, version uint) (string, error) {
	files, err := os.ReadDir(migrationDir)
	if err != nil {
		return "", err
	}

	prefix := fmt.Sprintf("%d_", version)
	// Also handle padded version if necessary, though our generator uses 06d
	prefixPadded := fmt.Sprintf("%06d_", version)

	for _, file := range files {
		name := file.Name()
		if (strings.HasPrefix(name, prefix) || strings.HasPrefix(name, prefixPadded)) && strings.HasSuffix(name, ".up.sql") {
			return name, nil
		}
	}

	return "", fmt.Errorf("migration file not found for version %d", version)
}

func (m *MigrationManager) getFriendlyName(filename string) string {
	if filename == "" {
		return ""
	}
	name := strings.TrimSuffix(filename, ".up.sql")
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return name
}
