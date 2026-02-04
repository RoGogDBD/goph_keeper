package store

import (
	"context"
	"testing"
	"time"
)

func TestLocalStoreCRUDAndSync(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	local, err := NewLocalStore(dir)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	t.Cleanup(func() {
		if err := local.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	item := Item{
		ID:        "id1",
		Type:      "text",
		Payload:   []byte("payload"),
		Meta:      map[string]string{"a": "b"},
		Deleted:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := local.Upsert(ctx, item, true); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := local.Get(ctx, "id1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != item.ID || string(got.Payload) != string(item.Payload) {
		t.Fatalf("Get mismatch: %+v", got)
	}

	list, err := local.List(ctx, false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List len=%d want=1", len(list))
	}

	dirty, err := local.ListDirty(ctx)
	if err != nil {
		t.Fatalf("ListDirty: %v", err)
	}
	if len(dirty) != 1 {
		t.Fatalf("ListDirty len=%d want=1", len(dirty))
	}

	if err := local.MarkClean(ctx, []string{"id1"}); err != nil {
		t.Fatalf("MarkClean: %v", err)
	}
	dirty, err = local.ListDirty(ctx)
	if err != nil {
		t.Fatalf("ListDirty after clean: %v", err)
	}
	if len(dirty) != 0 {
		t.Fatalf("ListDirty len=%d want=0", len(dirty))
	}

	remote := Item{
		ID:        "id1",
		Type:      "text",
		Payload:   []byte("remote"),
		Meta:      map[string]string{"a": "c"},
		Deleted:   false,
		CreatedAt: now,
		UpdatedAt: now.Add(time.Minute),
	}
	if err := local.ApplyRemote(ctx, []Item{remote}); err != nil {
		t.Fatalf("ApplyRemote: %v", err)
	}
	got, err = local.Get(ctx, "id1")
	if err != nil {
		t.Fatalf("Get after ApplyRemote: %v", err)
	}
	if string(got.Payload) != "remote" {
		t.Fatalf("payload=%q want=%q", got.Payload, "remote")
	}
}

func TestLocalStoreGetMissing(t *testing.T) {
	t.Parallel()

	local, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	t.Cleanup(func() {
		if err := local.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if _, err := local.Get(context.Background(), "missing"); err == nil {
		t.Fatalf("expected error for missing secret")
	}
}
