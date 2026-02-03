package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"goph_keeper/internal/models"
	"goph_keeper/internal/repository"
)

var (
	// ErrSyncInvalidID indicates a missing item ID.
	ErrSyncInvalidID = errors.New("id is required")
	// ErrSyncInvalidType indicates a missing type for a non-deleted item.
	ErrSyncInvalidType = errors.New("type is required")
)

// SyncService defines sync business logic.
type SyncService interface {
	Pull(ctx context.Context, ownerID string, since time.Time) ([]models.Secret, error)
	Push(ctx context.Context, ownerID string, items []models.Secret) (int, error)
}

// SyncServiceImpl implements SyncService.
type SyncServiceImpl struct {
	store repository.SecretSyncStore
	nowFn func() time.Time
}

// NewSyncService creates a SyncService.
func NewSyncService(store repository.SecretSyncStore) SyncService {
	return &SyncServiceImpl{
		store: store,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *SyncServiceImpl) Pull(ctx context.Context, ownerID string, since time.Time) ([]models.Secret, error) {
	return s.store.ListUpdatedSince(ctx, ownerID, since)
}

func (s *SyncServiceImpl) Push(ctx context.Context, ownerID string, items []models.Secret) (int, error) {
	applied := 0
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return 0, ErrSyncInvalidID
		}
		if !item.Deleted && strings.TrimSpace(item.Type) == "" {
			return 0, ErrSyncInvalidType
		}
		if item.CreatedAt.IsZero() {
			item.CreatedAt = s.nowFn()
		}
		if item.UpdatedAt.IsZero() {
			item.UpdatedAt = s.nowFn()
		}
		item.OwnerID = ownerID

		if _, err := s.store.Upsert(ctx, item); err != nil {
			return 0, err
		}
		applied++
	}
	return applied, nil
}
