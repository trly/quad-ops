// Package db provides database functionality for quad-ops.
package db

import (
	"database/sql"
	"embed"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/log"

	// Register migrate's sqlite3 driver.
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"

	// Register sqlite3 driver.
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// GetConnectionString returns the database connection string.
func GetConnectionString(cfg config.Settings) string {
	return "sqlite3://" + cfg.DBPath
}

// Connect establishes a connection to the database.
func Connect() (*sql.DB, error) {
	// Remove sqlite3:// prefix if present for direct SQL connection
	dbPath := strings.TrimPrefix(config.DefaultProvider().GetConfig().DBPath, "sqlite3://")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	log.GetLogger().Info("Connected to database", "path", dbPath)

	return db, nil
}

// Up runs database migrations to latest version.
func Up(cfg config.Settings) error {
	m, err := getMigrationInstance(cfg)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

// Down rolls back all database migrations.
func Down(cfg config.Settings) error {
	m, err := getMigrationInstance(cfg)
	if err != nil {
		return err
	}
	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}
func getMigrationInstance(cfg config.Settings) (*migrate.Migrate, error) {
	dbConnStr := GetConnectionString(cfg)
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, dbConnStr)
	if err != nil {
		return nil, err
	}

	// Set up migration logger
	m.Log = &migrationLogger{}

	return m, nil
}

type migrationLogger struct{}

func (l *migrationLogger) Printf(format string, v ...interface{}) {
	log.GetLogger().Debug("Migration: "+format, v...)
}

func (l *migrationLogger) Verbose() bool {
	return true
}
