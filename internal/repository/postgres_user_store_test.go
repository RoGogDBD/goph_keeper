package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"goph_keeper/internal/models"
)

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	return db, mock
}

func TestPostgresUserStoreCreate(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close: %v", err)
		}
	})
	store := NewPostgresUserStore(db)

	user := testUser()
	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.ID, user.Email, user.PasswordHash, user.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if _, err := store.Create(context.Background(), user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Expectations: %v", err)
	}
}

func TestPostgresUserStoreCreateDuplicate(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close: %v", err)
		}
	})
	store := NewPostgresUserStore(db)

	user := testUser()
	pgErr := &pgconn.PgError{Code: pgerrcode.UniqueViolation}
	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.ID, user.Email, user.PasswordHash, user.CreatedAt).
		WillReturnError(pgErr)

	if _, err := store.Create(context.Background(), user); err != ErrUserExists {
		t.Fatalf("Create err=%v want ErrUserExists", err)
	}
}

func TestPostgresUserStoreGetByEmail(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close: %v", err)
		}
	})
	store := NewPostgresUserStore(db)

	user := testUser()
	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "created_at"}).
		AddRow(user.ID, user.Email, user.PasswordHash, user.CreatedAt)
	mock.ExpectQuery("SELECT").
		WithArgs(user.Email).
		WillReturnRows(rows)

	got, err := store.GetByEmail(context.Background(), user.Email)
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.Email != user.Email {
		t.Fatalf("email=%q want=%q", got.Email, user.Email)
	}
}

func TestPostgresUserStoreGetByEmailNotFound(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close: %v", err)
		}
	})
	store := NewPostgresUserStore(db)

	mock.ExpectQuery("SELECT").
		WithArgs("missing@example.com").
		WillReturnError(sql.ErrNoRows)

	if _, err := store.GetByEmail(context.Background(), "missing@example.com"); err != ErrUserNotFound {
		t.Fatalf("err=%v want ErrUserNotFound", err)
	}
}

func TestPostgresUserStoreCreateError(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close: %v", err)
		}
	})
	store := NewPostgresUserStore(db)

	user := testUser()
	mock.ExpectExec("INSERT INTO users").
		WillReturnError(sql.ErrConnDone)

	if _, err := store.Create(context.Background(), user); err == nil {
		t.Fatalf("expected error")
	}
}

func testUser() models.User {
	return models.User{
		ID:           "id1",
		Email:        "user@example.com",
		PasswordHash: []byte("hash"),
		CreatedAt:    time.Now().UTC(),
	}
}
