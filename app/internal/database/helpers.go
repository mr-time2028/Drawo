package database

import (
	config2 "drawo/config"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func MigrationURL() string {
	config2.SetConfig()
	cfg := config2.Get()

	// dbURL for golang-migrate package
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DB.User,
		cfg.DB.Pass,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.Name,
	)
}

func getMigrationPath() string {
	config2.SetConfig()
	cfg := config2.Get()

	return cfg.App.MigrationsPath
}

func NewMigrator() (*migrate.Migrate, error) {
	dbURL := MigrationURL()

	return migrate.New(
		"file://"+getMigrationPath(),
		dbURL,
	)
}

func checkMigrationExists(migrationDir, moduleName, migrationFileName string) bool {
	matches, err := filepath.Glob(
		filepath.Join(
			migrationDir,
			fmt.Sprintf(
				"*_%s_%s.up.sql",
				moduleName,
				migrationFileName,
			),
		),
	)

	if err != nil {
		return false
	}

	return len(matches) > 0
}

type MigrationFile struct {
	Version uint
	Name    string
}

func getAllMigrationFiles(migrationDir string) ([]MigrationFile, error) {
	entries, err := os.ReadDir(migrationDir)
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

func getAppliedMigrations(migrationDir string, fromVersion uint, toVersion uint) ([]string, error) {
	allMigrationFiles, err := getAllMigrationFiles(migrationDir)
	if err != nil {
		return nil, err
	}

	var applied []string

	for _, migration := range allMigrationFiles {
		if migration.Version > fromVersion && migration.Version <= toVersion {
			applied = append(applied, migration.Name)
		}
	}

	return applied, nil
}

func getMigrationFileName(migrationDir string, version uint) (string, error) {
	files, err := os.ReadDir(migrationDir)
	if err != nil {
		return "", err
	}

	prefix := fmt.Sprintf("%d_", version)

	for _, file := range files {
		name := file.Name()

		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".up.sql") {
			return name, nil
		}
	}

	return "", fmt.Errorf(
		"migration file not found for version %d",
		version,
	)
}

func getMigrationName(filename string) string {
	name := strings.TrimSuffix(filename, ".up.sql")

	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 2 {
		return parts[1]
	}

	return name
}

func validateMigrations() error {
	dir := getMigrationPath()

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	upFiles := make(map[string]string)
	downFiles := make(map[string]string)

	for _, file := range files {
		name := file.Name()

		switch {
		case strings.HasSuffix(name, ".up.sql"):
			key := strings.TrimSuffix(name, ".up.sql")
			upFiles[key] = name

		case strings.HasSuffix(name, ".down.sql"):
			key := strings.TrimSuffix(name, ".down.sql")
			downFiles[key] = name
		}
	}

	for key, upFile := range upFiles {
		downFile, exists := downFiles[key]

		if !exists {
			return fmt.Errorf(
				"migration %q is missing a matching .down.sql file",
				upFile,
			)
		}

		if err := validateNotEmpty(filepath.Join(dir, upFile)); err != nil {
			return err
		}

		if err := validateNotEmpty(filepath.Join(dir, downFile)); err != nil {
			return err
		}
	}

	for key, downFile := range downFiles {
		if _, exists := upFiles[key]; !exists {
			return fmt.Errorf(
				"migration %q is missing a matching .up.sql file",
				downFile,
			)
		}
	}

	return nil
}

func validateNotEmpty(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(content)) == "" {
		return fmt.Errorf(
			"migration file %q is empty",
			filepath.Base(path),
		)
	}

	return nil
}
