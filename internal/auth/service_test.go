package auth

import (
	"context"
	"testing"
	"time"

	"goph_keeper/internal/storage"
)

func TestServiceRegisterLogin(t *testing.T) {
	store := storage.NewMemoryUserStore()
	jwtSvc, err := NewJWTService("secret")
	if err != nil {
		t.Fatalf("jwt init: %v", err)
	}
	service := NewService(store, jwtSvc, time.Hour)

	_, err = service.Register(context.Background(), "user@example.com", "pass123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	token, err := service.Login(context.Background(), "user@example.com", "pass123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token == "" {
		t.Fatalf("token is empty")
	}
}

func TestServiceInvalidLogin(t *testing.T) {
	store := storage.NewMemoryUserStore()
	jwtSvc, _ := NewJWTService("secret")
	service := NewService(store, jwtSvc, time.Hour)

	_, _ = service.Register(context.Background(), "user@example.com", "pass123")

	_, err := service.Login(context.Background(), "user@example.com", "wrong")
	if err == nil {
		t.Fatalf("expected error")
	}
}
