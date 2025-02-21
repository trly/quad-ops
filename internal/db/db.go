// internal/db/db.go
package db

import (
	"database/sql"
	"embed"
	"log"
	"quad-ops/internal/db/model"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func GetMigrationInstance(dbConnStr string, verbose bool) (*migrate.Migrate, error) {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, dbConnStr)
	if err != nil {
		return nil, err
	}

	// Enable verbose logging if requested
	if verbose {
		m.Log = &migrationLogger{verbose: verbose}
	}

	return m, nil
}

type migrationLogger struct {
	verbose bool
}

func (l *migrationLogger) Printf(format string, v ...interface{}) {
	if l.verbose {
		log.Printf("[Migration] "+format, v...)
	}
}

func (l *migrationLogger) Verbose() bool {
	return l.verbose
}
