package db

import (
	"database/sql"
	"time"

	"github.com/trly/quad-ops/internal/config"
)

// Repository represents a record in the repositories table.
type Repository struct {
	ID                    int64          `db:"id"`
	Name                  string         `db:"name"`
	URL                   string         `db:"url"`
	Reference             sql.NullString `db:"reference"`
	ComposeDir            sql.NullString `db:"compose_dir"`
	CleanupPolicy         sql.NullString `db:"cleanup_policy"`
	UsePodmanDefaultNames bool           `db:"use_podman_default_names"`
	CreatedAt             time.Time      `db:"created_at"`
}

// RepositoryRepository defines the interface for repository data access operations.
type RepositoryRepository interface {
	FindAll() ([]Repository, error)
	FindByName(name string) (Repository, error)
	Create(repo *Repository) (int64, error)
	Update(repo *Repository) error
	Delete(id int64) error
	SyncFromConfig() error
}

// SQLRepositoryRepository implements RepositoryRepository interface with SQL database.
type SQLRepositoryRepository struct {
	db *sql.DB
}

// NewRepositoryRepository creates a new SQL-based repository repository.
func NewRepositoryRepository(db *sql.DB) RepositoryRepository {
	return &SQLRepositoryRepository{db: db}
}

// FindAll retrieves all repositories from the database.
func (r *SQLRepositoryRepository) FindAll() ([]Repository, error) {
	rows, err := r.db.Query("SELECT id, name, url, reference, compose_dir, cleanup_policy, use_podman_default_names, created_at FROM repositories")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var repositories []Repository
	for rows.Next() {
		var repo Repository
		if err := rows.Scan(&repo.ID, &repo.Name, &repo.URL, &repo.Reference, &repo.ComposeDir, &repo.CleanupPolicy, &repo.UsePodmanDefaultNames, &repo.CreatedAt); err != nil {
			return nil, err
		}
		repositories = append(repositories, repo)
	}
	return repositories, nil
}

// FindByName retrieves a repository by name.
func (r *SQLRepositoryRepository) FindByName(name string) (Repository, error) {
	row := r.db.QueryRow("SELECT id, name, url, reference, compose_dir, cleanup_policy, use_podman_default_names, created_at FROM repositories WHERE name = ?", name)

	var repo Repository
	err := row.Scan(&repo.ID, &repo.Name, &repo.URL, &repo.Reference, &repo.ComposeDir, &repo.CleanupPolicy, &repo.UsePodmanDefaultNames, &repo.CreatedAt)
	return repo, err
}

// Create inserts a new repository into the database.
func (r *SQLRepositoryRepository) Create(repo *Repository) (int64, error) {
	result, err := r.db.Exec(
		"INSERT INTO repositories (name, url, reference, compose_dir, cleanup_policy, use_podman_default_names) VALUES (?, ?, ?, ?, ?, ?)",
		repo.Name, repo.URL, repo.Reference, repo.ComposeDir, repo.CleanupPolicy, repo.UsePodmanDefaultNames,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// Update updates an existing repository in the database.
func (r *SQLRepositoryRepository) Update(repo *Repository) error {
	_, err := r.db.Exec(
		"UPDATE repositories SET url = ?, reference = ?, compose_dir = ?, cleanup_policy = ?, use_podman_default_names = ? WHERE id = ?",
		repo.URL, repo.Reference, repo.ComposeDir, repo.CleanupPolicy, repo.UsePodmanDefaultNames, repo.ID,
	)
	return err
}

// Delete removes a repository from the database.
func (r *SQLRepositoryRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM repositories WHERE id = ?", id)
	return err
}

// SyncFromConfig synchronizes repositories in database with those in the config file.
func (r *SQLRepositoryRepository) SyncFromConfig() error {
	// Start a transaction
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// First, fetch all existing repositories
	rows, err := tx.Query("SELECT id, name FROM repositories")
	if err != nil {
		return err
	}

	existingRepos := make(map[string]int64)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			_ = rows.Close()
			return err
		}
		existingRepos[name] = id
	}
	_ = rows.Close()

	// Track which repositories we've seen in the config
	configRepos := make(map[string]bool)

	// For each repository in config
	for _, repoConfig := range config.GetConfig().Repositories {
		configRepos[repoConfig.Name] = true

		// Create NullString values for optional fields
		var refNS, composeDirNS, cleanupNS sql.NullString

		// Only set the value as valid if the string is not empty
		if repoConfig.Reference != "" {
			refNS = sql.NullString{String: repoConfig.Reference, Valid: true}
		}
		if repoConfig.ComposeDir != "" {
			composeDirNS = sql.NullString{String: repoConfig.ComposeDir, Valid: true}
		}
		if repoConfig.Cleanup != "" {
			cleanupNS = sql.NullString{String: repoConfig.Cleanup, Valid: true}
		}

		// If repository already exists, update it
		if id, exists := existingRepos[repoConfig.Name]; exists {
			_, err = tx.Exec(
				"UPDATE repositories SET url = ?, reference = ?, compose_dir = ?, cleanup_policy = ?, use_podman_default_names = ? WHERE id = ?",
				repoConfig.URL, refNS, composeDirNS, cleanupNS, repoConfig.UsePodmanDefaultNames, id,
			)
			if err != nil {
				return err
			}
		} else {
			// Otherwise, insert a new repository
			_, err = tx.Exec(
				"INSERT INTO repositories (name, url, reference, compose_dir, cleanup_policy, use_podman_default_names) VALUES (?, ?, ?, ?, ?, ?)",
				repoConfig.Name, repoConfig.URL, refNS, composeDirNS, cleanupNS, repoConfig.UsePodmanDefaultNames,
			)
			if err != nil {
				return err
			}
		}
	}

	// Find and link orphaned units
	_, err = tx.Exec(`
		UPDATE units SET repository_id = (
			SELECT r.id FROM repositories r 
			WHERE units.name LIKE (r.name || '-%')
			ORDER BY LENGTH(r.name) DESC
			LIMIT 1
		) WHERE repository_id IS NULL
	`)
	if err != nil {
		return err
	}

	// Commit the transaction
	return tx.Commit()
}
