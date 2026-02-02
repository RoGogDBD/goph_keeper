package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"goph_keeper/internal/models"
)

func TestPostgresSecretStoreCreateGetList(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	cleanupDB(t, db, mock)
	store := NewPostgresSecretStore(db)

	secret := testSecret()
	meta, err := json.Marshal(secret.Meta)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	mock.ExpectExec("INSERT INTO secrets").
		WithArgs(secret.ID, secret.OwnerID, secret.Type, secret.Payload, meta, secret.CreatedAt, secret.UpdatedAt, secret.Deleted).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if _, err := store.Create(context.Background(), secret); err != nil {
		t.Fatalf("Create: %v", err)
	}

	rows := sqlmock.NewRows([]string{"id", "owner_id", "type", "payload", "meta", "created_at", "updated_at"}).
		AddRow(secret.ID, secret.OwnerID, secret.Type, secret.Payload, meta, secret.CreatedAt, secret.UpdatedAt)
	mock.ExpectQuery("SELECT").
		WillReturnRows(rows)

	got, err := store.GetByID(context.Background(), secret.OwnerID, secret.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != secret.ID {
		t.Fatalf("GetByID id=%q want=%q", got.ID, secret.ID)
	}

	listRows := sqlmock.NewRows([]string{"id", "owner_id", "type", "payload", "meta", "created_at", "updated_at"}).
		AddRow(secret.ID, secret.OwnerID, secret.Type, secret.Payload, meta, secret.CreatedAt, secret.UpdatedAt)
	mock.ExpectQuery("SELECT").
		WillReturnRows(listRows)

	list, err := store.ListByOwner(context.Background(), secret.OwnerID)
	if err != nil {
		t.Fatalf("ListByOwner: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListByOwner len=%d want=1", len(list))
	}
}

func TestPostgresSecretStoreUpdateDeleteUpsert(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	cleanupDB(t, db, mock)
	store := NewPostgresSecretStore(db)

	secret := testSecret()
	meta, err := json.Marshal(secret.Meta)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	mock.ExpectExec("UPDATE secrets").
		WithArgs(secret.Type, secret.Payload, meta, sqlmock.AnyArg(), false, secret.ID, secret.OwnerID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := store.Update(context.Background(), secret); err != nil {
		t.Fatalf("Update: %v", err)
	}

	mock.ExpectExec("UPDATE secrets").
		WithArgs(true, sqlmock.AnyArg(), false, secret.ID, secret.OwnerID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.Delete(context.Background(), secret.OwnerID, secret.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	mock.ExpectExec("INSERT INTO secrets").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := store.Upsert(context.Background(), secret); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
}

func TestPostgresSecretStoreNotFound(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	cleanupDB(t, db, mock)
	store := NewPostgresSecretStore(db)

	mock.ExpectQuery("SELECT").
		WillReturnError(sql.ErrNoRows)
	if _, err := store.GetByID(context.Background(), "owner", "id"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("GetByID err=%v want ErrSecretNotFound", err)
	}

	mock.ExpectExec("UPDATE secrets").
		WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := store.Update(context.Background(), testSecret()); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("Update err=%v want ErrSecretNotFound", err)
	}

	mock.ExpectExec("UPDATE secrets").
		WillReturnResult(sqlmock.NewResult(0, 0))
	if err := store.Delete(context.Background(), "owner", "id"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("Delete err=%v want ErrSecretNotFound", err)
	}
}

func TestPostgresSecretStoreErrors(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	cleanupDB(t, db, mock)
	store := NewPostgresSecretStore(db)

	secret := testSecret()
	mock.ExpectExec("INSERT INTO secrets").
		WillReturnError(sql.ErrConnDone)
	if _, err := store.Create(context.Background(), secret); err == nil {
		t.Fatalf("expected error on create")
	}

	mock.ExpectQuery("SELECT").
		WillReturnError(sql.ErrConnDone)
	if _, err := store.ListUpdatedSince(context.Background(), "owner", time.Time{}); err == nil {
		t.Fatalf("expected error on list updated")
	}

	mock.ExpectExec("INSERT INTO secrets").
		WillReturnError(sql.ErrConnDone)
	if _, err := store.Upsert(context.Background(), secret); err == nil {
		t.Fatalf("expected error on upsert")
	}
}

func testSecret() models.Secret {
	now := time.Now().UTC()
	return models.Secret{
		ID:        "id1",
		OwnerID:   "owner1",
		Type:      "text",
		Payload:   []byte("payload"),
		Meta:      map[string]string{"a": "b"},
		Deleted:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
