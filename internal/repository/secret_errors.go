package repository

import "errors"

// ErrSecretNotFound indicates a missing secret.
var ErrSecretNotFound = errors.New("secret not found")
