// Package git provides git repository management functionality for quad-ops
package git

import (
	"log"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/trly/quad-ops/internal/config"
)

// Repository represents a Git repository with its local path, remote URL,
// and an instance of the underlying git repository.
type Repository struct {
	config.RepositoryConfig
	Path    string
	repo    *git.Repository
	verbose bool `yaml:"-"`
}

// NewGitRepository creates a new Repository instance with the given local path and remote URL.
// The repository is not initialized until SyncRepository is called.
func NewGitRepository(repository config.RepositoryConfig) *Repository {
	return &Repository{
		RepositoryConfig: repository,
		Path:             filepath.Join(config.GetConfig().RepositoryDir, repository.Name),
		verbose:          config.GetConfig().Verbose,
	}
}

// SyncRepository clones the remote repository to the local path if it doesn't exist,
// or opens the existing repository and pulls the latest changes if it does.
// It returns an error if any Git operations fail.
func (r *Repository) SyncRepository() error {
	if r.verbose {
		log.Printf("syncing repository to %s from %s", r.Path, r.URL)
	}

	repo, err := git.PlainClone(r.Path, false, &git.CloneOptions{
		URL:      r.URL,
		Progress: os.Stdout,
	})

	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			if r.verbose {
				log.Printf("repository already exists, opening from %s", r.Path)
			}

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
		if r.verbose {
			log.Printf("checking out target: %s", r.Reference)
		}
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
	if r.verbose {
		log.Printf("attempting to checkout target as commit hash: %s", r.Reference)
	}

	hash := plumbing.NewHash(r.Reference)
	err = worktree.Checkout(&git.CheckoutOptions{
		Hash: hash,
	})
	if err == nil {
		return nil
	}
	if r.verbose {
		log.Printf("attempting to checkout target as branch/tag: %s", r.Reference)
	}

	return worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(r.Reference),
	})
}

// pullLatest pulls the latest changes from the remote repository.
// It returns an error if any Git operations fail, except when the repository
// is already up to date.
func (r *Repository) pullLatest() error {
	if r.verbose {
		log.Printf("pulling latest changes from origin")
	}

	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	if r.verbose && err == git.NoErrAlreadyUpToDate {
		log.Printf("repository is already up to date")
	}
	return nil
}
