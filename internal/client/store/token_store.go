package store

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrTokenNotFound indicates a missing token.
var ErrTokenNotFound = errors.New("token not found")

// TokenStore stores the auth token on disk.
type TokenStore struct {
	path string
}

// NewTokenStore creates a TokenStore for a data directory.
func NewTokenStore(dataDir string) *TokenStore {
	return &TokenStore{path: filepath.Join(dataDir, "token")}
}

// Save persists the token to disk.
func (s *TokenStore) Save(token string) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.path, []byte(token), 0o600)
}

// Load returns the stored token.
func (s *TokenStore) Load() (string, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrTokenNotFound
		}
		return "", err
	}
	return string(b), nil
}
