package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
)

// setupTest creates a temporary directory for testing.
func setupTest(t *testing.T) string {
	return t.TempDir()
}

// createTestRepo creates a local git repository with an initial commit.
func createTestRepo(t *testing.T, repoDir string) (*git.Repository, string) {
	repo, err := git.PlainInit(repoDir, false)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	testFile := filepath.Join(repoDir, "test.txt")
	err = os.WriteFile(testFile, []byte("initial content"), 0600)
	require.NoError(t, err)

	_, err = worktree.Add("test.txt")
	require.NoError(t, err)

	commit, err := worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	return repo, commit.String()
}

func TestNew(t *testing.T) {
	tmpDir := setupTest(t)

	repoPath := filepath.Join(tmpDir, "test-repo")
	repo := New("test-repo", "https://github.com/test/repo.git", "main", "", repoPath)

	require.Equal(t, "test-repo", repo.Name)
	require.Equal(t, "https://github.com/test/repo.git", repo.URL)
	require.Equal(t, "main", repo.Reference)
	require.Equal(t, "", repo.ComposeDir)
	require.Equal(t, repoPath, repo.Path)
}

func TestSyncRepository(t *testing.T) {
	tmpDir := setupTest(t)

	repoPath := filepath.Join(tmpDir, "test-repo")
	repo := New("test-repo", "https://github.com/test/repo.git", "main", "", repoPath)

	// Test invalid repository URL
	err := repo.Sync(context.Background())
	require.Error(t, err, "Expected error for invalid repository URL")
}

func TestSyncRepositoryAlreadyExists(t *testing.T) {
	tmpDir := setupTest(t)

	repoPath := filepath.Join(tmpDir, "test-repo")
	repo := New("test-repo", "https://github.com/test/repo.git", "main", "", repoPath)

	// Create the repository directory to simulate existing repo
	require.NoError(t, os.MkdirAll(repoPath, 0700))

	// Create a .git directory to simulate an existing git repository
	gitDir := filepath.Join(repoPath, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0700))

	// Test that Sync handles existing repository case
	// This should fail because we've created a fake .git directory without proper git structure
	err := repo.Sync(context.Background())
	require.Error(t, err, "Expected error for invalid existing repository structure")
}

func TestCheckoutTargetWithLocalRepo(t *testing.T) {
	tmpDir := setupTest(t)

	// Create a real local git repository for testing
	localRepoDir := filepath.Join(tmpDir, "source-repo")
	_, commitHash := createTestRepo(t, localRepoDir)

	repoPath := filepath.Join(tmpDir, "test-repo")
	repo := New("test-repo", localRepoDir, commitHash, "", repoPath)

	// Test syncing and checking out specific commit
	err := repo.Sync(context.Background())
	require.NoError(t, err)

	// Verify the repository was synced and the correct commit was checked out
	require.NotNil(t, repo.repo, "Repository should be initialized after sync")

	// Get current HEAD commit
	ref, err := repo.repo.Head()
	require.NoError(t, err)
	require.Equal(t, commitHash, ref.Hash().String())
}

func TestPullLatest(t *testing.T) {
	tmpDir := setupTest(t)

	// Create a "remote" repository
	remoteRepoDir := filepath.Join(tmpDir, "remote-repo")
	remoteRepo, _ := createTestRepo(t, remoteRepoDir)

	repoPath := filepath.Join(tmpDir, "test-repo")
	repo := New("test-repo", remoteRepoDir, "", "", repoPath)

	// Initial sync to clone the repository
	err := repo.Sync(context.Background())
	require.NoError(t, err)

	// Create another commit in the remote repository
	remoteWorktree, err := remoteRepo.Worktree()
	require.NoError(t, err)

	testFile := filepath.Join(remoteRepoDir, "test.txt")
	err = os.WriteFile(testFile, []byte("updated content"), 0600)
	require.NoError(t, err)

	_, err = remoteWorktree.Add("test.txt")
	require.NoError(t, err)

	newCommit, err := remoteWorktree.Commit("second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Test pullLatest - should pull the new commit
	err = repo.pullLatest(context.Background())
	require.NoError(t, err)

	// Verify the new commit was pulled
	ref, err := repo.repo.Head()
	require.NoError(t, err)
	require.Equal(t, newCommit.String(), ref.Hash().String())

	// Test pullLatest again - should be already up to date
	err = repo.pullLatest(context.Background())
	require.NoError(t, err)
}

func TestSyncRepositoryExistingRepoFlow(t *testing.T) {
	tmpDir := setupTest(t)

	// Create a "remote" repository
	remoteRepoDir := filepath.Join(tmpDir, "remote-repo")
	remoteRepo, firstCommitHash := createTestRepo(t, remoteRepoDir)

	repoPath := filepath.Join(tmpDir, "test-repo")
	repo := New("test-repo", remoteRepoDir, "", "", repoPath)

	// First sync - should clone the repository
	err := repo.Sync(context.Background())
	require.NoError(t, err)

	// Verify first commit is checked out
	ref, err := repo.repo.Head()
	require.NoError(t, err)
	require.Equal(t, firstCommitHash, ref.Hash().String())

	// Add another commit to remote
	remoteWorktree, err := remoteRepo.Worktree()
	require.NoError(t, err)

	testFile := filepath.Join(remoteRepoDir, "test.txt")
	err = os.WriteFile(testFile, []byte("updated content"), 0600)
	require.NoError(t, err)

	_, err = remoteWorktree.Add("test.txt")
	require.NoError(t, err)

	secondCommit, err := remoteWorktree.Commit("second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Create a new Repository instance to simulate running the command again
	repo2 := New("test-repo", remoteRepoDir, "", "", repoPath)

	// Second sync - should open existing repo and pull latest changes
	err = repo2.Sync(context.Background())
	require.NoError(t, err)

	// Verify the second commit was pulled
	ref2, err := repo2.repo.Head()
	require.NoError(t, err)
	require.Equal(t, secondCommit.String(), ref2.Hash().String())
}

func TestCheckoutTargetBranchFallback(t *testing.T) {
	tmpDir := setupTest(t)

	// Create a "remote" repository with a branch
	remoteRepoDir := filepath.Join(tmpDir, "remote-repo")
	remoteRepo, commitHash := createTestRepo(t, remoteRepoDir)

	// Create "main" branch (repo initializes with "master" by default)
	remoteWorktree, err := remoteRepo.Worktree()
	require.NoError(t, err)

	mainBranchRef := plumbing.NewBranchReferenceName("main")
	err = remoteWorktree.Checkout(&git.CheckoutOptions{
		Branch: mainBranchRef,
		Create: true,
	})
	require.NoError(t, err)

	repoPath := filepath.Join(tmpDir, "test-repo")
	repo := New("test-repo", remoteRepoDir, "main", "", repoPath)

	// Sync repository - this will trigger checkoutTarget with branch name
	// ResolveRevision will resolve "main" to the commit hash, then checkout as branch
	err = repo.Sync(context.Background())
	require.NoError(t, err)

	// Verify the correct commit was checked out
	ref, err := repo.repo.Head()
	require.NoError(t, err)
	require.Equal(t, commitHash, ref.Hash().String())

	// Verify we're on the main branch
	require.True(t, ref.Name().IsBranch(), "Expected to be on a branch")
	require.Equal(t, "main", ref.Name().Short())
}

func TestCheckoutTargetTag(t *testing.T) {
	tmpDir := setupTest(t)

	// Create a "remote" repository with a tag
	remoteRepoDir := filepath.Join(tmpDir, "remote-repo")
	remoteRepo, commitHash := createTestRepo(t, remoteRepoDir)

	// Create a tag pointing to the commit
	tagName := "v1.0.0"
	_, err := remoteRepo.CreateTag(tagName, plumbing.NewHash(commitHash), &git.CreateTagOptions{
		Message: "Release v1.0.0",
		Tagger: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	repoPath := filepath.Join(tmpDir, "test-repo")
	repo := New("test-repo", remoteRepoDir, tagName, "", repoPath)

	// Sync repository - this will trigger checkoutTarget with tag name
	// ResolveRevision will resolve the tag to the commit hash, then checkout
	err = repo.Sync(context.Background())
	require.NoError(t, err)

	// Verify the correct commit was checked out
	ref, err := repo.repo.Head()
	require.NoError(t, err)
	require.Equal(t, commitHash, ref.Hash().String())
}

func TestCheckoutTargetForceBranchFallback(t *testing.T) {
	tmpDir := setupTest(t)

	// Create a "remote" repository
	remoteRepoDir := filepath.Join(tmpDir, "remote-repo")
	remoteRepo, _ := createTestRepo(t, remoteRepoDir)

	// Create a new branch with a different commit
	remoteWorktree, err := remoteRepo.Worktree()
	require.NoError(t, err)

	// Create feature branch
	featureBranchRef := plumbing.NewBranchReferenceName("feature")
	err = remoteWorktree.Checkout(&git.CheckoutOptions{
		Branch: featureBranchRef,
		Create: true,
	})
	require.NoError(t, err)

	// Make a commit on the feature branch
	testFile := filepath.Join(remoteRepoDir, "feature.txt")
	err = os.WriteFile(testFile, []byte("feature content"), 0600)
	require.NoError(t, err)

	_, err = remoteWorktree.Add("feature.txt")
	require.NoError(t, err)

	featureCommit, err := remoteWorktree.Commit("feature commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	repoPath := filepath.Join(tmpDir, "test-repo")
	repo := New("test-repo", remoteRepoDir, "feature", "", repoPath)

	// Sync repository - this will trigger checkoutTarget with branch name
	// ResolveRevision will resolve "feature" to the commit hash, then checkout
	err = repo.Sync(context.Background())
	require.NoError(t, err)

	// Verify the feature branch commit was checked out
	ref, err := repo.repo.Head()
	require.NoError(t, err)
	require.Equal(t, featureCommit.String(), ref.Hash().String())

	// Verify we're on the feature branch
	require.True(t, ref.Name().IsBranch(), "Expected to be on a branch")
	require.Equal(t, "feature", ref.Name().Short())
}

func TestCheckoutRef(t *testing.T) {
	tmpDir := setupTest(t)

	// Create a "remote" repository with two commits
	remoteRepoDir := filepath.Join(tmpDir, "remote-repo")
	remoteRepo, firstCommitHash := createTestRepo(t, remoteRepoDir)

	remoteWorktree, err := remoteRepo.Worktree()
	require.NoError(t, err)

	testFile := filepath.Join(remoteRepoDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("updated"), 0o600))
	_, err = remoteWorktree.Add("test.txt")
	require.NoError(t, err)

	_, err = remoteWorktree.Commit("second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Clone the repo at latest
	repoPath := filepath.Join(tmpDir, "test-repo")
	repo := New("test-repo", remoteRepoDir, "", "", repoPath)
	require.NoError(t, repo.Sync(context.Background()))

	// CheckoutRef to the first commit without pulling
	require.NoError(t, repo.CheckoutRef(firstCommitHash))

	hash, err := repo.GetCurrentCommitHash()
	require.NoError(t, err)
	require.Equal(t, firstCommitHash, hash)
}

func TestCheckoutRefNonExistentRepo(t *testing.T) {
	repo := New("missing", "", "", "", "/nonexistent/path")
	err := repo.CheckoutRef("abc123")
	require.Error(t, err)
}

func TestNewWithAllFields(t *testing.T) {
	repo := New("test-repo", "https://example.com/repo.git", "main", "examples", "/test/path")

	require.Equal(t, "test-repo", repo.Name)
	require.Equal(t, "https://example.com/repo.git", repo.URL)
	require.Equal(t, "main", repo.Reference)
	require.Equal(t, "examples", repo.ComposeDir)
	require.Equal(t, "/test/path", repo.Path)
}
