package storage

import (
	"fmt"
	"strings"

	"goph_keeper/internal/config"
)

// InitStores initializes stores based on server config.
func InitStores(cfg config.ServerConfig, migrationsPath string) (PostgresStores, func(), error) {
	storageType := strings.ToLower(strings.TrimSpace(cfg.Storage.Type))
	switch storageType {
	case "postgres":
		return InitPostgresStores(cfg.Storage.DSN, migrationsPath)
	default:
		return PostgresStores{}, nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
}
