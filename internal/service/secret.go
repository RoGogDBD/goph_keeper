package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"goph_keeper/internal/models"
	"goph_keeper/internal/repository"
)

// SecretService defines secret business logic.
type SecretService interface {
	Create(ctx context.Context, ownerID, secretType string, payload []byte, meta map[string]string) (models.Secret, error)
	Get(ctx context.Context, ownerID, id string) (models.Secret, error)
	List(ctx context.Context, ownerID string) ([]models.Secret, error)
	Update(ctx context.Context, ownerID, id, secretType string, payload []byte, meta map[string]string) (models.Secret, error)
	Delete(ctx context.Context, ownerID, id string) error
}

// SecretServiceImpl implements SecretService.
type SecretServiceImpl struct {
	store repository.SecretReadWriter
	nowFn func() time.Time
	idFn  func() (string, error)
}

// NewSecretService creates a SecretService.
func NewSecretService(store repository.SecretReadWriter) SecretService {
	return &SecretServiceImpl{
		store: store,
		nowFn: func() time.Time { return time.Now().UTC() },
		idFn:  newID,
	}
}

func (s *SecretServiceImpl) Create(ctx context.Context, ownerID, secretType string, payload []byte, meta map[string]string) (models.Secret, error) {
	id, err := s.idFn()
	if err != nil {
		return models.Secret{}, err
	}
	now := s.nowFn()
	secret := models.Secret{
		ID:        id,
		OwnerID:   ownerID,
		Type:      secretType,
		Payload:   payload,
		Meta:      meta,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.store.Create(ctx, secret)
}

func (s *SecretServiceImpl) Get(ctx context.Context, ownerID, id string) (models.Secret, error) {
	return s.store.GetByID(ctx, ownerID, id)
}

func (s *SecretServiceImpl) List(ctx context.Context, ownerID string) ([]models.Secret, error) {
	return s.store.ListByOwner(ctx, ownerID)
}

func (s *SecretServiceImpl) Update(ctx context.Context, ownerID, id, secretType string, payload []byte, meta map[string]string) (models.Secret, error) {
	secret := models.Secret{
		ID:        id,
		OwnerID:   ownerID,
		Type:      secretType,
		Payload:   payload,
		Meta:      meta,
		UpdatedAt: s.nowFn(),
	}
	return s.store.Update(ctx, secret)
}

func (s *SecretServiceImpl) Delete(ctx context.Context, ownerID, id string) error {
	return s.store.Delete(ctx, ownerID, id)
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
