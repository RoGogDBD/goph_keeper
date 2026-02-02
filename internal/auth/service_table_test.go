package auth

import (
	"context"
	"testing"
	"time"

	"goph_keeper/internal/repository"
)

func TestServiceRegisterValidation(t *testing.T) {
	t.Parallel()

	jwtSvc, err := NewJWTService("secret")
	if err != nil {
		t.Fatalf("jwt init: %v", err)
	}

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  bool
	}{
		{name: "empty email", email: "", password: "pass", wantErr: true},
		{name: "empty password", email: "user@example.com", password: "", wantErr: true},
		{name: "both empty", email: "", password: "", wantErr: true},
		{name: "trimmed email", email: "  user@example.com  ", password: "pass", wantErr: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := repository.NewMemoryUserStore()
			service := NewService(store, jwtSvc, time.Hour)

			_, err := service.Register(context.Background(), tt.email, tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Register() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceLoginValidation(t *testing.T) {
	t.Parallel()

	jwtSvc, err := NewJWTService("secret")
	if err != nil {
		t.Fatalf("jwt init: %v", err)
	}

	tests := []struct {
		name     string
		email    string
		password string
		setup    func(svc *Service) error
		wantErr  bool
	}{
		{
			name:     "unknown user",
			email:    "missing@example.com",
			password: "pass",
			setup:    func(*Service) error { return nil },
			wantErr:  true,
		},
		{
			name:     "wrong password",
			email:    "user@example.com",
			password: "bad",
			setup: func(svc *Service) error {
				_, err := svc.Register(context.Background(), "user@example.com", "pass")
				return err
			},
			wantErr: true,
		},
		{
			name:     "valid credentials",
			email:    "user@example.com",
			password: "pass",
			setup: func(svc *Service) error {
				_, err := svc.Register(context.Background(), "user@example.com", "pass")
				return err
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := repository.NewMemoryUserStore()
			service := NewService(store, jwtSvc, time.Hour)
			if err := tt.setup(service); err != nil {
				t.Fatalf("setup: %v", err)
			}

			_, err := service.Login(context.Background(), tt.email, tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Login() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
