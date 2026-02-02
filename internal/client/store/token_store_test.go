package store

import "testing"

func TestTokenStoreSaveLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := NewTokenStore(dir)
	if err := s.Save("token"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != "token" {
		t.Fatalf("Load=%q want=%q", got, "token")
	}
}

func TestTokenStoreLoadMissing(t *testing.T) {
	t.Parallel()

	s := NewTokenStore(t.TempDir())
	if _, err := s.Load(); err == nil {
		t.Fatalf("expected error for missing token")
	}
}
