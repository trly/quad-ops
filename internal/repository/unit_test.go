package repository

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/config"

	// Register sqlite3 driver for testing.
	_ "github.com/mattn/go-sqlite3"
)

func TestCreate(t *testing.T) {
	// Setup mock database
	db, mock := setupTestDB()
	defer teardownTestDB(db)

	r := NewRepository(db)

	// Prepare test data
	unit := &Unit{
		Name:          "test-unit",
		Type:          "pod",
		SHA1Hash:      []byte("abc123"),
		CleanupPolicy: "delete",
		UserMode:      false,
	}

	// Expect the INSERT query
	mock.ExpectExec(`INSERT INTO units`).
		WithArgs(unit.Name, unit.Type, unit.SHA1Hash, unit.CleanupPolicy, unit.UserMode).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Test Create method
	id, err := r.Create(unit)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), id)
}

func TestFindAll(t *testing.T) {
	db, mock := setupTestDB()
	defer teardownTestDB(db)

	r := NewRepository(db)

	// Prepare test data
	units := []Unit{
		{ID: 1, Name: "unit1", Type: "pod"},
		{ID: 2, Name: "unit2", Type: "service"},
	}

	// Expect SELECT query
	mock.ExpectQuery("SELECT id, name, type, sha1_hash, cleanup_policy, user_mode, created_at FROM units").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type", "sha1_hash", "cleanup_policy", "user_mode", "created_at"}).
			AddRow(units[0].ID, units[0].Name, units[0].Type, units[0].SHA1Hash, units[0].CleanupPolicy, units[0].UserMode, units[0].CreatedAt).
			AddRow(units[1].ID, units[1].Name, units[1].Type, units[1].SHA1Hash, units[1].CleanupPolicy, units[1].UserMode, units[1].CreatedAt))

	// Test FindAll method
	result, err := r.FindAll()
	assert.NoError(t, err)
	assert.Equal(t, units, result)
}

func TestFindByUnitType(t *testing.T) {
	db, mock := setupTestDB()
	defer teardownTestDB(db)

	r := NewRepository(db)

	// Prepare test data
	unitType := "pod"
	expectedUnits := []Unit{
		{ID: 1, Name: "unit1", Type: unitType},
	}

	// Expect SELECT query with WHERE clause
	mock.ExpectQuery("SELECT id, name, type, sha1_hash, cleanup_policy, user_mode, created_at FROM units WHERE type = ?").
		WithArgs(unitType).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type", "sha1_hash", "cleanup_policy", "user_mode", "created_at"}).
			AddRow(expectedUnits[0].ID, expectedUnits[0].Name, expectedUnits[0].Type,
				expectedUnits[0].SHA1Hash, expectedUnits[0].CleanupPolicy, expectedUnits[0].UserMode, expectedUnits[0].CreatedAt))

	// Test FindByUnitType method
	result, err := r.FindByUnitType(unitType)
	assert.NoError(t, err)
	assert.Equal(t, expectedUnits, result)
}

func TestFindById(t *testing.T) {
	db, mock := setupTestDB()
	defer teardownTestDB(db)

	r := NewRepository(db)

	// Prepare test data
	unitID := int64(1)
	expectedUnit := Unit{
		ID:   unitID,
		Name: "test-unit",
		Type: "pod",
	}

	// Expect SELECT query with WHERE clause
	mock.ExpectQuery("SELECT id, name, type, sha1_hash, cleanup_policy, user_mode, created_at FROM units WHERE id = ?").
		WithArgs(unitID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type", "sha1_hash", "cleanup_policy", "user_mode", "created_at"}).
			AddRow(expectedUnit.ID, expectedUnit.Name, expectedUnit.Type,
				expectedUnit.SHA1Hash, expectedUnit.CleanupPolicy, expectedUnit.UserMode, expectedUnit.CreatedAt))

	// Test FindById method
	result, err := r.FindByID(unitID)

	assert.NoError(t, err)
	assert.Equal(t, expectedUnit, result) // Compare value to value, not pointer to value
}

func TestDelete(t *testing.T) {
	db, mock := setupTestDB()
	defer teardownTestDB(db)

	r := NewRepository(db)

	// Prepare test data
	unitID := int64(1)

	// Expect DELETE query
	mock.ExpectExec("DELETE FROM units WHERE id = ?").
		WithArgs(unitID).
		WillReturnResult(sqlmock.NewResult(1, 1))

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

// TestUserModeHandling tests the user mode tracking and cleanup functionality.
func TestUserModeHandling(t *testing.T) {
	// Setup test database
	tempDir, err := os.MkdirTemp("", "quad-ops-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create test table
	_, err = db.Exec(`CREATE TABLE units (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR,
		type VARCHAR,
		cleanup_policy VARCHAR,
		sha1_hash BLOB,
		user_mode BOOLEAN DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
CREATE UNIQUE INDEX unique_name_type ON units(name, type);`)
	assert.NoError(t, err)

	// Create repository with mocked units
	repo := NewRepository(db)

	// 1. Test userMode tracking
	cfg := &config.Settings{
		UserMode: false,
	}
	config.SetConfig(cfg)

	// Create a unit with userMode=false
	unit := &Unit{
		Name:          "test-container",
		Type:          "container",
		SHA1Hash:      []byte("test-hash"),
		CleanupPolicy: "keep",
		UserMode:      false,
	}
	id, err := repo.Create(unit)
	assert.NoError(t, err)
	assert.Greater(t, id, int64(0))

	// Retrieve the unit to confirm it was saved with the correct userMode
	retrievedUnit, err := repo.FindByID(id)
	assert.NoError(t, err)
	assert.Equal(t, false, retrievedUnit.UserMode)

	// 2. Switch userMode and verify handling
	cfg.UserMode = true

	// The next Create call should update the userMode
	unit.UserMode = true
	id2, err := repo.Create(unit)
	assert.NoError(t, err)

	// Retrieve again to verify the userMode was updated
	retrievedUnit, err = repo.FindByID(id2)
	assert.NoError(t, err)
	assert.Equal(t, true, retrievedUnit.UserMode)

	// 3. Test usePodmanDefaultNames conflict detection
	// Create a mock unit that would be from usePodmanDefaultNames=true (with systemd- prefix)
	systemdUnit := &Unit{
		Name:          "systemd-test-db",
		Type:          "container",
		SHA1Hash:      []byte("systemd-hash"),
		CleanupPolicy: "keep",
		UserMode:      true,
	}
	id3, err := repo.Create(systemdUnit)
	assert.NoError(t, err)
	assert.Greater(t, id3, int64(0))

	// Now create a similar unit but without the systemd- prefix
	nonSystemdUnit := &Unit{
		Name:          "test-db",
		Type:          "container",
		SHA1Hash:      []byte("non-systemd-hash"),
		CleanupPolicy: "keep",
		UserMode:      true,
	}
	id4, err := repo.Create(nonSystemdUnit)
	assert.NoError(t, err)
	assert.Greater(t, id4, int64(0))

	// Both units should exist in the database
	allUnits, err := repo.FindAll()
	assert.NoError(t, err)

	// Count how many of our test units exist
	hasSystemdUnit := false
	hasNonSystemdUnit := false
	for _, u := range allUnits {
		if u.Name == "systemd-test-db" && u.Type == "container" {
			hasSystemdUnit = true
		}
		if u.Name == "test-db" && u.Type == "container" {
			hasNonSystemdUnit = true
		}
	}
	assert.True(t, hasSystemdUnit, "systemd-prefixed unit should exist")
	assert.True(t, hasNonSystemdUnit, "non-systemd-prefixed unit should exist")
}
