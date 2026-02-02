package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"goph_keeper/internal/client/crypto"
)

type memoryTokenStore struct {
	token string
}

func (s *memoryTokenStore) Load() (string, error) {
	return s.token, nil
}

func (s *memoryTokenStore) Save(token string) error {
	s.token = token
	return nil
}

func TestClientRegisterLogin(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/register":
			w.WriteHeader(http.StatusCreated)
		case "/api/login":
			if err := json.NewEncoder(w).Encode(LoginResponse{Token: "t"}); err != nil {
				t.Fatalf("encode: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)

	store := &memoryTokenStore{}
	cli := New(ts.URL, store, nil)

	if err := cli.Register(context.Background(), RegisterRequest{Email: "a@b.c", Password: "p"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := cli.Login(context.Background(), LoginRequest{Email: "a@b.c", Password: "p"}); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if store.token != "t" {
		t.Fatalf("token=%q want=%q", store.token, "t")
	}
}

func TestClientSecretAndSync(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	crypt, err := crypto.NewCrypto("secret", dir)
	if err != nil {
		t.Fatalf("NewCrypto: %v", err)
	}

	var lastAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastAuth = r.Header.Get("Authorization")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/login":
			if err := json.NewEncoder(w).Encode(LoginResponse{Token: "t"}); err != nil {
				t.Fatalf("encode: %v", err)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/api/secrets":
			var req SecretRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			resp := SecretResponse{
				ID:        "id1",
				Type:      req.Type,
				Payload:   req.Payload,
				Meta:      req.Meta,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("encode: %v", err)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/api/secrets/id1":
			enc, err := crypt.Encrypt([]byte("hello"))
			if err != nil {
				t.Fatalf("encrypt: %v", err)
			}
			resp := SecretResponse{
				ID:        "id1",
				Type:      "text",
				Payload:   enc,
				Meta:      map[string]string{"a": "b"},
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("encode: %v", err)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/api/secrets":
			resp := []SecretResponse{{
				ID:        "id1",
				Type:      "text",
				Payload:   []byte("p"),
				Meta:      map[string]string{},
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("encode: %v", err)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/api/secrets/id1":
			var req SecretRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			resp := SecretResponse{
				ID:        "id1",
				Type:      req.Type,
				Payload:   req.Payload,
				Meta:      req.Meta,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("encode: %v", err)
			}
		case r.Method == http.MethodDelete && r.URL.Path == "/api/secrets/id1":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/sync":
			enc, err := crypt.Encrypt([]byte("hi"))
			if err != nil {
				t.Fatalf("encrypt: %v", err)
			}
			resp := SyncPullResponse{
				Items: []SyncItem{{
					ID:        "id1",
					Type:      "text",
					Payload:   enc,
					Meta:      map[string]string{},
					Deleted:   false,
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}},
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("encode: %v", err)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/api/sync":
			if err := json.NewEncoder(w).Encode(SyncPushResponse{Applied: 1}); err != nil {
				t.Fatalf("encode: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)

	store := &memoryTokenStore{token: "t"}
	cli := New(ts.URL, store, crypt)
	cli.SetCrypto(crypt)
	if cli.Crypto() == nil {
		t.Fatalf("Crypto is nil")
	}

	secret, err := cli.CreateSecret(context.Background(), SecretRequest{
		Type:    "text",
		Payload: []byte("hello"),
		Meta:    map[string]string{"a": "b"},
	})
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	if secret.ID == "" {
		t.Fatalf("CreateSecret missing id")
	}
	if !strings.HasPrefix(lastAuth, "Bearer ") {
		t.Fatalf("auth header not set")
	}

	got, err := cli.GetSecret(context.Background(), "id1")
	if err != nil {
		t.Fatalf("GetSecret: %v", err)
	}
	if string(got.Payload) != "hello" {
		t.Fatalf("GetSecret payload=%q want=%q", got.Payload, "hello")
	}

	if _, err := cli.ListSecrets(context.Background()); err != nil {
		t.Fatalf("ListSecrets: %v", err)
	}

	if _, err := cli.UpdateSecret(context.Background(), "id1", SecretRequest{Type: "text", Payload: []byte("p")}); err != nil {
		t.Fatalf("UpdateSecret: %v", err)
	}

	if err := cli.DeleteSecret(context.Background(), "id1"); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}

	items, err := cli.SyncPull(context.Background(), time.Time{})
	if err != nil {
		t.Fatalf("SyncPull: %v", err)
	}
	if len(items) != 1 || string(items[0].Payload) != "hi" {
		t.Fatalf("SyncPull payload=%q", items[0].Payload)
	}

	applied, err := cli.SyncPush(context.Background(), items)
	if err != nil {
		t.Fatalf("SyncPush: %v", err)
	}
	if applied != 1 {
		t.Fatalf("SyncPush applied=%d want=1", applied)
	}

	if _, err := cli.SyncPushEncrypted(context.Background(), items); err != nil {
		t.Fatalf("SyncPushEncrypted: %v", err)
	}
}

func TestClientErrorResponse(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"error":"bad"}`)); err != nil {
			t.Fatalf("write: %v", err)
		}
	}))
	t.Cleanup(ts.Close)

	cli := New(ts.URL, &memoryTokenStore{}, nil)
	if err := cli.Register(context.Background(), RegisterRequest{Email: "a@b.c", Password: "p"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSyncPullEncrypted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	crypt, err := crypto.NewCrypto("secret", dir)
	if err != nil {
		t.Fatalf("NewCrypto: %v", err)
	}
	enc, err := crypt.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SyncPullResponse{
			Items: []SyncItem{{
				ID:        "id1",
				Type:      "text",
				Payload:   enc,
				Meta:      map[string]string{},
				Deleted:   false,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}},
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}))
	t.Cleanup(ts.Close)

	cli := New(ts.URL, &memoryTokenStore{token: "t"}, crypt)
	items, err := cli.SyncPullEncrypted(context.Background(), time.Time{})
	if err != nil {
		t.Fatalf("SyncPullEncrypted: %v", err)
	}
	if len(items) != 1 || string(items[0].Payload) != string(enc) {
		t.Fatalf("payload was decrypted unexpectedly")
	}
}
