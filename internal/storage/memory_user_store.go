package storage

import (
	"context"
	"sync"

	"goph_keeper/internal/models"
)

type MemoryUserStore struct {
	mu    sync.RWMutex
	users map[string]models.User
}

func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{users: make(map[string]models.User)}
}

func (s *MemoryUserStore) Create(_ context.Context, user models.User) (models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[user.Email]; ok {
		return models.User{}, ErrUserExists
	}

	s.users[user.Email] = user
	return user, nil
}

func (s *MemoryUserStore) GetByEmail(_ context.Context, email string) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[email]
	if !ok {
		return models.User{}, ErrUserNotFound
	}

	return user, nil
}
