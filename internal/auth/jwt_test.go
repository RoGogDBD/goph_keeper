package auth

import (
	"testing"
	"time"
)

func TestJWTServiceIssueParse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{name: "issue_and_parse"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			svc, err := NewJWTService("secret")
			if err != nil {
				t.Fatalf("NewJWTService: %v", err)
			}
			token, err := svc.IssueToken("id1", "u@example.com", time.Minute)
			if err != nil {
				t.Fatalf("IssueToken: %v", err)
			}
			claims, err := svc.ParseToken(token)
			if err != nil {
				t.Fatalf("ParseToken: %v", err)
			}
			if claims.UserID != "id1" {
				t.Fatalf("UserID=%q", claims.UserID)
			}
		})
	}
}

func TestJWTServiceInvalidToken(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		token string
	}{
		{name: "invalid_token", token: "bad"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			svc, err := NewJWTService("secret")
			if err != nil {
				t.Fatalf("NewJWTService: %v", err)
			}
			if _, err := svc.ParseToken(tc.token); err == nil {
				t.Fatalf("expected error for invalid token")
			}
		})
	}
}
