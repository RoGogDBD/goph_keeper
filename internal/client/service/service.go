package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"goph_keeper/internal/client/api"
	"goph_keeper/internal/client/crypto"
	"goph_keeper/internal/client/store"
)

// Service coordinates client-side operations.
type Service struct {
	api       *api.Client
	local     *store.LocalStore
	syncStore *store.SyncStore
	crypto    *crypto.Crypto
	nowFn     func() time.Time
	idFn      func() (string, error)
}

// New creates a client service.
func New(apiClient *api.Client, local *store.LocalStore, syncStore *store.SyncStore, cryptoSvc *crypto.Crypto) *Service {
	return &Service{
		api:       apiClient,
		local:     local,
		syncStore: syncStore,
		crypto:    cryptoSvc,
		nowFn:     func() time.Time { return time.Now().UTC() },
		idFn:      NewID,
	}
}

// Register registers a new user.
func (s *Service) Register(ctx context.Context, email, password string) error {
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return errors.New("email and password are required")
	}
	return s.api.Register(ctx, api.RegisterRequest{Email: email, Password: password})
}

// Login authenticates a user.
func (s *Service) Login(ctx context.Context, email, password string) error {
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return errors.New("email and password are required")
	}
	return s.api.Login(ctx, api.LoginRequest{Email: email, Password: password})
}

// Add creates a new local secret.
func (s *Service) Add(ctx context.Context, secretType, payload, meta string) (string, error) {
	secretType = strings.TrimSpace(secretType)
	if secretType == "" || payload == "" {
		return "", errors.New("type and payload are required")
	}
	if s.crypto == nil {
		return "", errors.New("master password is required")
	}

	id, err := s.idFn()
	if err != nil {
		return "", err
	}

	now := s.nowFn()
	item := store.Item{
		ID:        id,
		Type:      secretType,
		Payload:   []byte(payload),
		Meta:      store.ParseMeta(meta),
		Deleted:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	encPayload, err := s.crypto.Encrypt(item.Payload)
	if err != nil {
		return "", err
	}
	item.Payload = encPayload

	if err := s.local.Upsert(ctx, item, true); err != nil {
		return "", err
	}
	return item.ID, nil
}

// List returns all local secrets.
func (s *Service) List(ctx context.Context) ([]store.Item, error) {
	return s.local.List(ctx, false)
}

// Get returns a local secret by id, decrypting its payload.
func (s *Service) Get(ctx context.Context, id string) (store.Item, error) {
	if strings.TrimSpace(id) == "" {
		return store.Item{}, errors.New("id is required")
	}
	secret, err := s.local.Get(ctx, id)
	if err != nil {
		return store.Item{}, err
	}
	if s.crypto == nil {
		return store.Item{}, errors.New("master password is required")
	}
	if len(secret.Payload) > 0 {
		dec, err := s.crypto.Decrypt(secret.Payload)
		if err != nil {
			return store.Item{}, err
		}
		secret.Payload = dec
	}
	return secret, nil
}

// Update modifies a local secret.
func (s *Service) Update(ctx context.Context, id, secretType, payload, meta string) (string, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(secretType) == "" || payload == "" {
		return "", errors.New("id, type and payload are required")
	}
	if s.crypto == nil {
		return "", errors.New("master password is required")
	}

	createdAt := s.nowFn()
	if existing, err := s.local.Get(ctx, id); err == nil {
		createdAt = existing.CreatedAt
	}

	item := store.Item{
		ID:        id,
		Type:      strings.TrimSpace(secretType),
		Payload:   []byte(payload),
		Meta:      store.ParseMeta(meta),
		Deleted:   false,
		CreatedAt: createdAt,
		UpdatedAt: s.nowFn(),
	}

	enc, err := s.crypto.Encrypt(item.Payload)
	if err != nil {
		return "", err
	}
	item.Payload = enc

	if err := s.local.Upsert(ctx, item, true); err != nil {
		return "", err
	}
	return item.ID, nil
}

// Delete marks a local secret as deleted.
func (s *Service) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("id is required")
	}

	createdAt := s.nowFn()
	itemType := "deleted"
	var payload []byte
	if existing, err := s.local.Get(ctx, id); err == nil {
		createdAt = existing.CreatedAt
		itemType = existing.Type
		payload = existing.Payload
	}

	item := store.Item{
		ID:        id,
		Type:      itemType,
		Payload:   payload,
		Deleted:   true,
		CreatedAt: createdAt,
		UpdatedAt: s.nowFn(),
	}
	return s.local.Upsert(ctx, item, true)
}

// Sync pushes dirty local items and pulls remote updates.
func (s *Service) Sync(ctx context.Context) ([]api.SyncItem, error) {
	since, err := s.syncStore.Load()
	if err != nil && !errors.Is(err, store.ErrSyncNotFound) {
		return nil, err
	}

	dirty, err := s.local.ListDirty(ctx)
	if err != nil {
		return nil, err
	}
	if len(dirty) > 0 {
		if _, err := s.api.SyncPushEncrypted(ctx, ToSyncItems(dirty)); err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(dirty))
		for _, item := range dirty {
			ids = append(ids, item.ID)
		}
		if err := s.local.MarkClean(ctx, ids); err != nil {
			return nil, err
		}
	}

	items, err := s.api.SyncPullEncrypted(ctx, since)
	if err != nil {
		return nil, err
	}

	if err := s.local.ApplyRemote(ctx, FromSyncItems(items)); err != nil {
		return nil, err
	}

	if err := s.syncStore.Save(s.nowFn()); err != nil {
		return nil, err
	}

	return items, nil
}

// NewID generates a new random identifier.
func NewID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ToSyncItems maps local items to API sync items.
func ToSyncItems(items []store.Item) []api.SyncItem {
	out := make([]api.SyncItem, 0, len(items))
	for _, it := range items {
		out = append(out, api.SyncItem{
			ID:        it.ID,
			Type:      it.Type,
			Payload:   it.Payload,
			Meta:      it.Meta,
			Deleted:   it.Deleted,
			CreatedAt: it.CreatedAt,
			UpdatedAt: it.UpdatedAt,
		})
	}
	return out
}

// FromSyncItems maps API sync items to local items.
func FromSyncItems(items []api.SyncItem) []store.Item {
	out := make([]store.Item, 0, len(items))
	for _, it := range items {
		out = append(out, store.Item{
			ID:        it.ID,
			Type:      it.Type,
			Payload:   it.Payload,
			Meta:      it.Meta,
			Deleted:   it.Deleted,
			CreatedAt: it.CreatedAt,
			UpdatedAt: it.UpdatedAt,
		})
	}
	return out
}
