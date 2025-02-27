package model

import (
	"time"
)

// Unit represents a record in the units table
type Unit struct {
	ID            int64     `db:"id"`
	Name          string    `db:"name"`
	Type          string    `db:"type"`
	CleanupPolicy string    `db:"cleanup_policy"`
	SHA1Hash      []byte    `db:"sha1_hash"`
	CreatedAt     time.Time `db:"created_at"`
}
