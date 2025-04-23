// Package repository provides database operations for Quadlet units
package repository

import (
	"database/sql"
	"fmt"

	"github.com/trly/quad-ops/internal/unit/model"
)

// Repository defines the interface for unit data access operations.
type Repository interface {
	FindAll() ([]model.Unit, error)
	FindByUnitType(unitType string) ([]model.Unit, error)
	FindByID(id int64) (model.Unit, error)
	Create(unit *model.Unit) (int64, error)
	Delete(id int64) error
}

// SQLRepository implements Repository interface with SQL database.
type SQLRepository struct {
	db *sql.DB
}

// NewUnitRepository creates a new SQL-based unit repository.
func NewUnitRepository(db *sql.DB) Repository {
	return &SQLRepository{db: db}
}

// FindAll retrieves all units from the database.
func (r *SQLRepository) FindAll() ([]model.Unit, error) {
	rows, err := r.db.Query("SELECT id, name, type, sha1_hash, cleanup_policy, user_mode, repository_id, created_at FROM units")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanUnits(rows)
}

// FindByUnitType retrieves units filtered by type.
func (r *SQLRepository) FindByUnitType(unitType string) ([]model.Unit, error) {
	rows, err := r.db.Query("SELECT id, name, type, sha1_hash, cleanup_policy, user_mode, repository_id, created_at FROM units WHERE type = ?", unitType)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanUnits(rows)
}

// FindByID retrieves a single unit by ID.
func (r *SQLRepository) FindByID(id int64) (model.Unit, error) {
	row := r.db.QueryRow("SELECT id, name, type, sha1_hash, cleanup_policy, user_mode, repository_id, created_at FROM units WHERE id = ?", id)
	units, err := scanUnits(row)
	if err != nil {
		return model.Unit{}, err
	}
	if len(units) == 0 {
		return model.Unit{}, fmt.Errorf("unit with id %d not found", id)
	}
	return units[0], nil
}

// Create inserts or updates a unit in the database.
func (r *SQLRepository) Create(unit *model.Unit) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO units (name, type, sha1_hash, cleanup_policy, user_mode, repository_id)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(name, type) DO UPDATE SET
		sha1_hash = excluded.sha1_hash,
		cleanup_policy = excluded.cleanup_policy,
		user_mode = excluded.user_mode,
		repository_id = excluded.repository_id
	`, unit.Name, unit.Type, unit.SHA1Hash, unit.CleanupPolicy, unit.UserMode, unit.RepositoryID)
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
func scanUnits(scanner interface{}) ([]model.Unit, error) {
	var units []model.Unit
	switch s := scanner.(type) {
	case *sql.Rows:
		for s.Next() {
			var unit model.Unit
			if err := s.Scan(&unit.ID, &unit.Name, &unit.Type, &unit.SHA1Hash, &unit.CleanupPolicy, &unit.UserMode, &unit.RepositoryID, &unit.CreatedAt); err != nil {
				return nil, err
			}
			units = append(units, unit)
		}
	case *sql.Row:
		var unit model.Unit
		if err := s.Scan(&unit.ID, &unit.Name, &unit.Type, &unit.SHA1Hash, &unit.CleanupPolicy, &unit.UserMode, &unit.RepositoryID, &unit.CreatedAt); err != nil {
			return nil, err
		}
		units = append(units, unit)
	default:
		return nil, fmt.Errorf("unsupported scanner type: %T", scanner)
	}
	return units, nil
}
