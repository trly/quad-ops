// Package repository provides data access layer for quad-ops units.
package repository

import (
	"database/sql"
	"fmt"
	"time"
)

// Unit represents a record in the units table.
type Unit struct {
	ID            int64     `db:"id"`
	Name          string    `db:"name"`
	Type          string    `db:"type"`
	CleanupPolicy string    `db:"cleanup_policy"`
	SHA1Hash      []byte    `db:"sha1_hash"`
	UserMode      bool      `db:"user_mode"`
	CreatedAt     time.Time `db:"created_at"` // Set by database, but not updated on every change
}

// Repository defines the interface for unit data access operations.
type Repository interface {
	FindAll() ([]Unit, error)
	FindByUnitType(unitType string) ([]Unit, error)
	FindByID(id int64) (Unit, error)
	Create(unit *Unit) (int64, error)
	Delete(id int64) error
}

// SQLRepository implements Repository interface with SQL database.
type SQLRepository struct {
	db *sql.DB
}

// NewRepository creates a new SQL-based unit repository.
func NewRepository(db *sql.DB) Repository {
	return &SQLRepository{db: db}
}

// FindAll retrieves all units from the database.
func (r *SQLRepository) FindAll() ([]Unit, error) {
	rows, err := r.db.Query("SELECT id, name, type, sha1_hash, cleanup_policy, user_mode, created_at FROM units")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanUnits(rows)
}

// FindByUnitType retrieves units filtered by type.
func (r *SQLRepository) FindByUnitType(unitType string) ([]Unit, error) {
	rows, err := r.db.Query("SELECT id, name, type, sha1_hash, cleanup_policy, user_mode, created_at FROM units WHERE type = ?", unitType)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanUnits(rows)
}

// FindByID retrieves a single unit by ID.
func (r *SQLRepository) FindByID(id int64) (Unit, error) {
	row := r.db.QueryRow("SELECT id, name, type, sha1_hash, cleanup_policy, user_mode, created_at FROM units WHERE id = ?", id)
	units, err := scanUnits(row)
	if err != nil {
		return Unit{}, err
	}
	if len(units) == 0 {
		return Unit{}, fmt.Errorf("unit with id %d not found", id)
	}
	return units[0], nil
}

// Create inserts or updates a unit in the database.
func (r *SQLRepository) Create(unit *Unit) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO units (name, type, sha1_hash, cleanup_policy, user_mode)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(name, type) DO UPDATE SET
		sha1_hash = excluded.sha1_hash,
		cleanup_policy = excluded.cleanup_policy,
		user_mode = excluded.user_mode
	`, unit.Name, unit.Type, unit.SHA1Hash, unit.CleanupPolicy, unit.UserMode)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// Delete removes a unit from the database.
func (r *SQLRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM units WHERE id = ?", id)
	return err
}

// scanUnits is a helper function to scan rows or a single row into Unit structs.
func scanUnits(scanner interface{}) ([]Unit, error) {
	var units []Unit
	switch s := scanner.(type) {
	case *sql.Rows:
		for s.Next() {
			var unit Unit
			if err := s.Scan(&unit.ID, &unit.Name, &unit.Type, &unit.SHA1Hash, &unit.CleanupPolicy, &unit.UserMode, &unit.CreatedAt); err != nil {
				return nil, err
			}
			units = append(units, unit)
		}
	case *sql.Row:
		var unit Unit
		if err := s.Scan(&unit.ID, &unit.Name, &unit.Type, &unit.SHA1Hash, &unit.CleanupPolicy, &unit.UserMode, &unit.CreatedAt); err != nil {
			return nil, err
		}
		units = append(units, unit)
	default:
		return nil, fmt.Errorf("unsupported scanner type: %T", scanner)
	}
	return units, nil
}
