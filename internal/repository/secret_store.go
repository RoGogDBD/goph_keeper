package repository

import (
	"context"
	"time"

	"goph_keeper/internal/models"
)

// SecretStore aggregates interfaces for secrets persistence.
type SecretStore interface {
	SecretReadWriter
	SecretSyncStore
}

// SecretReadWriter defines basic CRUD operations for secrets.
type SecretReadWriter interface {
	Create(ctx context.Context, secret models.Secret) (models.Secret, error)
	GetByID(ctx context.Context, ownerID, id string) (models.Secret, error)
	ListByOwner(ctx context.Context, ownerID string) ([]models.Secret, error)
	Update(ctx context.Context, secret models.Secret) (models.Secret, error)
	Delete(ctx context.Context, ownerID, id string) error
}

// SecretSyncStore defines synchronization operations for secrets.
type SecretSyncStore interface {
	ListUpdatedSince(ctx context.Context, ownerID string, since time.Time) ([]models.Secret, error)
	Upsert(ctx context.Context, secret models.Secret) (models.Secret, error)
}
