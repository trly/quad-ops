package unit

import (
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewUnitRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindAll() ([]Unit, error) {
	rows, err := r.db.Query("SELECT id, name, type, sha1_hash, cleanup_policy FROM units")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanUnits(rows)
}

func (r *Repository) FindByUnitType(unitType string) ([]Unit, error) {
	rows, err := r.db.Query("SELECT id, name, type, sha1_hash, cleanup_policy FROM units WHERE type = ?", unitType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanUnits(rows)
}

func (r *Repository) FindByID(id int64) (Unit, error) {
	row := r.db.QueryRow("SELECT id, name, type, sha1_hash, cleanup_policy FROM units WHERE id = ?", id)
	units, err := scanUnits(row)
	if err != nil {
		return Unit{}, err // Return zero value instead of nil
	}
	if len(units) == 0 {
		return Unit{}, fmt.Errorf("unit with id %d not found", id) // Return zero value
	}
	return units[0], nil // Return the value, not a pointer
}

func (r *Repository) Create(unit *Unit) (int64, error) {
	result, err := r.db.Exec(`
    INSERT INTO units (name, type, sha1_hash, cleanup_policy)
    VALUES (?, ?, ?, ?)
    ON CONFLICT(name, type) DO UPDATE SET
    sha1_hash = excluded.sha1_hash,
    cleanup_policy = excluded.cleanup_policy
    `, unit.Name, unit.Type, unit.SHA1Hash, unit.CleanupPolicy)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *Repository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM units WHERE id = ?", id)
	return err
}

func scanUnits(scanner interface{}) ([]Unit, error) {
	var units []Unit
	switch s := scanner.(type) {
	case *sql.Rows:
		for s.Next() {
			var unit Unit
			if err := s.Scan(&unit.ID, &unit.Name, &unit.Type, &unit.SHA1Hash, &unit.CleanupPolicy); err != nil {
				return nil, err
			}
			units = append(units, unit)
		}
	case *sql.Row:
		var unit Unit
		if err := s.Scan(&unit.ID, &unit.Name, &unit.Type, &unit.SHA1Hash, &unit.CleanupPolicy); err != nil {
			return nil, err
		}
		units = append(units, unit)
	default:
		return nil, fmt.Errorf("unsupported scanner type: %T", scanner)
	}

	return units, nil
}
