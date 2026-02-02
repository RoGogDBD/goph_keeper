package storage

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies DB migrations.
func RunMigrations(dsn, migrationsPath string) error {
	sourceURL := migrationsPath
	if !strings.HasPrefix(sourceURL, "file://") {
		sourceURL = "file://" + filepath.ToSlash(sourceURL)
	}

	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrate: %w", err)
	}
	return nil
}
