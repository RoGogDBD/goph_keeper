package storage

import (
	"database/sql"
	"fmt"
	"strings"
)

type PostgresStores struct {
	DB      *sql.DB
	Users   UserStore
	Secrets SecretStore
}

func InitPostgresStores(dsn, migrationsPath string) (PostgresStores, func(), error) {
	if strings.TrimSpace(dsn) == "" {
		return PostgresStores{}, nil, fmt.Errorf("storage dsn is required for postgres")
	}

	db, err := OpenPostgres(dsn)
	if err != nil {
		return PostgresStores{}, nil, err
	}

	if err := RunMigrations(dsn, migrationsPath); err != nil {
		_ = db.Close()
		return PostgresStores{}, nil, err
	}

	stores := PostgresStores{
		DB:      db,
		Users:   NewPostgresUserStore(db),
		Secrets: NewPostgresSecretStore(db),
	}

	return stores, func() { _ = db.Close() }, nil
}
