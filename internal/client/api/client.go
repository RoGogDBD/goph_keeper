package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"goph_keeper/internal/client/crypto"
)

// Client is an HTTP client for GophKeeper API.
type Client struct {
	baseURL string
	http    *http.Client
	tokens  TokenStore
	crypto  *crypto.Crypto
}

// TokenStore persists auth tokens between runs.
type TokenStore interface {
	Load() (string, error)
	Save(token string) error
}

// New creates a Client with a default HTTP client.
func New(baseURL string, tokens TokenStore, crypto *crypto.Crypto) *Client {
	return NewWithHTTPClient(baseURL, tokens, crypto, nil)
}

// NewWithHTTPClient creates a Client with a custom HTTP client.
func NewWithHTTPClient(baseURL string, tokens TokenStore, crypto *crypto.Crypto, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    httpClient,
		tokens:  tokens,
		crypto:  crypto,
	}
}

// SetCrypto updates the crypto service used for payload encryption.
func (c *Client) SetCrypto(crypto *crypto.Crypto) {
	c.crypto = crypto
}

// Crypto returns the currently configured crypto service.
func (c *Client) Crypto() *crypto.Crypto {
	return c.crypto
}

// RegisterRequest is a registration payload.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is a login payload.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is a login response.
type LoginResponse struct {
	Token string `json:"token"`
}

// SecretRequest is a secret create/update payload.
type SecretRequest struct {
	Type    string            `json:"type"`
	Payload []byte            `json:"payload"`
	Meta    map[string]string `json:"meta"`
}

// SecretResponse is a secret response.
type SecretResponse struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Payload   []byte            `json:"payload"`
	Meta      map[string]string `json:"meta"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// SyncItem represents an item in sync operations.
type SyncItem struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Payload   []byte            `json:"payload"`
	Meta      map[string]string `json:"meta"`
	Deleted   bool              `json:"deleted"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// SyncPullResponse contains items returned by sync pull.
type SyncPullResponse struct {
	Items []SyncItem `json:"items"`
}

// SyncPushRequest contains items sent to sync push.
type SyncPushRequest struct {
	Items []SyncItem `json:"items"`
}

// SyncPushResponse contains count of applied items.
type SyncPushResponse struct {
	Applied int `json:"applied"`
}

// Register registers a new user.
func (c *Client) Register(ctx context.Context, req RegisterRequest) error {
	return c.do(ctx, http.MethodPost, "/api/register", req, nil, false)
}

// Login authenticates a user and stores the token.
func (c *Client) Login(ctx context.Context, req LoginRequest) error {
	var resp LoginResponse
	if err := c.do(ctx, http.MethodPost, "/api/login", req, &resp, false); err != nil {
		return err
	}
	if resp.Token == "" {
		return errors.New("empty token")
	}
	return c.tokens.Save(resp.Token)
}

// CreateSecret creates a secret on the server.
func (c *Client) CreateSecret(ctx context.Context, req SecretRequest) (SecretResponse, error) {
	if c.crypto != nil && len(req.Payload) > 0 {
		enc, err := c.crypto.Encrypt(req.Payload)
		if err != nil {
			return SecretResponse{}, err
		}
		req.Payload = enc
	}
	var resp SecretResponse
	if err := c.do(ctx, http.MethodPost, "/api/secrets", req, &resp, true); err != nil {
		return SecretResponse{}, err
	}
	return resp, nil
}

// ListSecrets lists all secrets for the current user.
func (c *Client) ListSecrets(ctx context.Context) ([]SecretResponse, error) {
	var resp []SecretResponse
	if err := c.do(ctx, http.MethodGet, "/api/secrets", nil, &resp, true); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSecret fetches a secret by ID.
func (c *Client) GetSecret(ctx context.Context, id string) (SecretResponse, error) {
	var resp SecretResponse
	if err := c.do(ctx, http.MethodGet, "/api/secrets/"+id, nil, &resp, true); err != nil {
		return SecretResponse{}, err
	}
	if c.crypto != nil && len(resp.Payload) > 0 {
		dec, err := c.crypto.Decrypt(resp.Payload)
		if err != nil {
			return SecretResponse{}, err
		}
		resp.Payload = dec
	}
	return resp, nil
}

// UpdateSecret updates a secret by ID.
func (c *Client) UpdateSecret(ctx context.Context, id string, req SecretRequest) (SecretResponse, error) {
	if c.crypto != nil && len(req.Payload) > 0 {
		enc, err := c.crypto.Encrypt(req.Payload)
		if err != nil {
			return SecretResponse{}, err
		}
		req.Payload = enc
	}
	var resp SecretResponse
	if err := c.do(ctx, http.MethodPut, "/api/secrets/"+id, req, &resp, true); err != nil {
		return SecretResponse{}, err
	}
	return resp, nil
}

// DeleteSecret removes a secret by ID.
func (c *Client) DeleteSecret(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/secrets/"+id, nil, nil, true)
}

// SyncPull pulls changes and decrypts payloads when crypto is set.
func (c *Client) SyncPull(ctx context.Context, since time.Time) ([]SyncItem, error) {
	return c.syncPull(ctx, since, true)
}

// SyncPullEncrypted pulls changes without decrypting payloads.
func (c *Client) SyncPullEncrypted(ctx context.Context, since time.Time) ([]SyncItem, error) {
	return c.syncPull(ctx, since, false)
}

func (c *Client) syncPull(ctx context.Context, since time.Time, decrypt bool) ([]SyncItem, error) {
	path := "/api/sync"
	if !since.IsZero() {
		path = path + "?since=" + since.UTC().Format(time.RFC3339)
	}
	var resp SyncPullResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &resp, true); err != nil {
		return nil, err
	}
	if decrypt && c.crypto != nil {
		for i := range resp.Items {
			if resp.Items[i].Deleted || len(resp.Items[i].Payload) == 0 {
				continue
			}
			dec, err := c.crypto.Decrypt(resp.Items[i].Payload)
			if err != nil {
				return nil, err
			}
			resp.Items[i].Payload = dec
		}
	}
	return resp.Items, nil
}

// SyncPush pushes changes and encrypts payloads when crypto is set.
func (c *Client) SyncPush(ctx context.Context, items []SyncItem) (int, error) {
	return c.syncPush(ctx, items, true)
}

// SyncPushEncrypted pushes changes without encrypting payloads.
func (c *Client) SyncPushEncrypted(ctx context.Context, items []SyncItem) (int, error) {
	return c.syncPush(ctx, items, false)
}

func (c *Client) syncPush(ctx context.Context, items []SyncItem, encrypt bool) (int, error) {
	if encrypt && c.crypto != nil {
		for i := range items {
			if items[i].Deleted || len(items[i].Payload) == 0 {
				continue
			}
			enc, err := c.crypto.Encrypt(items[i].Payload)
			if err != nil {
				return 0, err
			}
			items[i].Payload = enc
		}
	}
	var resp SyncPushResponse
	if err := c.do(ctx, http.MethodPost, "/api/sync", SyncPushRequest{Items: items}, &resp, true); err != nil {
		return 0, err
	}
	return resp.Applied, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any, auth bool) error {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, buf)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		token, err := c.tokens.Load()
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			_ = cerr
		}
	}()

	if resp.StatusCode >= 400 {
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("request failed: %s", strings.TrimSpace(string(msg)))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return err
		}
	}

	return nil
}
