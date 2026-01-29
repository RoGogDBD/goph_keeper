package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"goph_keeper/internal/models"
)

type PostgresUserStore struct {
	db *sql.DB
}

func NewPostgresUserStore(db *sql.DB) *PostgresUserStore {
	return &PostgresUserStore{db: db}
}

func (s *PostgresUserStore) Create(ctx context.Context, user models.User) (models.User, error) {
	query := `
INSERT INTO users (id, email, password_hash, created_at)
VALUES ($1, $2, $3, $4)
`

	_, err := s.db.ExecContext(ctx, query, user.ID, user.Email, user.PasswordHash, user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return models.User{}, ErrUserExists
		}
		return models.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *PostgresUserStore) GetByEmail(ctx context.Context, email string) (models.User, error) {
	query := `
SELECT id, email, password_hash, created_at
FROM users
WHERE email = $1
`

	var user models.User
	row := s.db.QueryRowContext(ctx, query, email)
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}
