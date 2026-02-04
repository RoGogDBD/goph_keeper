package models

import "time"

// Secret represents a stored secret payload.
type Secret struct {
	ID        string
	OwnerID   string
	Type      string
	Payload   []byte
	Meta      map[string]string
	Deleted   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
