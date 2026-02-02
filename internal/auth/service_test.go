package auth

import (
	"context"
	"testing"
	"time"

	"goph_keeper/internal/repository"
)

func TestServiceRegisterLogin(t *testing.T) {
	cases := []struct {
		name string
	}{
		{name: "register_and_login"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			store := repository.NewMemoryUserStore()
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
		})
	}
}

func TestServiceInvalidLogin(t *testing.T) {
	cases := []struct {
		name string
	}{
		{name: "invalid_password"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			store := repository.NewMemoryUserStore()
			jwtSvc, err := NewJWTService("secret")
			if err != nil {
				t.Fatalf("jwt init: %v", err)
			}
			service := NewService(store, jwtSvc, time.Hour)

			if _, err := service.Register(context.Background(), "user@example.com", "pass123"); err != nil {
				t.Fatalf("register: %v", err)
			}

			_, err = service.Login(context.Background(), "user@example.com", "wrong")
			if err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}
