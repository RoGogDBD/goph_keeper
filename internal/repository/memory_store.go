package repository

import (
	"context"
	"sync"
)

// MemoryStore is a generic in-memory store keyed by a comparable identifier.
type MemoryStore[T any, K comparable] struct {
	mu          sync.RWMutex
	items       map[K]T
	keyFn       func(T) K
	errExists   error
	errNotFound error
}

// NewMemoryStore creates a new MemoryStore with key extraction and errors.
func NewMemoryStore[T any, K comparable](keyFn func(T) K, errExists, errNotFound error) *MemoryStore[T, K] {
	return &MemoryStore[T, K]{
		items:       make(map[K]T),
		keyFn:       keyFn,
		errExists:   errExists,
		errNotFound: errNotFound,
	}
}

// Create stores an item if it does not already exist.
func (s *MemoryStore[T, K]) Create(_ context.Context, item T) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.keyFn(item)
	if _, ok := s.items[key]; ok {
		var zero T
		return zero, s.errExists
	}

	s.items[key] = item
	return item, nil
}

// Get returns an item by key.
func (s *MemoryStore[T, K]) Get(_ context.Context, key K) (T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[key]
	if !ok {
		var zero T
		return zero, s.errNotFound
	}

	return item, nil
}
