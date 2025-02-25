// internal/db/db.go
package db

import (
	"embed"
	"log"
	"os"
	"quad-ops/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func GetConnectionString(cfg config.Config) string {
	return "sqlite3://" + cfg.DBPath
}

func Up(cfg config.Config) error {
	m, err := getMigrationInstance(cfg)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	} else if cfg.Verbose {
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
		log.Fatalf("[database] could not initialize migrations: %v", err)
		os.Exit(1)
	}
	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return err
	} else if cfg.Verbose {
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
	if cfg.Verbose {
		m.Log = &migrationLogger{verbose: cfg.Verbose}
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
