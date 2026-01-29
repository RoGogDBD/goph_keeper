package models

import "time"

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
