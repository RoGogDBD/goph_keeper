package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goph_keeper/internal/auth"
	"goph_keeper/internal/models"
	"goph_keeper/internal/repository"
)

type routerSecretStore struct{}

func (r *routerSecretStore) Create(_ context.Context, secret models.Secret) (models.Secret, error) {
	return secret, nil
}
func (r *routerSecretStore) GetByID(_ context.Context, ownerID, id string) (models.Secret, error) {
	return models.Secret{}, repository.ErrSecretNotFound
}
func (r *routerSecretStore) ListByOwner(_ context.Context, ownerID string) ([]models.Secret, error) {
	return []models.Secret{}, nil
}
func (r *routerSecretStore) ListUpdatedSince(_ context.Context, ownerID string, since time.Time) ([]models.Secret, error) {
	return []models.Secret{}, nil
}
func (r *routerSecretStore) Update(_ context.Context, secret models.Secret) (models.Secret, error) {
	return secret, nil
}
func (r *routerSecretStore) Delete(_ context.Context, ownerID, id string) error {
	return nil
}
func (r *routerSecretStore) Upsert(_ context.Context, secret models.Secret) (models.Secret, error) {
	return secret, nil
}

func TestNewRouter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{name: "router_swagger"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			userStore := repository.NewMemoryUserStore()
			secretStore := &routerSecretStore{}
			jwtSvc, err := auth.NewJWTService("secret")
			if err != nil {
				t.Fatalf("NewJWTService: %v", err)
			}
			router, err := NewRouter(userStore, secretStore, jwtSvc)
			if err != nil {
				t.Fatalf("NewRouter: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusMovedPermanently {
				t.Fatalf("swagger status=%d", rr.Code)
			}
		})
	}
}
