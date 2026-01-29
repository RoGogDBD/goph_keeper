package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type LocalStore struct {
	db *sql.DB
}

func NewLocalStore(dataDir string) (*LocalStore, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}

	path := filepath.Join(dataDir, "local.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := migrateLocal(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &LocalStore{db: db}, nil
}

func (s *LocalStore) Close() error {
	return s.db.Close()
}

func (s *LocalStore) Upsert(ctx context.Context, item Item, dirty bool) error {
	if item.Deleted && item.Type == "" {
		item.Type = "deleted"
	}
	if item.Payload == nil {
		item.Payload = []byte{}
	}
	meta, err := json.Marshal(item.Meta)
	if err != nil {
		return err
	}

	query := `
INSERT INTO secrets_local (id, type, payload, meta, deleted, created_at, updated_at, dirty)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  type=excluded.type,
  payload=excluded.payload,
  meta=excluded.meta,
  deleted=excluded.deleted,
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  dirty=excluded.dirty
`

	_, err = s.db.ExecContext(ctx, query,
		item.ID,
		item.Type,
		item.Payload,
		meta,
		item.Deleted,
		item.CreatedAt.UTC().Format(time.RFC3339),
		item.UpdatedAt.UTC().Format(time.RFC3339),
		dirty,
	)
	if err != nil {
		return fmt.Errorf("upsert local: %w", err)
	}
	return nil
}

func (s *LocalStore) Get(ctx context.Context, id string) (Item, error) {
	query := `
SELECT id, type, payload, meta, deleted, created_at, updated_at, dirty
FROM secrets_local
WHERE id = ?
`

	var item Item
	var meta []byte
	var createdAt, updatedAt string
	var dirty bool
	row := s.db.QueryRowContext(ctx, query, id)
	if err := row.Scan(&item.ID, &item.Type, &item.Payload, &meta, &item.Deleted, &createdAt, &updatedAt, &dirty); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Item{}, ErrNotFound
		}
		return Item{}, err
	}
	_ = json.Unmarshal(meta, &item.Meta)
	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return item, nil
}

func (s *LocalStore) List(ctx context.Context, includeDeleted bool) ([]Item, error) {
	query := `
SELECT id, type, payload, meta, deleted, created_at, updated_at, dirty
FROM secrets_local
`
	if !includeDeleted {
		query += " WHERE deleted = 0"
	}
	query += " ORDER BY updated_at DESC"

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		var meta []byte
		var createdAt, updatedAt string
		var dirty bool
		if err := rows.Scan(&item.ID, &item.Type, &item.Payload, &meta, &item.Deleted, &createdAt, &updatedAt, &dirty); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(meta, &item.Meta)
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *LocalStore) ListDirty(ctx context.Context) ([]Item, error) {
	query := `
SELECT id, type, payload, meta, deleted, created_at, updated_at, dirty
FROM secrets_local
WHERE dirty = 1
ORDER BY updated_at ASC
`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		var meta []byte
		var createdAt, updatedAt string
		var dirty bool
		if err := rows.Scan(&item.ID, &item.Type, &item.Payload, &meta, &item.Deleted, &createdAt, &updatedAt, &dirty); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(meta, &item.Meta)
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *LocalStore) MarkClean(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query := "UPDATE secrets_local SET dirty = 0 WHERE id = ?"
	for _, id := range ids {
		if _, err := s.db.ExecContext(ctx, query, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *LocalStore) ApplyRemote(ctx context.Context, items []Item) error {
	for _, item := range items {
		local, err := s.Get(ctx, item.ID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}

		if errors.Is(err, ErrNotFound) || item.UpdatedAt.After(local.UpdatedAt) {
			if err := s.Upsert(ctx, item, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func migrateLocal(db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS secrets_local (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  payload BLOB NOT NULL,
  meta TEXT NOT NULL DEFAULT '{}',
  deleted INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  dirty INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_secrets_local_updated ON secrets_local(updated_at);
CREATE INDEX IF NOT EXISTS idx_secrets_local_dirty ON secrets_local(dirty);
`

	_, err := db.Exec(ddl)
	return err
}

var ErrNotFound = errors.New("local secret not found")
