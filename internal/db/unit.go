package db

import (
	"database/sql"
	"fmt"

	"github.com/trly/quad-ops/internal/db/model"
)

type UnitRepository struct {
	db *sql.DB
}

func NewUnitRepository(db *sql.DB) *UnitRepository {
	return &UnitRepository{db: db}
}

func (r *UnitRepository) List() ([]model.Unit, error) {
	rows, err := r.db.Query("SELECT id, name, type, sha1_hash, cleanup_policy FROM units")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanUnits(rows)
}

func (r *UnitRepository) Get(id int) (*model.Unit, error) {
	row := r.db.QueryRow("SELECT id, name, type, sha1_hash, cleanup_policy FROM units WHERE id = ?", id)
	units, err := scanUnits(row)
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, fmt.Errorf("unit with id %d not found", id)
	}
	return &units[0], nil
}

func (r *UnitRepository) Create(unit *model.Unit) (*model.Unit, error) {
	result, err := r.db.Exec(`
        INSERT INTO units (name, type, sha1_hash, cleanup_policy) 
        VALUES (?, ?, ?, ?)
        ON CONFLICT(name, type) DO UPDATE SET
            sha1_hash = excluded.sha1_hash,
            cleanup_policy = excluded.cleanup_policy
    `, unit.Name, unit.Type, unit.SHA1Hash, unit.CleanupPolicy)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	unit.ID = int64(id)
	return unit, nil
}

func (r *UnitRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM units WHERE id = ?", id)
	return err
}

func scanUnits(scanner interface{}) ([]model.Unit, error) {
	var units []model.Unit
	switch s := scanner.(type) {
	case *sql.Rows:
		for s.Next() {
			var unit model.Unit
			if err := s.Scan(&unit.ID, &unit.Name, &unit.Type, &unit.SHA1Hash, &unit.CleanupPolicy); err != nil {
				return nil, err
			}
			units = append(units, unit)
		}
	case *sql.Row:
		var unit model.Unit
		if err := s.Scan(&unit.ID, &unit.Name, &unit.Type, &unit.SHA1Hash, &unit.CleanupPolicy); err != nil {
			return nil, err
		}
		units = append(units, unit)
	default:
		return nil, fmt.Errorf("unsupported scanner type: %T", scanner)
	}

	return units, nil
}
