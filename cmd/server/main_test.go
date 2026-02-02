package main

import (
	"context"
	"io"
	"log"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"net/http"

	"goph_keeper/internal/config"
	"goph_keeper/internal/models"
	"goph_keeper/internal/repository"
	"goph_keeper/internal/storage"
)

func TestMigrationsDir(t *testing.T) {
	t.Parallel()

	dir, err := migrationsDir()
	if err != nil {
		t.Fatalf("migrationsDir: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(dir), "/migrations") {
		t.Fatalf("unexpected migrations path: %s", dir)
	}
}

type stubSecretStore struct{}

func (s *stubSecretStore) Create(_ context.Context, secret models.Secret) (models.Secret, error) {
	return secret, nil
}
func (s *stubSecretStore) GetByID(_ context.Context, ownerID, id string) (models.Secret, error) {
	return models.Secret{}, repository.ErrSecretNotFound
}
func (s *stubSecretStore) ListByOwner(_ context.Context, ownerID string) ([]models.Secret, error) {
	return []models.Secret{}, nil
}
func (s *stubSecretStore) ListUpdatedSince(_ context.Context, ownerID string, since time.Time) ([]models.Secret, error) {
	return []models.Secret{}, nil
}
func (s *stubSecretStore) Update(_ context.Context, secret models.Secret) (models.Secret, error) {
	return secret, nil
}
func (s *stubSecretStore) Delete(_ context.Context, ownerID, id string) error {
	return nil
}
func (s *stubSecretStore) Upsert(_ context.Context, secret models.Secret) (models.Secret, error) {
	return secret, nil
}

func TestRun(t *testing.T) {
	t.Setenv("GOPHKEEPER_SERVER_HOST", "127.0.0.1")
	t.Setenv("GOPHKEEPER_SERVER_PORT", "8080")
	t.Setenv("GOPHKEEPER_SERVER_STORAGE", "postgres")
	t.Setenv("GOPHKEEPER_SERVER_DSN", "dsn")
	t.Setenv("GOPHKEEPER_SERVER_JWT_KEY", "key")
	t.Setenv("GOPHKEEPER_LOG_LEVEL", "info")
	t.Setenv("GOPHKEEPER_LOG_FORMAT", "text")
	t.Setenv("GOPHKEEPER_SERVER_TLS", "true")
	t.Setenv("GOPHKEEPER_SERVER_TLS_CERT", "cert")
	t.Setenv("GOPHKEEPER_SERVER_TLS_KEY", "keyfile")

	origInit := initStores
	origListen := listenAndServe
	origListenTLS := listenAndServeTLS
	initStores = func(cfg config.ServerConfig, path string) (storage.PostgresStores, func(), error) {
		return storage.PostgresStores{
			Users:   repository.NewMemoryUserStore(),
			Secrets: &stubSecretStore{},
		}, func() {}, nil
	}
	listenAndServe = func(addr string, handler http.Handler) error {
		return nil
	}
	listenAndServeTLS = func(addr, certFile, keyFile string, handler http.Handler) error {
		return nil
	}
	t.Cleanup(func() {
		initStores = origInit
		listenAndServe = origListen
		listenAndServeTLS = origListenTLS
	})

	logger := log.New(io.Discard, "", 0)
	if err := run([]string{}, logger); err != nil {
		t.Fatalf("run: %v", err)
	}
}
