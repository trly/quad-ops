package git

import (
	"log"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Repository represents a Git repository with its local path, remote URL,
// and an instance of the underlying git repository.
type Repository struct {
	Path    string
	URL     string
	Target  string // Can be tag, branch or commit hash
	repo    *git.Repository
	verbose bool
}

// NewRepository creates a new Repository instance with the given local path and remote URL.
// The repository is not initialized until SyncRepository is called.
func NewRepository(path, url, target string, verbose bool) *Repository {
	return &Repository{

		Path:    path,
		URL:     url,
		Target:  target,
		verbose: verbose,
	}
}

// SyncRepository clones the remote repository to the local path if it doesn't exist,
// or opens the existing repository and pulls the latest changes if it does.
// It returns an error if any Git operations fail.
func (r *Repository) SyncRepository() error {
	if r.verbose {
		log.Printf("Syncing repository at %s from %s", r.Path, r.URL)
	}

	repo, err := git.PlainClone(r.Path, false, &git.CloneOptions{
		URL:      r.URL,
		Progress: os.Stdout,
	})

	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			if r.verbose {
				log.Printf("Repository already exists, opening from %s", r.Path)
			}
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
		if r.verbose {
			log.Printf("Checking out target: %s", r.Target)
		}
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
	if r.verbose {
		log.Printf("Attempting to checkout target as commit hash: %s", r.Target)
	}

	hash := plumbing.NewHash(r.Target)
	err = worktree.Checkout(&git.CheckoutOptions{
		Hash: hash,
	})
	if err == nil {
		return nil
	}
	if r.verbose {
		log.Printf("Attempting to checkout target as branch/tag: %s", r.Target)
	}

	return worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(r.Target),
	})
}

// pullLatest pulls the latest changes from the remote repository.
// It returns an error if any Git operations fail, except when the repository
// is already up to date.
func (r *Repository) pullLatest() error {
	if r.verbose {
		log.Printf("Pulling latest changes from origin")
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
		log.Printf("Repository is already up to date")
	}
	return nil
}
