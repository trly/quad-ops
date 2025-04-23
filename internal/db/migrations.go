package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/trly/quad-ops/internal/config"
)

// ApplyRepositoryMigration handles the repositories table migration with current config.
// This will be called from a migration callback to properly populate the repositories table.
func ApplyRepositoryMigration(db *sql.DB) error {
	if config.GetConfig().Verbose {
		log.Println("Applying repository migration with current config")
	}

	// Start with truncating the placeholder entry
	_, err := db.Exec("DELETE FROM repositories WHERE name = 'migration_placeholder'")
	if err != nil {
		return fmt.Errorf("error clearing placeholder repository: %w", err)
	}

	// Insert all repositories from the current configuration
	for _, repo := range config.GetConfig().Repositories {
		_, err := db.Exec(
			"INSERT INTO repositories (name, url, reference, compose_dir, cleanup_policy, use_podman_default_names) VALUES (?, ?, ?, ?, ?, ?)",
			repo.Name, repo.URL, repo.Reference, repo.ComposeDir, repo.Cleanup, repo.UsePodmanDefaultNames,
		)
		if err != nil {
			return fmt.Errorf("error inserting repository %s: %w", repo.Name, err)
		}
	}

	// Now update units to link them to repositories
	_, err = db.Exec(`
		UPDATE units SET repository_id = (
			SELECT r.id FROM repositories r 
			WHERE units.name LIKE (r.name || '-%')
			ORDER BY LENGTH(r.name) DESC
			LIMIT 1
		)
	`)
	if err != nil {
		return fmt.Errorf("error updating units repository references: %w", err)
	}

	if config.GetConfig().Verbose {
		// Report how many units were linked to repositories
		var linkedCount, totalCount int
		err := db.QueryRow("SELECT COUNT(*) FROM units WHERE repository_id IS NOT NULL").Scan(&linkedCount)
		if err != nil {
			return fmt.Errorf("error counting linked units: %w", err)
		}

		err = db.QueryRow("SELECT COUNT(*) FROM units").Scan(&totalCount)
		if err != nil {
			return fmt.Errorf("error counting total units: %w", err)
		}

		log.Printf("Migration complete: %d of %d units linked to repositories", linkedCount, totalCount)
	}

	return nil
}

// RegisterMigrationCallbacks sets up custom callbacks for migrations.
func RegisterMigrationCallbacks(_ *migrate.Migrate) {
	// Currently we can't easily hook into the migrate library's callbacks
	// without modifying the library itself.
	// Instead, we'll apply our migration separately after the schema migration completes
	log.Println("Repository migration will be applied after schema migrations")
}
