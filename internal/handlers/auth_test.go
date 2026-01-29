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
	store := repository.NewMemoryUserStore()
	jwtSvc, _ := auth.NewJWTService("secret")
	authSvc := auth.NewService(store, jwtSvc, time.Hour)
	h := NewAuthHandler(authSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("/register", h.Register)
	mux.HandleFunc("/login", h.Login)

	registerBody, _ := json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": "pass123",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(registerBody))
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}

	loginBody, _ := json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": "pass123",
	})
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	resp = httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}
