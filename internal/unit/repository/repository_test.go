package repository

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/unit/model"
)

func TestCreate(t *testing.T) {
	// Setup mock database
	db, mock := setupTestDB()
	defer teardownTestDB(db)

	r := NewUnitRepository(db)

	// Prepare test data
	unit := &model.Unit{
		Name:          "test-unit",
		Type:          "pod",
		SHA1Hash:      []byte("abc123"),
		CleanupPolicy: "delete",
		UserMode:      false,
	}

	// Expect the INSERT query with repository_id
	mock.ExpectExec(`INSERT INTO units`).WithArgs(
		unit.Name, unit.Type, unit.SHA1Hash, unit.CleanupPolicy, unit.UserMode, unit.RepositoryID,
	).WillReturnResult(sqlmock.NewResult(1, 1))

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
	units := []model.Unit{
		{ID: 1, Name: "unit1", Type: "pod"},
		{ID: 2, Name: "unit2", Type: "service"},
	}

	// Expect SELECT query with repository_id
	mock.ExpectQuery("SELECT id, name, type, sha1_hash, cleanup_policy, user_mode, repository_id, created_at FROM units").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "type", "sha1_hash", "cleanup_policy", "user_mode", "repository_id", "created_at"}).
			AddRow(units[0].ID, units[0].Name, units[0].Type, units[0].SHA1Hash, units[0].CleanupPolicy, units[0].UserMode, units[0].RepositoryID, units[0].CreatedAt).
			AddRow(units[1].ID, units[1].Name, units[1].Type, units[1].SHA1Hash, units[1].CleanupPolicy, units[1].UserMode, units[1].RepositoryID, units[1].CreatedAt),
	)

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
	expectedUnits := []model.Unit{
		{ID: 1, Name: "unit1", Type: unitType},
	}

	// Expect SELECT query with WHERE clause and repository_id
	mock.ExpectQuery("SELECT id, name, type, sha1_hash, cleanup_policy, user_mode, repository_id, created_at FROM units WHERE type = \\?").WithArgs(
		unitType,
	).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "type", "sha1_hash", "cleanup_policy", "user_mode", "repository_id", "created_at"}).
			AddRow(expectedUnits[0].ID, expectedUnits[0].Name, expectedUnits[0].Type, expectedUnits[0].SHA1Hash,
				expectedUnits[0].CleanupPolicy, expectedUnits[0].UserMode, expectedUnits[0].RepositoryID, expectedUnits[0].CreatedAt),
	)

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
	unitID := int64(1)
	expectedUnit := model.Unit{
		ID:   unitID,
		Name: "test-unit",
		Type: "pod",
	}

	// Expect SELECT query with WHERE clause and repository_id
	mock.ExpectQuery("SELECT id, name, type, sha1_hash, cleanup_policy, user_mode, repository_id, created_at FROM units WHERE id = \\?").WithArgs(
		unitID,
	).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "type", "sha1_hash", "cleanup_policy", "user_mode", "repository_id", "created_at"}).
			AddRow(expectedUnit.ID, expectedUnit.Name, expectedUnit.Type, expectedUnit.SHA1Hash,
				expectedUnit.CleanupPolicy, expectedUnit.UserMode, expectedUnit.RepositoryID, expectedUnit.CreatedAt),
	)

	// Test FindById method
	result, err := r.FindByID(unitID)

	assert.NoError(t, err)
	assert.Equal(t, expectedUnit, result) // Compare value to value, not pointer to value
}

func TestDelete(t *testing.T) {
	db, mock := setupTestDB()
	defer teardownTestDB(db)

	r := NewUnitRepository(db)

	// Prepare test data
	unitID := int64(1)

	// Expect DELETE query
	mock.ExpectExec("DELETE FROM units WHERE id = \\?").WithArgs(
		unitID,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	// Test Delete method
	err := r.Delete(unitID)
	assert.NoError(t, err)
}

func setupTestDB() (*sql.DB, sqlmock.Sqlmock) {
	db, mock, _ := sqlmock.New()
	return db, mock
}

func teardownTestDB(db *sql.DB) {
	_ = db.Close()
}
