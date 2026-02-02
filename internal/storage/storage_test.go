package storage

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestStorage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "open_postgres_invalid_dsn",
			fn: func(t *testing.T) {
				if _, err := OpenPostgres("postgres://invalid:invalid@127.0.0.1:1/db?sslmode=disable"); err == nil {
					t.Fatalf("expected error for invalid dsn")
				}
			},
		},
		{
			name: "run_migrations_invalid_path",
			fn: func(t *testing.T) {
				if err := RunMigrations("postgres://invalid:invalid@127.0.0.1:1/db?sslmode=disable", "/no/such/path"); err == nil {
					t.Fatalf("expected error for invalid migrations path")
				}
			},
		},
		{
			name: "open_postgres_with_retry",
			fn: func(t *testing.T) {
				if _, err := openPostgresWithRetry("postgres://invalid:invalid@127.0.0.1:1/db?sslmode=disable", 2, 0); err == nil {
					t.Fatalf("expected error from openPostgresWithRetry")
				}
			},
		},
		{
			name: "init_postgres_stores_empty_dsn",
			fn: func(t *testing.T) {
				if _, _, err := InitPostgresStores("", "/tmp"); err == nil {
					t.Fatalf("expected error for empty dsn")
				}
			},
		},
		{
			name: "init_postgres_stores_success",
			fn: func(t *testing.T) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatalf("sqlmock.New: %v", err)
				}
				t.Cleanup(func() {
					if err := db.Close(); err != nil {
						t.Fatalf("db.Close: %v", err)
					}
				})
				_ = mock

				origOpen := openPostgresFn
				origMigrate := runMigrationsFn
				openPostgresFn = func(dsn string) (*sql.DB, error) {
					return db, nil
				}
				runMigrationsFn = func(dsn, path string) error {
					return nil
				}
				t.Cleanup(func() {
					openPostgresFn = origOpen
					runMigrationsFn = origMigrate
				})

				stores, cleanup, err := InitPostgresStores("dsn", "/tmp")
				if err != nil {
					t.Fatalf("InitPostgresStores: %v", err)
				}
				if stores.DB == nil || cleanup == nil {
					t.Fatalf("expected db and cleanup")
				}
				cleanup()
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, tc.fn)
	}
}
