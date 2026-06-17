package gorm

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MigrateUp applies all pending up migrations from the supplied directory
// (relative to the working directory). Returns nil if there are no
// migrations to apply.
func MigrateUp(databaseURL, migrationsDir string) error {
	m, err := newMigrator(databaseURL, migrationsDir)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// MigrateDown rolls back a single migration step.
func MigrateDown(databaseURL, migrationsDir string) error {
	m, err := newMigrator(databaseURL, migrationsDir)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}

func newMigrator(databaseURL, migrationsDir string) (*migrate.Migrate, error) {
	if databaseURL == "" {
		return nil, errors.New("database URL is required")
	}
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}
	return migrate.New("file://"+migrationsDir, databaseURL)
}
