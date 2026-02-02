package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"goph_keeper/internal/auth"
	"goph_keeper/internal/models"
	"goph_keeper/internal/repository"
)

type fakeSecretStore struct {
	createErr error
	updateErr error
	deleteErr error
	getErr    error
	listErr   error

	secret models.Secret
}

func (f *fakeSecretStore) Create(_ context.Context, secret models.Secret) (models.Secret, error) {
	if f.createErr != nil {
		return models.Secret{}, f.createErr
	}
	f.secret = secret
	return secret, nil
}

func (f *fakeSecretStore) GetByID(_ context.Context, ownerID, id string) (models.Secret, error) {
	if f.getErr != nil {
		return models.Secret{}, f.getErr
	}
	if f.secret.ID == id && f.secret.OwnerID == ownerID && !f.secret.Deleted {
		return f.secret, nil
	}
	return models.Secret{}, repository.ErrSecretNotFound
}

func (f *fakeSecretStore) ListByOwner(_ context.Context, ownerID string) ([]models.Secret, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.secret.OwnerID == ownerID && !f.secret.Deleted {
		return []models.Secret{f.secret}, nil
	}
	return []models.Secret{}, nil
}

func (f *fakeSecretStore) Update(_ context.Context, secret models.Secret) (models.Secret, error) {
	if f.updateErr != nil {
		return models.Secret{}, f.updateErr
	}
	f.secret = secret
	return secret, nil
}

func (f *fakeSecretStore) Delete(_ context.Context, ownerID, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if f.secret.ID == id && f.secret.OwnerID == ownerID {
		f.secret.Deleted = true
		return nil
	}
	return repository.ErrSecretNotFound
}

func TestSecretHandlerCreateListGet(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{name: "create_list_get"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeSecretStore{}
			h := NewSecretHandler(store)

			claims := auth.Claims{UserID: "user1", Email: "u@example.com"}
			body, err := json.Marshal(map[string]any{
				"type":    "text",
				"payload": []byte("payload"),
				"meta":    map[string]string{"a": "b"},
			})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(withClaims(req.Context(), claims))
			rr := httptest.NewRecorder()
			h.Create(rr, req)
			if rr.Code != http.StatusCreated {
				t.Fatalf("Create status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodGet, "/api/secrets", nil)
			req = req.WithContext(withClaims(req.Context(), claims))
			rr = httptest.NewRecorder()
			h.List(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("List status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodGet, "/api/secrets/id1", nil)
			req = req.WithContext(withClaims(req.Context(), claims))
			req = addURLParam(req, "id", store.secret.ID)
			rr = httptest.NewRecorder()
			h.Get(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("Get status=%d", rr.Code)
			}
		})
	}
}

func TestSecretHandlerUpdateDelete(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{name: "update_delete"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeSecretStore{
				secret: models.Secret{
					ID:        "id1",
					OwnerID:   "user1",
					Type:      "text",
					Payload:   []byte("old"),
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				},
			}
			h := NewSecretHandler(store)
			claims := auth.Claims{UserID: "user1", Email: "u@example.com"}

			body, err := json.Marshal(map[string]any{
				"type":    "text",
				"payload": []byte("new"),
			})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			req := httptest.NewRequest(http.MethodPut, "/api/secrets/id1", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(withClaims(req.Context(), claims))
			req = addURLParam(req, "id", "id1")
			rr := httptest.NewRecorder()
			h.Update(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("Update status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodDelete, "/api/secrets/id1", nil)
			req = req.WithContext(withClaims(req.Context(), claims))
			req = addURLParam(req, "id", "id1")
			rr = httptest.NewRecorder()
			h.Delete(rr, req)
			if rr.Code != http.StatusNoContent {
				t.Fatalf("Delete status=%d", rr.Code)
			}
		})
	}
}

func withClaims(ctx context.Context, claims auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func addURLParam(r *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
}
