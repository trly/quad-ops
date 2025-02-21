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
	CreatedAt     time.Time `db:"created_at"`
}
