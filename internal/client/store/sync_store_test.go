package store

import (
	"testing"
	"time"
)

func TestSyncStoreSaveLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := NewSyncStore(dir)
	now := time.Now().UTC().Truncate(time.Second)
	if err := s.Save(now); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !got.Equal(now) {
		t.Fatalf("Load=%v want=%v", got, now)
	}
}

func TestSyncStoreLoadMissing(t *testing.T) {
	t.Parallel()

	s := NewSyncStore(t.TempDir())
	if _, err := s.Load(); err == nil {
		t.Fatalf("expected error for missing sync state")
	}
}
