package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"goph_keeper/internal/repository"
)

// PostgresStores bundles Postgres-backed stores and DB handle.
type PostgresStores struct {
	DB      *sql.DB
	Users   repository.UserStore
	Secrets repository.SecretStore
}

var openPostgresFn = OpenPostgres
var runMigrationsFn = RunMigrations

// InitPostgresStores initializes Postgres stores and runs migrations.
func InitPostgresStores(dsn, migrationsPath string) (PostgresStores, func(), error) {
	if strings.TrimSpace(dsn) == "" {
		return PostgresStores{}, nil, fmt.Errorf("storage dsn is required for postgres")
	}

	db, err := openPostgresWithRetry(dsn, 10, time.Second)
	if err != nil {
		return PostgresStores{}, nil, err
	}

	if err := runMigrationsFn(dsn, migrationsPath); err != nil {
		if cerr := db.Close(); cerr != nil {
			return PostgresStores{}, nil, fmt.Errorf("close db after migrations: %v (migrate: %w)", cerr, err)
		}
		return PostgresStores{}, nil, err
	}

	stores := PostgresStores{
		DB:      db,
		Users:   repository.NewPostgresUserStore(db),
		Secrets: repository.NewPostgresSecretStore(db),
	}

	return stores, func() {
		if err := db.Close(); err != nil {
			_ = err
		}
	}, nil
}

func openPostgresWithRetry(dsn string, attempts int, delay time.Duration) (*sql.DB, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		db, err := openPostgresFn(dsn)
		if err == nil {
			return db, nil
		}
		lastErr = err
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("open postgres after retries: %w", lastErr)
}
