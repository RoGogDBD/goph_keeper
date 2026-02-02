package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goph_keeper/internal/auth"
)

func TestSecretHandlerErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{name: "errors"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeSecretStore{}
			h := NewSecretHandler(store)
			claims := auth.Claims{UserID: "user1", Email: "u@example.com"}

			req := httptest.NewRequest(http.MethodGet, "/api/secrets", nil)
			rr := httptest.NewRecorder()
			h.Create(rr, req)
			if rr.Code != http.StatusMethodNotAllowed {
				t.Fatalf("Create status=%d", rr.Code)
			}

			body, err := json.Marshal(map[string]any{"type": "text", "payload": []byte("p")})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			req = httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader(body))
			rr = httptest.NewRecorder()
			h.Create(rr, req)
			if rr.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("Create status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr = httptest.NewRecorder()
			h.Create(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("Create status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader([]byte("{")))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(withClaims(req.Context(), claims))
			rr = httptest.NewRecorder()
			h.Create(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("Create status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodGet, "/api/secrets", nil)
			rr = httptest.NewRecorder()
			h.List(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("List status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodPut, "/api/secrets/id1", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr = httptest.NewRecorder()
			h.Update(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("Update status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodDelete, "/api/secrets/id1", nil)
			rr = httptest.NewRecorder()
			h.Delete(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("Delete status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodGet, "/api/secrets/id1", nil)
			req = req.WithContext(withClaims(req.Context(), claims))
			req = addURLParam(req, "id", "id1")
			rr = httptest.NewRecorder()
			h.Get(rr, req)
			if rr.Code != http.StatusNotFound {
				t.Fatalf("Get status=%d", rr.Code)
			}
		})
	}
}
