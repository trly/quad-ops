// internal/db/db.go
package db

import (
	"database/sql"
	"embed"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/trly/quad-ops/internal/config"

	// Register migrate's sqlite3 driver
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"

	// Register sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func GetConnectionString(cfg config.Config) string {
	return "sqlite3://" + cfg.DBPath
}

func Connect() (*sql.DB, error) {
	// Remove sqlite3:// prefix if present for direct SQL connection
	dbPath := strings.TrimPrefix(config.GetConfig().DBPath, "sqlite3://")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	if config.GetConfig().Verbose {
		log.Printf("Connected to database at %s", dbPath)
	}

	return db, nil
}

func Up(cfg config.Config) error {
	m, err := getMigrationInstance(cfg)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	} else if config.GetConfig().Verbose {
		if err == migrate.ErrNoChange {
			log.Println("[database] no new migrations to apply")
		} else {
			log.Println("[database] migrations applied successfully")
		}
	}

	return nil
}

func Down(cfg config.Config) error {
	m, err := getMigrationInstance(cfg)
	if err != nil {
		return err
	}
	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return err
	} else if config.GetConfig().Verbose {
		if err == migrate.ErrNoChange {
			log.Println("[database] no new migrations to apply")
		} else {
			log.Println("[database] migrations applied successfully")
		}
	}

	return nil
}
func getMigrationInstance(cfg config.Config) (*migrate.Migrate, error) {
	dbConnStr := GetConnectionString(cfg)
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, dbConnStr)
	if err != nil {
		return nil, err
	}

	// Enable verbose logging if requested
	if config.GetConfig().Verbose {
		m.Log = &migrationLogger{verbose: config.GetConfig().Verbose}
	}

	return m, nil
}

type migrationLogger struct {
	verbose bool
}

func (l *migrationLogger) Printf(format string, v ...interface{}) {
	if l.verbose {
		log.Printf("[migration] "+format, v...)
	}
}

func (l *migrationLogger) Verbose() bool {
	return l.verbose
}
