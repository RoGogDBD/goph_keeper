package store

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrTokenNotFound = errors.New("token not found")

type TokenStore struct {
	path string
}

func NewTokenStore(dataDir string) *TokenStore {
	return &TokenStore{path: filepath.Join(dataDir, "token")}
}

func (s *TokenStore) Save(token string) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.path, []byte(token), 0o600)
}

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
