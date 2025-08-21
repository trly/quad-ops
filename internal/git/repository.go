// Package git provides git repository management functionality for quad-ops
package git

import (
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/log"
)

// Repository represents a Git repository with its local path, remote URL,
// and an instance of the underlying git repository.
type Repository struct {
	config.Repository
	Path    string
	repo    *git.Repository
	verbose bool `yaml:"-"`
	logger  log.Logger
}

// NewGitRepository creates a new Repository instance with explicit config provider.
func NewGitRepository(repository config.Repository, configProvider config.Provider) *Repository {
	cfg := configProvider.GetConfig()
	return &Repository{
		Repository: repository,
		Path:       filepath.Join(cfg.RepositoryDir, repository.Name),
		verbose:    cfg.Verbose,
		logger:     log.NewLogger(cfg.Verbose),
	}
}

// NewGitRepositoryWithLogger creates a new Repository instance with explicit dependencies.
func NewGitRepositoryWithLogger(repository config.Repository, configProvider config.Provider, logger log.Logger) *Repository {
	cfg := configProvider.GetConfig()
	return &Repository{
		Repository: repository,
		Path:       filepath.Join(cfg.RepositoryDir, repository.Name),
		verbose:    cfg.Verbose,
		logger:     logger,
	}
}

// SyncRepository clones the remote repository to the local path if it doesn't exist,
// or opens the existing repository and pulls the latest changes if it does.
// It returns an error if any Git operations fail.
func (r *Repository) SyncRepository() error {
	r.logger.Debug("Syncing repository", "path", filepath.Base(r.Path))

	cloneOptions := &git.CloneOptions{URL: r.URL}
	if r.verbose {
		cloneOptions.Progress = os.Stdout
	}

	repo, err := git.PlainClone(r.Path, false, cloneOptions)

	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			r.logger.Debug("Repository already exists, opening", "path", r.Path)

			repo, err = git.PlainOpen(r.Path)
			if err != nil {
				return err
			}

			r.repo = repo
			if err := r.pullLatest(); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	r.repo = repo

	if r.Reference != "" {
		r.logger.Debug("Checking out target", "ref", r.Reference)
		return r.checkoutTarget()
	}
	return nil
}

// checkoutTarget attempts to checkout the target reference, which can be a commit hash,
// tag, or branch.
func (r *Repository) checkoutTarget() error {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	r.logger.Debug("Attempting to checkout target as commit hash", "hash", r.Reference)

	hash := plumbing.NewHash(r.Reference)
	err = worktree.Checkout(&git.CheckoutOptions{
		Hash: hash,
	})
	if err == nil {
		return nil
	}
	r.logger.Debug("Attempting to checkout target as branch/tag", "ref", r.Reference)

	return worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(r.Reference),
	})
}

// pullLatest pulls the latest changes from the remote repository.
// It returns an error if any Git operations fail, except when the repository
// is already up to date.
func (r *Repository) pullLatest() error {
	r.logger.Debug("Pulling latest changes from origin")

	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	if err == git.NoErrAlreadyUpToDate {
		r.logger.Debug("Repository is already up to date")
	}
	return nil
}
