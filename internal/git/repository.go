// Package git provides git repository management functionality for quad-ops
package git

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Repository represents a Git repository to sync to a local path.
type Repository struct {
	Name       string // repository identifier
	URL        string // remote git URL
	Reference  string // git ref: branch, tag, or commit hash
	ComposeDir string // optional subdirectory in repo containing compose files
	Path       string // local path where repository will be cloned/synced
	repo       *git.Repository
}

// New creates a new Repository instance.
func New(name, url, ref, composeDir, path string) *Repository {
	return &Repository{
		Name:       name,
		URL:        url,
		Reference:  ref,
		ComposeDir: composeDir,
		Path:       path,
	}
}

// Sync clones the remote repository to the local path if it doesn't exist,
// or opens the existing repository and pulls the latest changes if it does.
// It returns an error if any Git operations fail.
// Context can be used to signal cancellation.
func (r *Repository) Sync(ctx context.Context) error {
	cloneOptions := &git.CloneOptions{URL: r.URL}

	repo, err := git.PlainClone(r.Path, false, cloneOptions)
	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			repo, err = git.PlainOpen(r.Path)
			if err != nil {
				return err
			}

			r.repo = repo
			if err := r.pullLatest(ctx); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	r.repo = repo

	if r.Reference != "" {
		return r.checkoutTarget()
	}
	return nil
}

// checkoutTarget attempts to checkout the target reference, which can be a commit hash,
// tag, or branch. It tries to checkout as a branch first, then falls back to hash checkout.
func (r *Repository) checkoutTarget() error {
	// Resolve the reference to get the actual commit hash
	hash, err := r.repo.ResolveRevision(plumbing.Revision(r.Reference))
	if err != nil {
		return fmt.Errorf("reference %q not found: %w", r.Reference, err)
	}

	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}

	// Try to checkout as a branch first to keep HEAD attached
	branchRef := plumbing.NewBranchReferenceName(r.Reference)
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Create: false,
	})
	if err == nil {
		return nil
	}

	// Fall back to checkout by hash (for tags, commits, or non-existent branches)
	return worktree.Checkout(&git.CheckoutOptions{Hash: *hash})
}

// pullLatest pulls the latest changes from the remote repository.
// It returns an error if any Git operations fail, except when the repository
// is already up to date.
func (r *Repository) pullLatest(_ context.Context) error {
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

// CheckoutRef opens an existing repository and checks out the given reference
// without fetching from the remote. Used for rollback to a known commit.
func (r *Repository) CheckoutRef(ref string) error {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}
	r.repo = repo
	r.Reference = ref
	return r.checkoutTarget()
}

// GetCurrentCommitHash returns the current HEAD commit hash.
func (r *Repository) GetCurrentCommitHash() (string, error) {
	if r.repo == nil {
		return "", fmt.Errorf("repository not initialized")
	}

	ref, err := r.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	return ref.Hash().String(), nil
}
