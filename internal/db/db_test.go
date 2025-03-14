package db

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/config"
)

func TestGetConnectionString(t *testing.T) {
	cfg := config.Config{
		DBPath: "/test/path/db.sqlite",
	}
	expected := "sqlite3:///test/path/db.sqlite"
	assert.Equal(t, expected, GetConnectionString(cfg))
}

func TestConnect(t *testing.T) {
	// Create temp db file
	tmpDB, err := os.CreateTemp("", "test.*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpDB.Name())

	// Set up test config
	testConfig := &config.Config{
		DBPath:  tmpDB.Name(),
		Verbose: true,
	}
	config.SetConfig(testConfig)

	// Test connection
	db, err := Connect()
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Verify connection works
	err = db.Ping()
	assert.NoError(t, err)

	db.Close()
}

func TestConnectError(t *testing.T) {
	testConfig := &config.Config{
		DBPath: "/nonexistent/path/db.sqlite",
	}
	config.SetConfig(testConfig)

	db, err := Connect()
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestMigrations(t *testing.T) {
	// Create temp db file
	tmpDB, err := os.CreateTemp("", "test.*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpDB.Name())

	testConfig := config.Config{
		DBPath:  tmpDB.Name(),
		Verbose: true,
	}

	// Test Up migration
	err = Up(testConfig)
	assert.NoError(t, err)

	// Test Down migration
	err = Down(testConfig)
	assert.NoError(t, err)
}

func TestMigrationsWithInvalidPath(t *testing.T) {
	testConfig := config.Config{
		DBPath:  "/nonexistent/path/db.sqlite",
		Verbose: true,
	}

	// Test Up migration with invalid path
	err := Up(testConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open database")

	// Test Down migration with invalid path - we expect an error
	err = Down(testConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open database")
}
