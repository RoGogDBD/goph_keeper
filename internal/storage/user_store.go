package storage

import (
	"context"
	"errors"

	"goph_keeper/internal/models"
)

var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
)

type UserStore interface {
	Create(ctx context.Context, user models.User) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
}
