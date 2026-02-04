package models

import "time"

// User represents a registered user.
type User struct {
	ID           string
	Email        string
	PasswordHash []byte
	CreatedAt    time.Time
}
