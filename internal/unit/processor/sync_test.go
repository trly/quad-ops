package processor

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/unit/model"
	"github.com/trly/quad-ops/internal/unit/repository"
)

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

	// Create test table with repository_id column
	_, err = db.Exec(`CREATE TABLE units (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR,
		type VARCHAR,
		cleanup_policy VARCHAR,
		sha1_hash BLOB,
		user_mode BOOLEAN DEFAULT 0,
		repository_id INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE UNIQUE INDEX unique_name_type ON units(name, type);`)
	assert.NoError(t, err)

	// Create repository with mocked units
	repo := repository.NewUnitRepository(db)

	// 1. Test userMode tracking
	cfg := &config.Config{
		UserMode: false,
	}
	config.SetConfig(cfg)

	// Create a unit with userMode=false
	unit := &model.Unit{
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
	systemdUnit := &model.Unit{
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
	nonSystemdUnit := &model.Unit{
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