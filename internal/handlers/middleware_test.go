package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goph_keeper/internal/auth"
)

func TestAuthMiddleware(t *testing.T) {
	t.Parallel()

	jwtSvc, err := auth.NewJWTService("secret")
	if err != nil {
		t.Fatalf("NewJWTService: %v", err)
	}
	token, err := jwtSvc.IssueToken("user1", "u@example.com", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := AuthMiddleware(jwtSvc)(next)

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{name: "missing", header: "", want: http.StatusUnauthorized},
		{name: "invalid format", header: "bad", want: http.StatusUnauthorized},
		{name: "invalid token", header: "Bearer bad", want: http.StatusUnauthorized},
		{name: "ok", header: "Bearer " + token, want: http.StatusOK},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("status=%d want=%d", rr.Code, tt.want)
			}
		})
	}
}
