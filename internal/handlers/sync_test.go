package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goph_keeper/internal/auth"
	"goph_keeper/internal/models"
)

type fakeSyncStore struct {
	items []models.Secret
}

func (f *fakeSyncStore) ListUpdatedSince(_ context.Context, ownerID string, since time.Time) ([]models.Secret, error) {
	var out []models.Secret
	for _, s := range f.items {
		if s.OwnerID == ownerID && s.UpdatedAt.After(since) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeSyncStore) Upsert(_ context.Context, secret models.Secret) (models.Secret, error) {
	f.items = append(f.items, secret)
	return secret, nil
}

func TestSyncHandlerPullPush(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{name: "pull_push"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeSyncStore{
				items: []models.Secret{{
					ID:        "id1",
					OwnerID:   "user1",
					Type:      "text",
					Payload:   []byte("payload"),
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}},
			}
			h := NewSyncHandler(store)
			claims := auth.Claims{UserID: "user1", Email: "u@example.com"}

			req := httptest.NewRequest(http.MethodGet, "/api/sync", nil)
			req = req.WithContext(withClaims(req.Context(), claims))
			rr := httptest.NewRecorder()
			h.Pull(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("Pull status=%d", rr.Code)
			}

			body, err := json.Marshal(map[string]any{
				"items": []map[string]any{{
					"id":         "id2",
					"type":       "text",
					"payload":    []byte("p"),
					"meta":       map[string]string{},
					"deleted":    false,
					"created_at": time.Now().UTC(),
					"updated_at": time.Now().UTC(),
				}},
			})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			req = httptest.NewRequest(http.MethodPost, "/api/sync", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(withClaims(req.Context(), claims))
			rr = httptest.NewRecorder()
			h.Push(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("Push status=%d", rr.Code)
			}
		})
	}
}
