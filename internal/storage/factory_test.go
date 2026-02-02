package storage

import (
	"testing"

	"goph_keeper/internal/config"
)

func TestInitStoresUnsupported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		storageType string
	}{
		{name: "empty", storageType: ""},
		{name: "memory", storageType: "memory"},
		{name: "sqlite", storageType: "sqlite"},
		{name: "unknown", storageType: "unknown"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.ServerConfig{
				Storage: config.StorageConfig{
					Type: tt.storageType,
					DSN:  "dsn",
				},
			}

			_, _, err := InitStores(cfg, "/tmp")
			if err == nil {
				t.Fatalf("expected error for storage type %q", tt.storageType)
			}
		})
	}
}
