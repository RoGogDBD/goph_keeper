package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"goph_keeper/internal/models"
	"goph_keeper/internal/storage"
)

type Service struct {
	store storage.UserStore
	jwt   *JWTService
	ttl   time.Duration
}

func NewService(store storage.UserStore, jwt *JWTService, ttl time.Duration) *Service {
	return &Service{store: store, jwt: jwt, ttl: ttl}
}

var ErrInvalidCredentials = errors.New("invalid credentials")

func (s *Service) Register(ctx context.Context, email, password string) (models.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return models.User{}, errors.New("email and password are required")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return models.User{}, err
	}

	id, err := newID()
	if err != nil {
		return models.User{}, err
	}

	user := models.User{
		ID:           id,
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}

	return s.store.Create(ctx, user)
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.store.GetByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	if err != nil {
		return "", ErrInvalidCredentials
	}
	if err := ComparePassword(user.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials
	}

	return s.jwt.IssueToken(user.ID, user.Email, s.ttl)
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
