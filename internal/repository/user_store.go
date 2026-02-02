package repository

import (
	"context"
	"errors"

	"goph_keeper/internal/models"
)

var (
	// ErrUserExists indicates a user already exists.
	ErrUserExists = errors.New("user already exists")
	// ErrUserNotFound indicates a user was not found.
	ErrUserNotFound = errors.New("user not found")
)

// UserStore defines operations for user persistence.
type UserStore interface {
	Create(ctx context.Context, user models.User) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
}
