package git

import (
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Repository represents a Git repository with its local path, remote URL,
// and an instance of the underlying git repository.
type Repository struct {
	Path   string
	URL    string
	Target string // Can be tag, branch or commit hash
	repo   *git.Repository
}

// NewRepository creates a new Repository instance with the given local path and remote URL.
// The repository is not initialized until SyncRepository is called.
func NewRepository(path, url, target string) *Repository {
	return &Repository{
		Path:   path,
		URL:    url,
		Target: target,
	}
}

// SyncRepository clones the remote repository to the local path if it doesn't exist,
// or opens the existing repository and pulls the latest changes if it does.
// It returns an error if any Git operations fail.
func (r *Repository) SyncRepository() error {
	repo, err := git.PlainClone(r.Path, false, &git.CloneOptions{
		URL:      r.URL,
		Progress: os.Stdout,
	})

	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			repo, err = git.PlainOpen(r.Path)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	r.repo = repo

	if r.Target != "" {
		return r.checkoutTarget()
	}
	return r.pullLatest()
}

// checkoutTarget attempts to checkout the target reference, which can be a commit hash,
// tag, or branch.
func (r *Repository) checkoutTarget() error {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}

	// Try as commit hash first
	hash := plumbing.NewHash(r.Target)
	err = worktree.Checkout(&git.CheckoutOptions{
		Hash: hash,
	})
	if err == nil {
		return nil
	}

	// Try as tag/branch
	return worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(r.Target),
	})
}

// pullLatest pulls the latest changes from the remote repository.
// It returns an error if any Git operations fail, except when the repository
// is already up to date.
func (r *Repository) pullLatest() error {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
}
