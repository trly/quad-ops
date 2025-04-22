package unit

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
)

func TestUpdateUnitDatabaseCleanupPolicy(t *testing.T) {
	// Create a test database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create units table
	_, err = db.Exec(`
		CREATE TABLE units (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			sha1_hash BLOB NOT NULL,
			cleanup_policy TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(name, type)
		);
	`)
	require.NoError(t, err)

	// Create a repository with config
	unitRepo := NewUnitRepository(db)

	// Setup test config with repositories
	cfg := &config.Config{
		Repositories: []config.RepositoryConfig{
			{
				Name:    "test-repo",
				Cleanup: "delete",
			},
			{
				Name:    "keep-repo",
				Cleanup: "keep",
			},
			{
				Name: "default-repo", // no cleanup policy specified
			},
		},
	}
	config.SetConfig(cfg)

	// Test cases
	tests := []struct {
		name           string
		unitName       string
		expectedPolicy string
	}{
		{"Delete policy", "test-repo-service", "delete"},
		{"Keep policy", "keep-repo-service", "keep"},
		{"Default policy", "default-repo-service", "keep"},
		{"Unknown repo", "unknown-service", "keep"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a unit with the test name
			unit := &QuadletUnit{
				Name: tc.unitName,
				Type: "container",
			}

			// Update the database
			err := updateUnitDatabase(unitRepo, unit, "test content")
			require.NoError(t, err)

			// Query the database to verify the cleanup policy
			var policy string
			err = db.QueryRow("SELECT cleanup_policy FROM units WHERE name = ?", tc.unitName).Scan(&policy)
			require.NoError(t, err)

			// Assert the cleanup policy matches the expected value
			assert.Equal(t, tc.expectedPolicy, policy)
		})
	}
}