package repository

import (
	"context"
	"time"

	"goph_keeper/internal/models"
)

type SecretStore interface {
	Create(ctx context.Context, secret models.Secret) (models.Secret, error)
	GetByID(ctx context.Context, ownerID, id string) (models.Secret, error)
	ListByOwner(ctx context.Context, ownerID string) ([]models.Secret, error)
	ListUpdatedSince(ctx context.Context, ownerID string, since time.Time) ([]models.Secret, error)
	Update(ctx context.Context, secret models.Secret) (models.Secret, error)
	Delete(ctx context.Context, ownerID, id string) error
	Upsert(ctx context.Context, secret models.Secret) (models.Secret, error)
}
