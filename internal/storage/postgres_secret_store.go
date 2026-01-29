package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"goph_keeper/internal/models"
)

type PostgresSecretStore struct {
	db *sql.DB
}

func NewPostgresSecretStore(db *sql.DB) *PostgresSecretStore {
	return &PostgresSecretStore{db: db}
}

func (s *PostgresSecretStore) Create(ctx context.Context, secret models.Secret) (models.Secret, error) {
	meta, err := json.Marshal(secret.Meta)
	if err != nil {
		return models.Secret{}, err
	}

	query := `
INSERT INTO secrets (id, owner_id, type, payload, meta, created_at, updated_at, deleted)
VALUES ($1, $2, $3, $4, $5, $6, $7, FALSE)
`

	_, err = s.db.ExecContext(ctx, query,
		secret.ID,
		secret.OwnerID,
		secret.Type,
		secret.Payload,
		meta,
		secret.CreatedAt,
		secret.UpdatedAt,
	)
	if err != nil {
		return models.Secret{}, fmt.Errorf("create secret: %w", err)
	}

	return secret, nil
}

func (s *PostgresSecretStore) GetByID(ctx context.Context, ownerID, id string) (models.Secret, error) {
	query := `
SELECT id, owner_id, type, payload, meta, created_at, updated_at
FROM secrets
WHERE id = $1 AND owner_id = $2 AND deleted = FALSE
`

	var secret models.Secret
	var meta []byte
	row := s.db.QueryRowContext(ctx, query, id, ownerID)
	if err := row.Scan(&secret.ID, &secret.OwnerID, &secret.Type, &secret.Payload, &meta, &secret.CreatedAt, &secret.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return models.Secret{}, ErrSecretNotFound
		}
		return models.Secret{}, fmt.Errorf("get secret: %w", err)
	}
	if len(meta) > 0 {
		_ = json.Unmarshal(meta, &secret.Meta)
	}
	return secret, nil
}

func (s *PostgresSecretStore) ListByOwner(ctx context.Context, ownerID string) ([]models.Secret, error) {
	query := `
SELECT id, owner_id, type, payload, meta, created_at, updated_at
FROM secrets
WHERE owner_id = $1 AND deleted = FALSE
ORDER BY updated_at DESC
`

	rows, err := s.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	defer rows.Close()

	var secrets []models.Secret
	for rows.Next() {
		var secret models.Secret
		var meta []byte
		if err := rows.Scan(&secret.ID, &secret.OwnerID, &secret.Type, &secret.Payload, &meta, &secret.CreatedAt, &secret.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan secret: %w", err)
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &secret.Meta)
		}
		secrets = append(secrets, secret)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate secrets: %w", err)
	}

	return secrets, nil
}

func (s *PostgresSecretStore) Update(ctx context.Context, secret models.Secret) (models.Secret, error) {
	meta, err := json.Marshal(secret.Meta)
	if err != nil {
		return models.Secret{}, err
	}

	query := `
UPDATE secrets
SET type = $1, payload = $2, meta = $3, updated_at = $4
WHERE id = $5 AND owner_id = $6 AND deleted = FALSE
`

	result, err := s.db.ExecContext(ctx, query,
		secret.Type,
		secret.Payload,
		meta,
		secret.UpdatedAt,
		secret.ID,
		secret.OwnerID,
	)
	if err != nil {
		return models.Secret{}, fmt.Errorf("update secret: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return models.Secret{}, ErrSecretNotFound
	}

	return secret, nil
}

func (s *PostgresSecretStore) Delete(ctx context.Context, ownerID, id string) error {
	query := `
UPDATE secrets
SET deleted = TRUE, updated_at = $1
WHERE id = $2 AND owner_id = $3 AND deleted = FALSE
`

	result, err := s.db.ExecContext(ctx, query, time.Now().UTC(), id, ownerID)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return ErrSecretNotFound
	}
	return nil
}
