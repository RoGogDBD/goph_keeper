package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

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

	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("secrets").
		Columns("id", "owner_id", "type", "payload", "meta", "created_at", "updated_at", "deleted").
		Values(secret.ID, secret.OwnerID, secret.Type, secret.Payload, meta, secret.CreatedAt, secret.UpdatedAt, false).
		ToSql()
	if err != nil {
		return models.Secret{}, fmt.Errorf("build insert secret: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return models.Secret{}, fmt.Errorf("create secret: %w", err)
	}

	return secret, nil
}

func (s *PostgresSecretStore) GetByID(ctx context.Context, ownerID, id string) (models.Secret, error) {
	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "owner_id", "type", "payload", "meta", "created_at", "updated_at").
		From("secrets").
		Where(sq.Eq{"id": id, "owner_id": ownerID, "deleted": false}).
		ToSql()
	if err != nil {
		return models.Secret{}, fmt.Errorf("build select secret: %w", err)
	}

	var secret models.Secret
	var meta []byte
	row := s.db.QueryRowContext(ctx, query, args...)
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
	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "owner_id", "type", "payload", "meta", "created_at", "updated_at").
		From("secrets").
		Where(sq.Eq{"owner_id": ownerID, "deleted": false}).
		OrderBy("updated_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list secrets: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
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

	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Update("secrets").
		Set("type", secret.Type).
		Set("payload", secret.Payload).
		Set("meta", meta).
		Set("updated_at", secret.UpdatedAt).
		Where(sq.Eq{"id": secret.ID, "owner_id": secret.OwnerID, "deleted": false}).
		ToSql()
	if err != nil {
		return models.Secret{}, fmt.Errorf("build update secret: %w", err)
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return models.Secret{}, fmt.Errorf("update secret: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return models.Secret{}, ErrSecretNotFound
	}

	return secret, nil
}

func (s *PostgresSecretStore) Delete(ctx context.Context, ownerID, id string) error {
	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Update("secrets").
		Set("deleted", true).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{"id": id, "owner_id": ownerID, "deleted": false}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete secret: %w", err)
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return ErrSecretNotFound
	}
	return nil
}
