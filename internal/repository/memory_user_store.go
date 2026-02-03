package repository

import (
	"context"

	"goph_keeper/internal/models"
)

// MemoryUserStore stores users in memory for tests.
type MemoryUserStore struct {
	*MemoryStore[models.User, string]
}

// NewMemoryUserStore creates an in-memory user store.
func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		MemoryStore: NewMemoryStore(func(user models.User) string { return user.Email }, ErrUserExists, ErrUserNotFound),
	}
}

func (s *MemoryUserStore) Create(ctx context.Context, user models.User) (models.User, error) {
	return s.MemoryStore.Create(ctx, user)
}

func (s *MemoryUserStore) GetByEmail(ctx context.Context, email string) (models.User, error) {
	return s.MemoryStore.Get(ctx, email)
}
