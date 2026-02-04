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

func TestAuthHandlerRegisterErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{name: "register_errors"},
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

			req := httptest.NewRequest(http.MethodGet, "/register", nil)
			rr := httptest.NewRecorder()
			h.Register(rr, req)
			if rr.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte("{}")))
			rr = httptest.NewRecorder()
			h.Register(rr, req)
			if rr.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte("{")))
			req.Header.Set("Content-Type", "application/json")
			rr = httptest.NewRecorder()
			h.Register(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status=%d", rr.Code)
			}

			body, err := json.Marshal(map[string]string{"email": "user@example.com", "password": "pass"})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			req = httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr = httptest.NewRecorder()
			h.Register(rr, req)
			if rr.Code != http.StatusCreated {
				t.Fatalf("status=%d", rr.Code)
			}
			req = httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr = httptest.NewRecorder()
			h.Register(rr, req)
			if rr.Code != http.StatusConflict {
				t.Fatalf("status=%d", rr.Code)
			}
		})
	}
}

func TestAuthHandlerLoginErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{name: "login_errors"},
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

			req := httptest.NewRequest(http.MethodGet, "/login", nil)
			rr := httptest.NewRecorder()
			h.Login(rr, req)
			if rr.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("{}")))
			rr = httptest.NewRecorder()
			h.Login(rr, req)
			if rr.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("status=%d", rr.Code)
			}

			req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("{")))
			req.Header.Set("Content-Type", "application/json")
			rr = httptest.NewRecorder()
			h.Login(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status=%d", rr.Code)
			}

			body, err := json.Marshal(map[string]string{"email": "user@example.com", "password": "pass"})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr = httptest.NewRecorder()
			h.Login(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d", rr.Code)
			}
		})
	}
}

func TestAuthHandlerMeUnauthorized(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{name: "me_unauthorized"},
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

			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			rr := httptest.NewRecorder()
			h.Me(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d", rr.Code)
			}
		})
	}
}
