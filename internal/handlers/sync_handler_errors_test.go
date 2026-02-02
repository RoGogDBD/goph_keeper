package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goph_keeper/internal/auth"
)

func TestSyncHandlerErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "pull_invalid_since",
			fn: func(t *testing.T) {
				store := &fakeSyncStore{}
				h := NewSyncHandler(store)
				claims := auth.Claims{UserID: "user1", Email: "u@example.com"}

				req := httptest.NewRequest(http.MethodGet, "/api/sync?since=bad", nil)
				req = req.WithContext(withClaims(req.Context(), claims))
				rr := httptest.NewRecorder()
				h.Pull(rr, req)
				if rr.Code != http.StatusBadRequest {
					t.Fatalf("Pull status=%d", rr.Code)
				}
			},
		},
		{
			name: "push_missing_content_type",
			fn: func(t *testing.T) {
				store := &fakeSyncStore{}
				h := NewSyncHandler(store)
				req := httptest.NewRequest(http.MethodPost, "/api/sync", bytes.NewReader([]byte("{}")))
				rr := httptest.NewRecorder()
				h.Push(rr, req)
				if rr.Code != http.StatusUnsupportedMediaType {
					t.Fatalf("Push status=%d", rr.Code)
				}
			},
		},
		{
			name: "push_invalid_item",
			fn: func(t *testing.T) {
				store := &fakeSyncStore{}
				h := NewSyncHandler(store)
				claims := auth.Claims{UserID: "user1", Email: "u@example.com"}

				body, err := json.Marshal(map[string]any{"items": []map[string]any{{"id": "", "type": "text", "payload": []byte("p"), "created_at": time.Now().UTC(), "updated_at": time.Now().UTC()}}})
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/api/sync", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req = req.WithContext(withClaims(req.Context(), claims))
				rr := httptest.NewRecorder()
				h.Push(rr, req)
				if rr.Code != http.StatusBadRequest {
					t.Fatalf("Push status=%d", rr.Code)
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, tc.fn)
	}
}
