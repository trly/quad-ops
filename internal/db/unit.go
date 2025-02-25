package db

import (
	"database/sql"
	"quad-ops/internal/db/model"
)

type UnitRepository struct {
	db *sql.DB
}

func NewUnitRepository(db *sql.DB) *UnitRepository {
	return &UnitRepository{db: db}
}

func (r *UnitRepository) List() ([]model.Unit, error) {
	rows, err := r.db.Query("SELECT * FROM units")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToUnits(rows)
}

func (r *UnitRepository) Get(id int) (*model.Unit, error) {
	row := r.db.QueryRow("SELECT * FROM units WHERE id = ?", id)
	return rowToUnit(row)
}

func (r *UnitRepository) Create(unit *model.Unit) (*model.Unit, error) {
	result, err := r.db.Exec("INSERT INTO units (name, type, cleanup_policy) VALUES (?, ?, ?)", unit.Name, unit.Type, unit.CleanupPolicy)
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

func rowToUnit(row *sql.Row) (*model.Unit, error) {
	var unit model.Unit
	err := row.Scan(&unit.ID, &unit.Name, &unit.Type, &unit.CleanupPolicy, &unit.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &unit, nil
}

func rowsToUnits(rows *sql.Rows) ([]model.Unit, error) {
	var units []model.Unit
	for rows.Next() {
		var unit model.Unit
		if err := rows.Scan(&unit.ID, &unit.Name, &unit.Type, &unit.CleanupPolicy, &unit.CreatedAt); err != nil {
			return nil, err
		}
		units = append(units, unit)
	}
	return units, nil
}
