package store

import "time"

// Item mirrors the secret sync payload for local storage.
// Item is a locally stored secret.
type Item struct {
	ID        string
	Type      string
	Payload   []byte
	Meta      map[string]string
	Deleted   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
