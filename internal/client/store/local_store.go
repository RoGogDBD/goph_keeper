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

	sq "github.com/Masterminds/squirrel"
	_ "modernc.org/sqlite"
)

// LocalStore manages local secret storage on disk.
type LocalStore struct {
	db *sql.DB
}

// NewLocalStore opens a LocalStore for a data directory.
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
		if cerr := db.Close(); cerr != nil {
			return nil, fmt.Errorf("close local db after migrate: %v (migrate: %w)", cerr, err)
		}
		return nil, err
	}

	return &LocalStore{db: db}, nil
}

// Close closes the underlying database.
func (s *LocalStore) Close() error {
	return s.db.Close()
}

// Upsert inserts or updates a local secret.
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

	builder := sq.StatementBuilder.PlaceholderFormat(sq.Question).
		Insert("secrets_local").
		Columns("id", "type", "payload", "meta", "deleted", "created_at", "updated_at", "dirty").
		Values(
			item.ID,
			item.Type,
			item.Payload,
			meta,
			item.Deleted,
			item.CreatedAt.UTC().Format(time.RFC3339),
			item.UpdatedAt.UTC().Format(time.RFC3339),
			dirty,
		).
		Suffix(`
ON CONFLICT(id) DO UPDATE SET
  type=excluded.type,
  payload=excluded.payload,
  meta=excluded.meta,
  deleted=excluded.deleted,
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  dirty=excluded.dirty`)

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build upsert local: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("upsert local: %w", err)
	}
	return nil
}

// Get fetches a local secret by ID.
func (s *LocalStore) Get(ctx context.Context, id string) (Item, error) {
	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Question).
		Select("id", "type", "payload", "meta", "deleted", "created_at", "updated_at", "dirty").
		From("secrets_local").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return Item{}, err
	}

	var item Item
	var meta []byte
	var createdAt, updatedAt string
	var dirty bool
	row := s.db.QueryRowContext(ctx, query, args...)
	if err := row.Scan(&item.ID, &item.Type, &item.Payload, &meta, &item.Deleted, &createdAt, &updatedAt, &dirty); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Item{}, ErrNotFound
		}
		return Item{}, err
	}
	if len(meta) > 0 {
		if err := json.Unmarshal(meta, &item.Meta); err != nil {
			return Item{}, err
		}
	}
	parsedCreated, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return Item{}, err
	}
	parsedUpdated, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return Item{}, err
	}
	item.CreatedAt = parsedCreated
	item.UpdatedAt = parsedUpdated
	return item, nil
}

// List returns all local secrets.
func (s *LocalStore) List(ctx context.Context, includeDeleted bool) ([]Item, error) {
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Question).
		Select("id", "type", "payload", "meta", "deleted", "created_at", "updated_at", "dirty").
		From("secrets_local").
		OrderBy("updated_at DESC")
	if !includeDeleted {
		builder = builder.Where(sq.Eq{"deleted": 0})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	var items []Item
	for rows.Next() {
		var item Item
		var meta []byte
		var createdAt, updatedAt string
		var dirty bool
		if err := rows.Scan(&item.ID, &item.Type, &item.Payload, &meta, &item.Deleted, &createdAt, &updatedAt, &dirty); err != nil {
			if cerr := rows.Close(); cerr != nil {
				return nil, fmt.Errorf("close rows after scan error: %v (scan: %w)", cerr, err)
			}
			return nil, err
		}
		if len(meta) > 0 {
			if err := json.Unmarshal(meta, &item.Meta); err != nil {
				if cerr := rows.Close(); cerr != nil {
					return nil, fmt.Errorf("close rows after decode error: %v (decode: %w)", cerr, err)
				}
				return nil, err
			}
		}
		parsedCreated, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			if cerr := rows.Close(); cerr != nil {
				return nil, fmt.Errorf("close rows after parse error: %v (parse: %w)", cerr, err)
			}
			return nil, err
		}
		parsedUpdated, err := time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			if cerr := rows.Close(); cerr != nil {
				return nil, fmt.Errorf("close rows after parse error: %v (parse: %w)", cerr, err)
			}
			return nil, err
		}
		item.CreatedAt = parsedCreated
		item.UpdatedAt = parsedUpdated
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		if cerr := rows.Close(); cerr != nil {
			return nil, fmt.Errorf("close rows after iterate error: %v (iterate: %w)", cerr, err)
		}
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	return items, nil
}

// ListDirty returns secrets marked as dirty.
func (s *LocalStore) ListDirty(ctx context.Context) ([]Item, error) {
	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Question).
		Select("id", "type", "payload", "meta", "deleted", "created_at", "updated_at", "dirty").
		From("secrets_local").
		Where(sq.Eq{"dirty": 1}).
		OrderBy("updated_at ASC").
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	var items []Item
	for rows.Next() {
		var item Item
		var meta []byte
		var createdAt, updatedAt string
		var dirty bool
		if err := rows.Scan(&item.ID, &item.Type, &item.Payload, &meta, &item.Deleted, &createdAt, &updatedAt, &dirty); err != nil {
			if cerr := rows.Close(); cerr != nil {
				return nil, fmt.Errorf("close rows after scan error: %v (scan: %w)", cerr, err)
			}
			return nil, err
		}
		if len(meta) > 0 {
			if err := json.Unmarshal(meta, &item.Meta); err != nil {
				if cerr := rows.Close(); cerr != nil {
					return nil, fmt.Errorf("close rows after decode error: %v (decode: %w)", cerr, err)
				}
				return nil, err
			}
		}
		parsedCreated, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			if cerr := rows.Close(); cerr != nil {
				return nil, fmt.Errorf("close rows after parse error: %v (parse: %w)", cerr, err)
			}
			return nil, err
		}
		parsedUpdated, err := time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			if cerr := rows.Close(); cerr != nil {
				return nil, fmt.Errorf("close rows after parse error: %v (parse: %w)", cerr, err)
			}
			return nil, err
		}
		item.CreatedAt = parsedCreated
		item.UpdatedAt = parsedUpdated
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		if cerr := rows.Close(); cerr != nil {
			return nil, fmt.Errorf("close rows after iterate error: %v (iterate: %w)", cerr, err)
		}
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	return items, nil
}

// MarkClean clears the dirty flag for provided IDs.
func (s *LocalStore) MarkClean(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Question).
		Update("secrets_local").
		Set("dirty", 0).
		Where(sq.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

// ApplyRemote applies remote updates to local storage.
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

// ErrNotFound indicates a missing local secret.
var ErrNotFound = errors.New("local secret not found")
