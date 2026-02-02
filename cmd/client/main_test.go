package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goph_keeper/internal/client/api"
	"goph_keeper/internal/client/crypto"
	"goph_keeper/internal/client/store"
	"goph_keeper/internal/config"
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

func TestBuildCryptoAndHTTPClient(t *testing.T) {
	t.Parallel()

	if c, err := buildCrypto("", t.TempDir(), "list"); err != nil || c != nil {
		t.Fatalf("buildCrypto list err=%v c=%v", err, c)
	}
	if _, err := buildCrypto("", t.TempDir(), "add"); err == nil {
		t.Fatalf("expected error for empty master password")
	}

	cfg := config.ClientConfig{
		ServerURL: "https://example.com",
		DataDir:   t.TempDir(),
		Log:       config.LogConfig{Level: "info", Format: "text"},
	}
	cfg.TLS = false
	if _, err := buildHTTPClient(cfg); err != nil {
		t.Fatalf("buildHTTPClient: %v", err)
	}

	cfg.TLS = true
	cfg.TLSCA = ""
	if _, err := buildHTTPClient(cfg); err != nil {
		t.Fatalf("buildHTTPClient tls: %v", err)
	}

	cfg.TLSCA = "/no/such/file"
	if _, err := buildHTTPClient(cfg); err == nil {
		t.Fatalf("expected error for invalid tls_ca")
	}
}

func TestClientHandlers(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/register":
			w.WriteHeader(http.StatusCreated)
		case "/api/login":
			if err := json.NewEncoder(w).Encode(api.LoginResponse{Token: "t"}); err != nil {
				t.Fatalf("encode: %v", err)
			}
		case "/api/sync":
			if r.Method == http.MethodGet {
				if err := json.NewEncoder(w).Encode(api.SyncPullResponse{Items: []api.SyncItem{}}); err != nil {
					t.Fatalf("encode: %v", err)
				}
				return
			}
			if err := json.NewEncoder(w).Encode(api.SyncPushResponse{Applied: 0}); err != nil {
				t.Fatalf("encode: %v", err)
			}
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	local, err := store.NewLocalStore(dir)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	t.Cleanup(func() {
		if err := local.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	crypt, err := crypto.NewCrypto("secret", dir)
	if err != nil {
		t.Fatalf("NewCrypto: %v", err)
	}

	tokenStore := &memoryTokenStore{token: "t"}
	cli := api.New(ts.URL, tokenStore, crypt)

	if err := handleRegister(context.Background(), cli, []string{"--email", "a@b.c", "--password", "p"}); err != nil {
		t.Fatalf("handleRegister: %v", err)
	}
	if err := handleLogin(context.Background(), cli, []string{"--email", "a@b.c", "--password", "p"}); err != nil {
		t.Fatalf("handleLogin: %v", err)
	}

	if err := handleAdd(context.Background(), local, crypt, []string{"--type", "text", "--payload", "payload", "--meta", "k=v"}); err != nil {
		t.Fatalf("handleAdd: %v", err)
	}
	items, err := local.List(context.Background(), false)
	if err != nil || len(items) != 1 {
		t.Fatalf("expected item")
	}

	if err := handleList(context.Background(), local); err != nil {
		t.Fatalf("handleList: %v", err)
	}
	if err := handleGet(context.Background(), local, crypt, []string{"--id", items[0].ID}); err != nil {
		t.Fatalf("handleGet: %v", err)
	}
	if err := handleUpdate(context.Background(), local, crypt, []string{"--id", items[0].ID, "--type", "text", "--payload", "new"}); err != nil {
		t.Fatalf("handleUpdate: %v", err)
	}
	if err := handleDelete(context.Background(), local, []string{"--id", items[0].ID}); err != nil {
		t.Fatalf("handleDelete: %v", err)
	}

	if err := handleSync(context.Background(), cli, local, dir); err != nil {
		t.Fatalf("handleSync: %v", err)
	}
}

func TestSyncItemConversion(t *testing.T) {
	t.Parallel()

	item := store.Item{
		ID:        "id1",
		Type:      "text",
		Payload:   []byte("p"),
		Meta:      map[string]string{"a": "b"},
		Deleted:   false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	out := toSyncItems([]store.Item{item})
	back := fromSyncItems(out)
	if len(back) != 1 || back[0].ID != item.ID {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestNewID(t *testing.T) {
	t.Parallel()
	if len(newID()) != 32 {
		t.Fatalf("newID length mismatch")
	}
}

func TestPrintClientUsage(t *testing.T) {
	t.Parallel()
	printClientUsage()
}

func TestRun(t *testing.T) {
	t.Setenv("GOPHKEEPER_CLIENT_DATA_DIR", t.TempDir())

	if err := run([]string{"version"}); err != nil {
		t.Fatalf("run version: %v", err)
	}
	if err := run([]string{}); err == nil {
		t.Fatalf("expected usage error")
	}
	if err := run([]string{"list"}); err != nil {
		t.Fatalf("run list: %v", err)
	}
}
