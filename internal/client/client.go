package client

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
)

type Client struct {
	baseURL string
	http    *http.Client
	tokens  *TokenStore
}

func New(baseURL string, tokens *TokenStore) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
		tokens: tokens,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type SecretRequest struct {
	Type    string            `json:"type"`
	Payload []byte            `json:"payload"`
	Meta    map[string]string `json:"meta"`
}

type SecretResponse struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Payload   []byte            `json:"payload"`
	Meta      map[string]string `json:"meta"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type SyncItem struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Payload   []byte            `json:"payload"`
	Meta      map[string]string `json:"meta"`
	Deleted   bool              `json:"deleted"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type SyncPullResponse struct {
	Items []SyncItem `json:"items"`
}

type SyncPushRequest struct {
	Items []SyncItem `json:"items"`
}

type SyncPushResponse struct {
	Applied int `json:"applied"`
}

func (c *Client) Register(ctx context.Context, req RegisterRequest) error {
	return c.do(ctx, http.MethodPost, "/api/register", req, nil, false)
}

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

func (c *Client) CreateSecret(ctx context.Context, req SecretRequest) (SecretResponse, error) {
	var resp SecretResponse
	if err := c.do(ctx, http.MethodPost, "/api/secrets", req, &resp, true); err != nil {
		return SecretResponse{}, err
	}
	return resp, nil
}

func (c *Client) ListSecrets(ctx context.Context) ([]SecretResponse, error) {
	var resp []SecretResponse
	if err := c.do(ctx, http.MethodGet, "/api/secrets", nil, &resp, true); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) GetSecret(ctx context.Context, id string) (SecretResponse, error) {
	var resp SecretResponse
	if err := c.do(ctx, http.MethodGet, "/api/secrets/"+id, nil, &resp, true); err != nil {
		return SecretResponse{}, err
	}
	return resp, nil
}

func (c *Client) UpdateSecret(ctx context.Context, id string, req SecretRequest) (SecretResponse, error) {
	var resp SecretResponse
	if err := c.do(ctx, http.MethodPut, "/api/secrets/"+id, req, &resp, true); err != nil {
		return SecretResponse{}, err
	}
	return resp, nil
}

func (c *Client) DeleteSecret(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/secrets/"+id, nil, nil, true)
}

func (c *Client) SyncPull(ctx context.Context, since time.Time) ([]SyncItem, error) {
	path := "/api/sync"
	if !since.IsZero() {
		path = path + "?since=" + since.UTC().Format(time.RFC3339)
	}
	var resp SyncPullResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &resp, true); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *Client) SyncPush(ctx context.Context, items []SyncItem) (int, error) {
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
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %s", strings.TrimSpace(string(msg)))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return err
		}
	}

	return nil
}
