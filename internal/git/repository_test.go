package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/testutil"
)

// setupTest creates a temporary directory and returns config provider for testing.
func setupTest(t *testing.T) (string, config.Provider, func()) {
	tmpDir, cleanup := testutil.SetupTempDir(t)
	configProvider := testutil.NewMockConfig(t, testutil.WithRepositoryDir(tmpDir))
	return tmpDir, configProvider, cleanup
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

func TestNewRepository(t *testing.T) {
	tmpDir, configProv, cleanup := setupTest(t)
	defer cleanup()

	testRepo := config.Repository{
		Name:      "test-repo",
		URL:       "https://github.com/test/repo.git",
		Reference: "main",
	}

	repo := NewGitRepository(testRepo, configProv)

	require.Equal(t, testRepo.URL, repo.URL)
	require.Equal(t, filepath.Join(tmpDir, testRepo.Name), repo.Path)
	require.Equal(t, testRepo.Reference, repo.Reference)
}

func TestSyncRepository(t *testing.T) {
	_, configProv, cleanup := setupTest(t)
	defer cleanup()

	testRepo := config.Repository{
		Name:      "test-repo",
		URL:       "https://github.com/test/repo.git",
		Reference: "main",
	}

	repo := NewGitRepository(testRepo, configProv)

	// Test invalid repository URL
	err := repo.SyncRepository()
	require.Error(t, err, "Expected error for invalid repository URL")
}

func TestSyncRepositoryAlreadyExists(t *testing.T) {
	tmpDir, configProv, cleanup := setupTest(t)
	defer cleanup()

	testRepo := config.Repository{
		Name:      "test-repo",
		URL:       "https://github.com/test/repo.git",
		Reference: "main",
	}

	repo := NewGitRepository(testRepo, configProv)
	require.Equal(t, "main", repo.Reference)

	// Create the repository directory to simulate existing repo
	repoDir := filepath.Join(tmpDir, testRepo.Name)
	require.NoError(t, os.MkdirAll(repoDir, 0700))

	// Create a .git directory to simulate an existing git repository
	gitDir := filepath.Join(repoDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0700))

	// Test that SyncRepository handles existing repository case
	// This should fail because we've created a fake .git directory without proper git structure
	err := repo.SyncRepository()
	require.Error(t, err, "Expected error for invalid existing repository structure")
}

func TestCheckoutTargetWithLocalRepo(t *testing.T) {
	tmpDir, configProv, cleanup := setupTest(t)
	defer cleanup()

	// Create a real local git repository for testing
	localRepoDir := filepath.Join(tmpDir, "source-repo")
	_, commitHash := createTestRepo(t, localRepoDir)

	testRepo := config.Repository{
		Name:      "test-repo",
		URL:       localRepoDir,
		Reference: commitHash,
	}

	repo := NewGitRepository(testRepo, configProv)

	// Test syncing and checking out specific commit
	err := repo.SyncRepository()
	require.NoError(t, err)

	// Verify the repository was synced and the correct commit was checked out
	require.NotNil(t, repo.repo, "Repository should be initialized after sync")

	// Get current HEAD commit
	ref, err := repo.repo.Head()
	require.NoError(t, err)
	require.Equal(t, commitHash, ref.Hash().String())
}

func TestPullLatest(t *testing.T) {
	tmpDir, configProv, cleanup := setupTest(t)
	defer cleanup()

	// Create a "remote" repository
	remoteRepoDir := filepath.Join(tmpDir, "remote-repo")
	remoteRepo, _ := createTestRepo(t, remoteRepoDir)

	testRepo := config.Repository{
		Name: "test-repo",
		URL:  remoteRepoDir,
	}

	repo := NewGitRepository(testRepo, configProv)

	// Initial sync to clone the repository
	err := repo.SyncRepository()
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
	err = repo.pullLatest()
	require.NoError(t, err)

	// Verify the new commit was pulled
	ref, err := repo.repo.Head()
	require.NoError(t, err)
	require.Equal(t, newCommit.String(), ref.Hash().String())

	// Test pullLatest again - should be already up to date
	err = repo.pullLatest()
	require.NoError(t, err)
}

func TestSyncRepositoryExistingRepoFlow(t *testing.T) {
	tmpDir, configProv, cleanup := setupTest(t)
	defer cleanup()

	// Create a "remote" repository
	remoteRepoDir := filepath.Join(tmpDir, "remote-repo")
	remoteRepo, firstCommitHash := createTestRepo(t, remoteRepoDir)

	testRepo := config.Repository{
		Name: "test-repo",
		URL:  remoteRepoDir,
	}

	repo := NewGitRepository(testRepo, configProv)

	// First sync - should clone the repository
	err := repo.SyncRepository()
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
	repo2 := NewGitRepository(testRepo, configProv)

	// Second sync - should open existing repo and pull latest changes
	// This tests the git.ErrRepositoryAlreadyExists path (lines 46-56)
	err = repo2.SyncRepository()
	require.NoError(t, err)

	// Verify the second commit was pulled
	ref2, err := repo2.repo.Head()
	require.NoError(t, err)
	require.Equal(t, secondCommit.String(), ref2.Hash().String())
}

func TestCheckoutTargetBranchFallback(t *testing.T) {
	tmpDir, configProv, cleanup := setupTest(t)
	defer cleanup()

	// Create a "remote" repository with a branch
	remoteRepoDir := filepath.Join(tmpDir, "remote-repo")
	_, commitHash := createTestRepo(t, remoteRepoDir)

	testRepo := config.Repository{
		Name:      "test-repo",
		URL:       remoteRepoDir,
		Reference: "main", // Use branch name instead of commit hash
	}

	repo := NewGitRepository(testRepo, configProv)

	// Sync repository - this will trigger checkoutTarget with branch name
	// First it will try to checkout "main" as a commit hash (which will fail)
	// Then it will fall back to checking out as a branch (lines 87-91)
	err := repo.SyncRepository()
	require.NoError(t, err)

	// Verify the correct commit was checked out
	ref, err := repo.repo.Head()
	require.NoError(t, err)
	require.Equal(t, commitHash, ref.Hash().String())

	// Verify we're on a branch (git uses "master" by default, not "main")
	require.True(t, ref.Name().IsBranch(), "Expected to be on a branch")
	require.Contains(t, []string{"main", "master"}, ref.Name().Short())
}

func TestCheckoutTargetTag(t *testing.T) {
	tmpDir, configProv, cleanup := setupTest(t)
	defer cleanup()

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

	testRepo := config.Repository{
		Name:      "test-repo",
		URL:       remoteRepoDir,
		Reference: tagName, // Use tag name
	}

	repo := NewGitRepository(testRepo, configProv)

	// Sync repository - this will trigger checkoutTarget with tag name
	// First it will try to checkout the tag as a commit hash (which will fail)
	// Then it will fall back to checking out as a branch/tag (lines 87-91)
	err = repo.SyncRepository()
	require.NoError(t, err)

	// Verify the correct commit was checked out
	ref, err := repo.repo.Head()
	require.NoError(t, err)
	require.Equal(t, commitHash, ref.Hash().String())
}

func TestCheckoutTargetForceBranchFallback(t *testing.T) {
	tmpDir, configProv, cleanup := setupTest(t)
	defer cleanup()

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

	testRepo := config.Repository{
		Name:      "test-repo",
		URL:       remoteRepoDir,
		Reference: "feature", // Use branch name that is NOT a valid commit hash
	}

	repo := NewGitRepository(testRepo, configProv)

	// Sync repository - this will trigger checkoutTarget with branch name
	// "feature" is not a valid commit hash, so it will fail the first checkout
	// and fall back to the branch checkout (lines 87-91)
	err = repo.SyncRepository()
	require.NoError(t, err)

	// Verify the feature branch commit was checked out
	ref, err := repo.repo.Head()
	require.NoError(t, err)
	require.Equal(t, featureCommit.String(), ref.Hash().String())

	// Verify we're on the feature branch
	require.True(t, ref.Name().IsBranch(), "Expected to be on a branch")
	require.Equal(t, "feature", ref.Name().Short())
}

func TestNewGitRepository(t *testing.T) {
	configProvider := testutil.NewMockConfig(t,
		testutil.WithRepositoryDir("/test/custom/repo/dir"),
		testutil.WithVerbose(true))

	repo := config.Repository{
		Name: "test-repo",
		URL:  "https://example.com/repo.git",
	}

	gitRepo := NewGitRepository(repo, configProvider)

	require.Equal(t, "test-repo", gitRepo.Name)
	require.Equal(t, "https://example.com/repo.git", gitRepo.URL)
	require.Equal(t, "/test/custom/repo/dir/test-repo", gitRepo.Path)
	require.True(t, gitRepo.verbose)
	require.NotNil(t, gitRepo.logger)
}
