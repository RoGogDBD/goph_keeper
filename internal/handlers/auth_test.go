package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goph_keeper/internal/auth"
	"goph_keeper/internal/repository"
)

func TestAuthHandlers(t *testing.T) {
	cases := []struct {
		name string
	}{
		{name: "register_and_login"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			store := repository.NewMemoryUserStore()
			jwtSvc, err := auth.NewJWTService("secret")
			if err != nil {
				t.Fatalf("jwt init: %v", err)
			}
			authSvc := auth.NewService(store, jwtSvc, time.Hour)
			h := NewAuthHandler(authSvc)

			mux := http.NewServeMux()
			mux.HandleFunc("/register", h.Register)
			mux.HandleFunc("/login", h.Login)

			registerBody, err := json.Marshal(map[string]string{
				"email":    "user@example.com",
				"password": "pass123",
			})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(registerBody))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)
			if resp.Code != http.StatusCreated {
				t.Fatalf("expected 201, got %d", resp.Code)
			}

			loginBody, err := json.Marshal(map[string]string{
				"email":    "user@example.com",
				"password": "pass123",
			})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
			req.Header.Set("Content-Type", "application/json")
			resp = httptest.NewRecorder()
			mux.ServeHTTP(resp, req)
			if resp.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", resp.Code)
			}
		})
	}
}
