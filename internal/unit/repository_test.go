package unit

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	// Setup mock database
	db, mock := setupTestDB()
	defer teardownTestDB(db)

	r := NewUnitRepository(db)

	// Prepare test data
	unit := &Unit{
		Name:          "test-unit",
		Type:          "pod",
		SHA1Hash:      []byte("abc123"),
		CleanupPolicy: "delete",
	}

	// Expect the INSERT query
	mock.ExpectExec(`INSERT INTO units`).
		WithArgs(unit.Name, unit.Type, unit.SHA1Hash, unit.CleanupPolicy).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Test Create method
	id, err := r.Create(unit)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), id)
}

func TestFindAll(t *testing.T) {
	db, mock := setupTestDB()
	defer teardownTestDB(db)

	r := NewUnitRepository(db)

	// Prepare test data
	units := []Unit{
		{ID: 1, Name: "unit1", Type: "pod"},
		{ID: 2, Name: "unit2", Type: "service"},
	}

	// Expect SELECT query
	mock.ExpectQuery("SELECT id, name, type, sha1_hash, cleanup_policy FROM units").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type", "sha1_hash", "cleanup_policy"}).
			AddRow(units[0].ID, units[0].Name, units[0].Type, units[0].SHA1Hash, units[0].CleanupPolicy).
			AddRow(units[1].ID, units[1].Name, units[1].Type, units[1].SHA1Hash, units[1].CleanupPolicy))

	// Test FindAll method
	result, err := r.FindAll()
	assert.NoError(t, err)
	assert.Equal(t, units, result)
}

func TestFindByUnitType(t *testing.T) {
	db, mock := setupTestDB()
	defer teardownTestDB(db)

	r := NewUnitRepository(db)

	// Prepare test data
	unitType := "pod"
	expectedUnits := []Unit{
		{ID: 1, Name: "unit1", Type: unitType},
	}

	// Expect SELECT query with WHERE clause
	mock.ExpectQuery("SELECT id, name, type, sha1_hash, cleanup_policy FROM units WHERE type = ?").
		WithArgs(unitType).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type", "sha1_hash", "cleanup_policy"}).
			AddRow(expectedUnits[0].ID, expectedUnits[0].Name, expectedUnits[0].Type,
				expectedUnits[0].SHA1Hash, expectedUnits[0].CleanupPolicy))

	// Test FindByUnitType method
	result, err := r.FindByUnitType(unitType)
	assert.NoError(t, err)
	assert.Equal(t, expectedUnits, result)
}

func TestFindById(t *testing.T) {
	db, mock := setupTestDB()
	defer teardownTestDB(db)

	r := NewUnitRepository(db)

	// Prepare test data
	unitId := int64(1)
	expectedUnit := Unit{
		ID:   unitId,
		Name: "test-unit",
		Type: "pod",
	}

	// Expect SELECT query with WHERE clause
	mock.ExpectQuery("SELECT id, name, type, sha1_hash, cleanup_policy FROM units WHERE id = ?").
		WithArgs(unitId).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type", "sha1_hash", "cleanup_policy"}).
			AddRow(expectedUnit.ID, expectedUnit.Name, expectedUnit.Type,
				expectedUnit.SHA1Hash, expectedUnit.CleanupPolicy))

	// Test FindById method
	result, err := r.FindById(unitId)

	assert.NoError(t, err)
	assert.Equal(t, expectedUnit, result) // Compare value to value, not pointer to value
}

func TestDelete(t *testing.T) {
	db, mock := setupTestDB()
	defer teardownTestDB(db)

	r := NewUnitRepository(db)

	// Prepare test data
	unitId := int64(1)

	// Expect DELETE query
	mock.ExpectExec("DELETE FROM units WHERE id = ?").
		WithArgs(unitId).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Test Delete method
	err := r.Delete(unitId)
	assert.NoError(t, err)
}

func setupTestDB() (*sql.DB, sqlmock.Sqlmock) {
	db, mock, _ := sqlmock.New()
	return db, mock
}

func teardownTestDB(db *sql.DB) {
	db.Close()
}
