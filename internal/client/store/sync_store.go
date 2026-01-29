package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrSyncNotFound = errors.New("sync state not found")

type SyncStore struct {
	path string
}

func NewSyncStore(dataDir string) *SyncStore {
	return &SyncStore{path: filepath.Join(dataDir, "last_sync")}
}

func (s *SyncStore) Save(t time.Time) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.path, []byte(t.UTC().Format(time.RFC3339)), 0o600)
}

func (s *SyncStore) Load() (time.Time, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, ErrSyncNotFound
		}
		return time.Time{}, err
	}
	value := strings.TrimSpace(string(b))
	if value == "" {
		return time.Time{}, ErrSyncNotFound
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}
