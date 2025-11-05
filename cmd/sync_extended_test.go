package cmd

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/testutil"
)

// TestSyncCommand_GitSyncFailure tests error handling when git sync fails.
func TestSyncCommand_GitSyncFailure(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, _ []config.Repository) ([]repository.SyncResult, error) {
				return nil, errors.New("git sync failed")
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "test-repo", URL: "https://example.com/repo.git"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git sync failed")
}

// TestSyncCommand_RepositoryNotFound tests filtering by non-existent repo name.
func TestSyncCommand_RepositoryNotFound(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "test-repo", URL: "https://example.com/repo.git"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{
		RepoName: "non-existent-repo",
	}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository not found: non-existent-repo")
}

// TestSyncCommand_SingleRepoFilter tests syncing a single repository by name.
func TestSyncCommand_SingleRepoFilter(t *testing.T) {
	var syncedRepos []config.Repository

	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				syncedRepos = repos
				return []repository.SyncResult{
					{Repository: repos[0], Success: true, Changed: false},
				}, nil
			},
		},
		Lifecycle: &MockLifecycle{},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "repo1", URL: "https://example.com/repo1.git"},
				{Name: "repo2", URL: "https://example.com/repo2.git"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{
		RepoName: "repo2",
	}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.NoError(t, err)
	require.Len(t, syncedRepos, 1)
	assert.Equal(t, "repo2", syncedRepos[0].Name)
}

// TestSyncCommand_MultipleRepositories tests syncing multiple repositories.
func TestSyncCommand_MultipleRepositories(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				results := make([]repository.SyncResult, len(repos))
				for i, repo := range repos {
					results[i] = repository.SyncResult{
						Repository: repo,
						Success:    true,
						Changed:    false,
					}
				}
				return results, nil
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "repo1", URL: "https://example.com/repo1.git"},
				{Name: "repo2", URL: "https://example.com/repo2.git"},
				{Name: "repo3", URL: "https://example.com/repo3.git"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.NoError(t, err)
}

// TestSyncCommand_RepositorySyncError tests handling of individual repo sync errors.
func TestSyncCommand_RepositorySyncError(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				return []repository.SyncResult{
					{
						Repository: repos[0],
						Success:    false,
						Error:      errors.New("failed to clone repo"),
					},
				}, nil
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "test-repo", URL: "https://example.com/repo.git"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	assert.NoError(t, err)
}

// TestSyncCommand_NoChangesSkipsProcessing tests that unchanged repos are skipped.
func TestSyncCommand_NoChangesSkipsProcessing(t *testing.T) {
	var composeProcessed bool

	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				return []repository.SyncResult{
					{Repository: repos[0], Success: true, Changed: false},
				}, nil
			},
		},
		ComposeProcessor: &MockComposeProcessor{
			ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
				composeProcessed = true
				return []service.Spec{}, nil
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "test-repo"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.NoError(t, err)
	assert.False(t, composeProcessed, "Compose should not be processed for unchanged repo")
}

// TestSyncCommand_ForceProcessesUnchangedRepos tests force flag with no compose files triggers reload.
func TestSyncCommand_ForceProcessesUnchangedRepos(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				return []repository.SyncResult{
					{Repository: repos[0], Success: true, Changed: true},
				}, nil
			},
		},
		Lifecycle: &MockLifecycle{
			ReloadFunc: func(_ context.Context) error {
				return nil
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			RepositoryDir: t.TempDir(),
			Repositories: []config.Repository{
				{Name: "test-repo"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{
		Force: true,
	}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.NoError(t, err)
}

// TestSyncCommand_ReloadOnlyCalledWithChangesOrForce tests reload behavior.
func TestSyncCommand_ReloadOnlyCalledWithChangesOrForce(t *testing.T) {
	var reloadCalled bool

	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				return []repository.SyncResult{
					{Repository: repos[0], Success: true, Changed: false},
				}, nil
			},
		},
		Lifecycle: &MockLifecycle{
			ReloadFunc: func(_ context.Context) error {
				reloadCalled = true
				return nil
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			RepositoryDir: t.TempDir(),
			Repositories: []config.Repository{
				{Name: "test-repo"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.NoError(t, err)
	assert.False(t, reloadCalled, "Reload should not be called without changes or force")
}

// TestSyncCommand_RestartChangedServices tests restarting services when anyChanges is true.
func TestSyncCommand_RestartChangedServices(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				return []repository.SyncResult{
					{Repository: repos[0], Success: true, Changed: true},
				}, nil
			},
		},
		Lifecycle: &MockLifecycle{
			ReloadFunc: func(_ context.Context) error { return nil },
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			RepositoryDir: t.TempDir(),
			Repositories: []config.Repository{
				{Name: "test-repo"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{
		Force: true,
	}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.NoError(t, err)
}

// TestSyncCommand_RestartErrors tests handling when no services need restart (logs errors gracefully).
func TestSyncCommand_RestartErrors(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				return []repository.SyncResult{
					{Repository: repos[0], Success: true, Changed: false},
				}, nil
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			RepositoryDir: t.TempDir(),
			Repositories: []config.Repository{
				{Name: "test-repo"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	assert.NoError(t, err)
}

// TestSyncCommand_TrackChangedServices tests the trackChangedServices method.
func TestSyncCommand_TrackChangedServices(t *testing.T) {
	tests := []struct {
		name             string
		changedPaths     []string
		serviceChanges   map[string]platform.ChangeStatus
		force            bool
		expectedServices []string
	}{
		{
			name:         "service with changed artifact",
			changedPaths: []string{"/path1"},
			serviceChanges: map[string]platform.ChangeStatus{
				"service1": {ArtifactPaths: []string{"/path1"}},
			},
			force:            false,
			expectedServices: []string{"service1"},
		},
		{
			name:         "service without changed artifact",
			changedPaths: []string{"/path1"},
			serviceChanges: map[string]platform.ChangeStatus{
				"service1": {ArtifactPaths: []string{"/path2"}},
			},
			force:            false,
			expectedServices: []string{},
		},
		{
			name:         "force restarts all services",
			changedPaths: []string{},
			serviceChanges: map[string]platform.ChangeStatus{
				"service1": {ArtifactPaths: []string{"/path1"}},
				"service2": {ArtifactPaths: []string{"/path2"}},
			},
			force:            true,
			expectedServices: []string{"service1", "service2"},
		},
		{
			name:         "multiple services with changes",
			changedPaths: []string{"/path1", "/path2"},
			serviceChanges: map[string]platform.ChangeStatus{
				"service1": {ArtifactPaths: []string{"/path1"}},
				"service2": {ArtifactPaths: []string{"/path2"}},
				"service3": {ArtifactPaths: []string{"/path3"}},
			},
			force:            false,
			expectedServices: []string{"service1", "service2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewSyncCommand()
			servicesToRestart := make(map[string]bool)

			cmd.trackChangedServices(tt.changedPaths, tt.serviceChanges, tt.force, servicesToRestart)

			assert.Len(t, servicesToRestart, len(tt.expectedServices))
			for _, svc := range tt.expectedServices {
				assert.True(t, servicesToRestart[svc], "Expected %s to be in restart list", svc)
			}
		})
	}
}
