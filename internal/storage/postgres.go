package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const postgresDriver = "pgx"

// OpenPostgres opens a Postgres DB connection.
func OpenPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open(postgresDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		if cerr := db.Close(); cerr != nil {
			return nil, fmt.Errorf("ping postgres: %w (close: %v)", err, cerr)
		}
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}

// Migrations are handled via golang-migrate from SQL files.
