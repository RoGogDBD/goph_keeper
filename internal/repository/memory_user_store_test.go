package repository

import (
	"context"
	"testing"
	"time"

	"goph_keeper/internal/models"
)

func TestMemoryUserStore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "create_get_duplicate",
			fn: func(t *testing.T) {
				store := NewMemoryUserStore()
				user := models.User{
					ID:           "id1",
					Email:        "user@example.com",
					PasswordHash: []byte("hash"),
					CreatedAt:    time.Now().UTC(),
				}

				if _, err := store.Create(context.Background(), user); err != nil {
					t.Fatalf("Create: %v", err)
				}

				got, err := store.GetByEmail(context.Background(), user.Email)
				if err != nil {
					t.Fatalf("GetByEmail: %v", err)
				}
				if got.Email != user.Email {
					t.Fatalf("GetByEmail email=%q want=%q", got.Email, user.Email)
				}

				if _, err := store.Create(context.Background(), user); err == nil {
					t.Fatalf("expected ErrUserExists")
				}
			},
		},
		{
			name: "missing_user",
			fn: func(t *testing.T) {
				store := NewMemoryUserStore()
				if _, err := store.GetByEmail(context.Background(), "missing@example.com"); err == nil {
					t.Fatalf("expected ErrUserNotFound")
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, tc.fn)
	}
}
